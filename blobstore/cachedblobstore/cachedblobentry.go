package cachedblobstore

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/blobstore"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
	"github.com/nyaxt/otaru/util/cancellable"
)

type cacheEntryState int

const (
	cacheEntryUninitialized cacheEntryState = iota
	cacheEntryInvalidating
	cacheEntryInvalidateFailed
	cacheEntryClean
	cacheEntryDirty
	cacheEntryClosed
)

func (s cacheEntryState) IsActive() bool {
	return s == cacheEntryInvalidating || s == cacheEntryClean || s == cacheEntryDirty
}

func (s cacheEntryState) String() string {
	switch s {
	case cacheEntryUninitialized:
		return "Uninitialized"
	case cacheEntryInvalidating:
		return "Invalidating"
	case cacheEntryInvalidateFailed:
		return "InvalidateFailed"
	case cacheEntryClean:
		return "Clean"
	case cacheEntryDirty:
		return "Dirty"
	case cacheEntryClosed:
		return "Closed"
	default:
		return "<unknown>"
	}
}

type CachedBlobEntry struct {
	mu sync.Mutex

	cbs      *CachedBlobStore
	blobpath string
	cachebh  blobstore.BlobHandle

	state cacheEntryState

	bloblen              int64
	validlen             int64
	invalidationProgress *sync.Cond

	lastUsed  time.Time
	lastWrite time.Time
	lastSync  time.Time
	syncCount int

	handles map[*CachedBlobHandle]struct{}
}

const invalidateBlockSize int = 32 * 1024

type InvalidateFailedErr struct {
	Blobpath string
}

func (e InvalidateFailedErr) Error() string {
	return fmt.Sprintf("InvalidationFailed for blob \"%s\"", e.Blobpath)
}

func IsInvalidateFailedErr(e error) bool {
	if e == nil {
		return false
	}

	_, ok := e.(InvalidateFailedErr)
	return ok
}

func (be *CachedBlobEntry) waitUntilInvalidateAtLeast(requiredLen int64) error {
	logger.Infof(mylog, "Waiting for cache to be fulfilled: reqlen: %d, validlen: %d", requiredLen, be.validlen)
	for {
		switch be.state {
		case cacheEntryInvalidating:
			break

		case cacheEntryInvalidateFailed:
			return InvalidateFailedErr{be.blobpath}

		case cacheEntryClean, cacheEntryDirty:
			return nil

		case cacheEntryUninitialized, cacheEntryClosed:
			logger.Panicf(mylog, "Invalid state while in waitUntilInvalidateAtLeast! %+v", be)
		}

		if be.validlen >= requiredLen {
			return nil
		}

		be.invalidationProgress.Wait()
	}
}

func (be *CachedBlobEntry) waitUntilInvalidateDone() error {
	return be.waitUntilInvalidateAtLeast(be.bloblen)
}

// invalidateCacheBlob fetches new version of the blob from backendbs.
// This func should be only called from be.invalidateCache()
func (be *CachedBlobEntry) invalidateCacheBlob(ctx context.Context, cbs *CachedBlobStore) error {
	blobpath := be.blobpath

	backendr, err := cbs.backendbs.OpenReader(blobpath)
	if err != nil {
		return fmt.Errorf("Failed to open backend blob for cache invalidation: %v", err)
	}
	defer func() {
		if err := backendr.Close(); err != nil {
			logger.Criticalf(mylog, "Failed to close backend blob reader for cache invalidation: %v", err)
		}
	}()

	bs, ok := cbs.cachebs.(blobstore.BlobStore)
	if !ok {
		return fmt.Errorf("FIXME: only cachebs supporting OpenWriter is currently supported")
	}

	cachew, err := bs.OpenWriter(blobpath)
	defer func() {
		if err := cachew.Close(); err != nil {
			logger.Criticalf(mylog, "Failed to close cache blob writer for cache invalidation: %v", err)
		}
	}()

	buf := make([]byte, invalidateBlockSize)
	for {
		nr, er := cancellable.Read(ctx, backendr, buf)
		if nr > 0 {
			nw, ew := cachew.Write(buf[:nr])
			if nw > 0 {
				be.mu.Lock()
				be.validlen += int64(nw)
				be.invalidationProgress.Broadcast()
				be.mu.Unlock()
			}
			if ew != nil {
				return fmt.Errorf("Failed to write backend blob content to cache: %v", ew)
			}
			if nw != nr {
				return fmt.Errorf("Failed to write backend blob content to cache: %v", io.ErrShortWrite)
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return fmt.Errorf("Failed to read backend blob content: %v", er)
		}
	}

	// FIXME: check integrity here?

	return nil
}

func (be *CachedBlobEntry) invalidateCache(ctx context.Context, cbs *CachedBlobStore) error {
	if err := be.invalidateCacheBlob(ctx, cbs); err != nil {
		be.mu.Lock()
		be.validlen = 0
		be.state = cacheEntryInvalidateFailed
		be.invalidationProgress.Broadcast()
		be.mu.Unlock()
		return err
	}
	be.mu.Lock()
	be.state = cacheEntryClean
	be.mu.Unlock()
	return nil
}

func (be *CachedBlobEntry) initializeWithLock(cbs *CachedBlobStore) error {
	cachebh, err := cbs.cachebs.Open(be.blobpath, fl.O_RDWRCREATE)
	if err != nil {
		be.closeWithLock(abandonAndClose)
		return fmt.Errorf("Failed to open cache blob: %v", err)
	}
	cachever, err := cbs.queryVersion(&blobstore.OffsetReader{cachebh, 0})
	if err != nil {
		be.closeWithLock(abandonAndClose)
		return fmt.Errorf("Failed to query cached blob ver: %v", err)
	}
	backendver, err := cbs.bever.Query(be.blobpath)
	if err != nil {
		be.closeWithLock(abandonAndClose)
		return err
	}

	be.cbs = cbs
	be.cachebh = cachebh
	be.handles = make(map[*CachedBlobHandle]struct{})

	if cachever > backendver {
		logger.Warningf(mylog, "FIXME: cache is newer than backend when open")
		be.state = cacheEntryDirty
		be.bloblen = cachebh.Size()
		be.validlen = be.bloblen
	} else if cachever == backendver {
		be.state = cacheEntryClean
		be.bloblen = cachebh.Size()
		be.validlen = be.bloblen
	} else {
		blobsizer := cbs.backendbs.(blobstore.BlobSizer)
		be.bloblen, err = blobsizer.BlobSize(be.blobpath)
		if err != nil {
			be.closeWithLock(abandonAndClose)
			return fmt.Errorf("Failed to query backend blobsize: %v", err)
		}
		be.state = cacheEntryInvalidating
		be.validlen = 0

		cbs.s.RunImmediately(&InvalidateCacheTask{cbs, be}, nil)
	}
	if be.state == cacheEntryUninitialized {
		panic("be.state should be set above")
	}

	return nil
}

func (be *CachedBlobEntry) OpenHandle(cbs *CachedBlobStore, flags int) (*CachedBlobHandle, error) {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.state == cacheEntryUninitialized {
		if err := be.initializeWithLock(cbs); err != nil {
			return nil, err
		}
	}

	be.lastUsed = time.Now()

	bh := &CachedBlobHandle{be, flags}
	be.handles[bh] = struct{}{}

	return bh, nil
}

func (be *CachedBlobEntry) CloseHandle(bh *CachedBlobHandle) {
	be.mu.Lock()
	defer be.mu.Unlock()

	delete(be.handles, bh)
}

func (be *CachedBlobEntry) PRead(p []byte, offset int64) error {
	// FIXME: may be we should allow stale reads w/o lock
	be.mu.Lock()
	defer be.mu.Unlock()

	be.lastUsed = time.Now()

	requiredLen := util.Int64Min(offset+int64(len(p)), be.bloblen)
	if err := be.waitUntilInvalidateAtLeast(requiredLen); err != nil {
		return err
	}

	return be.cachebh.PRead(p, offset)
}

func (be *CachedBlobEntry) LastWrite() time.Time { return be.lastWrite }
func (be *CachedBlobEntry) LastSync() time.Time  { return be.lastSync }

func (be *CachedBlobEntry) markDirtyWithLock() {
	now := time.Now()
	be.lastUsed = now
	be.lastWrite = now

	if be.state == cacheEntryDirty {
		return
	}
	if be.state != cacheEntryClean {
		logger.Panicf(mylog, "markDirty called from unexpected state: %+v", be.infoWithLock())
	}
	be.state = cacheEntryDirty

	if be.lastSync.IsZero() {
		be.lastSync = time.Now()
	}
}

func (be *CachedBlobEntry) PWrite(p []byte, offset int64) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	// Avoid any write when in invalidating state.
	// FIXME: maybe allow when offset+len(p) < be.validlen
	if err := be.waitUntilInvalidateDone(); err != nil {
		return err
	}

	if len(p) == 0 {
		return nil
	}
	be.markDirtyWithLock()
	if err := be.cachebh.PWrite(p, offset); err != nil {
		return err
	}

	right := offset + int64(len(p))
	if right > be.bloblen {
		be.bloblen = right
		be.validlen = right
	}
	return nil
}

func (be *CachedBlobEntry) Size() int64 {
	return be.bloblen
}

func (be *CachedBlobEntry) Truncate(newsize int64) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	// Avoid truncate when in invalidating state.
	// FIXME: maybe allow if newsize < be.validlen
	if err := be.waitUntilInvalidateDone(); err != nil {
		logger.Infof(mylog, "Waiting for cache to be fully invalidated before truncate.")
	}

	if be.bloblen == newsize {
		return nil
	}
	be.markDirtyWithLock()
	if err := be.cachebh.Truncate(newsize); err != nil {
		return err
	}
	be.bloblen = newsize
	be.validlen = newsize
	return nil
}

func (be *CachedBlobEntry) writeBackWithLock() error {
	if be.state == cacheEntryInvalidating {
		panic("writeback while invalidating isn't supported!!!")
	}
	if be.state != cacheEntryDirty {
		return nil
	}

	cachever, err := be.cbs.queryVersion(&blobstore.OffsetReader{be.cachebh, 0})
	if err != nil {
		return fmt.Errorf("Failed to query cached blob ver: %v", err)
	}

	w, err := be.cbs.backendbs.OpenWriter(be.blobpath)
	if err != nil {
		return fmt.Errorf("Failed to open backend blob writer: %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			fmt.Printf("Failed to close backend blob writer: %v", err)
		}
	}()
	r := io.LimitReader(&blobstore.OffsetReader{be.cachebh, 0}, be.cachebh.Size())
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("Failed to copy dirty data to backend blob writer: %v", err)
	}

	be.cbs.bever.Set(be.blobpath, cachever)
	be.state = cacheEntryClean
	return nil
}

var _ = util.Syncer(&CachedBlobEntry{})

func (be *CachedBlobEntry) Sync() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	// Wait for invalidation to complete
	if err := be.waitUntilInvalidateDone(); err != nil {
		return err
	}

	if !be.state.IsActive() {
		logger.Warningf(mylog, "Attempted to sync already uninitialized/closed entry: %+v", be.infoWithLock())
		return nil
	}
	if be.state == cacheEntryClean {
		return nil
	}

	logger.Infof(mylog, "Sync entry: %+v", be.infoWithLock())

	errC := make(chan error)

	go func() {
		if err := be.writeBackWithLock(); err != nil {
			errC <- fmt.Errorf("Failed to writeback dirty: %v", err)
		} else {
			errC <- nil
		}
	}()

	go func() {
		if cs, ok := be.cachebh.(util.Syncer); ok {
			if err := cs.Sync(); err != nil {
				errC <- fmt.Errorf("Failed to sync cache blob: %v", err)
			} else {
				errC <- nil
			}
		} else {
			errC <- nil
		}
	}()

	errs := []error{}
	for i := 0; i < 2; i++ {
		if err := <-errC; err != nil {
			errs = append(errs, err)
		}
	}

	be.syncCount++
	be.lastSync = time.Now()
	return util.ToErrors(errs)
}

const (
	abandonAndClose   = true
	writebackAndClose = false
)

func (be *CachedBlobEntry) closeWithLock(abandon bool) error {
	if len(be.handles) > 0 {
		return fmt.Errorf("Entry has %d handles", len(be.handles))
	}

	logger.Infof(mylog, "Close entry: %+v", be.infoWithLock())

	if !abandon {
		if err := be.waitUntilInvalidateDone(); err != nil {
			return err
		}

		if err := be.writeBackWithLock(); err != nil {
			return fmt.Errorf("Failed to writeback dirty: %v", err)
		}
		be.syncCount++
		be.lastSync = time.Now()
	}

	if be.cachebh != nil {
		if err := be.cachebh.Close(); err != nil {
			return fmt.Errorf("Failed to close cache bh: %v", err)
		}
	}

	be.state = cacheEntryClosed
	return nil
}

func (be *CachedBlobEntry) Close(abandon bool) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if !be.state.IsActive() {
		logger.Warningf(mylog, "Attempted to close uninitialized/already closed entry: %+v", be.infoWithLock())
		return nil
	}

	return be.closeWithLock(abandon)
}

type CachedBlobEntryInfo struct {
	BlobPath              string    `json:"blobpath"`
	State                 string    `json:"state"`
	BlobLen               int64     `json:"blob_len"`
	ValidLen              int64     `json:"valid_len"`
	SyncCount             int       `json:"sync_count"`
	LastUsed              time.Time `json:"last_used"`
	LastWrite             time.Time `json:"last_write"`
	LastSync              time.Time `json:"last_sync"`
	NumberOfWriterHandles int       `json:"number_of_writer_handles"`
	NumberOfHandles       int       `json:"number_of_handles"`
}

func (be *CachedBlobEntry) infoWithLock() *CachedBlobEntryInfo {
	numWriters := 0
	for h, _ := range be.handles {
		if fl.IsWriteAllowed(h.Flags()) {
			numWriters++
		}
	}

	return &CachedBlobEntryInfo{
		BlobPath:              be.blobpath,
		State:                 be.state.String(),
		BlobLen:               be.bloblen,
		ValidLen:              be.validlen,
		SyncCount:             be.syncCount,
		LastUsed:              be.lastUsed,
		LastWrite:             be.lastWrite,
		LastSync:              be.lastSync,
		NumberOfWriterHandles: numWriters,
		NumberOfHandles:       len(be.handles),
	}
}

func (be *CachedBlobEntry) Info() *CachedBlobEntryInfo {
	be.mu.Lock()
	defer be.mu.Unlock()

	return be.infoWithLock()
}

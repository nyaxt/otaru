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
	cacheEntryErrored
	cacheEntryErroredClosed
	cacheEntryClean
	cacheEntryWriteInProgress
	cacheEntryDirty
	cacheEntryDirtyClosing
	cacheEntryClosing
	cacheEntryClosed
)

func (s cacheEntryState) AcceptsIO() bool {
	return s == cacheEntryInvalidating || s == cacheEntryClean || s == cacheEntryWriteInProgress || s == cacheEntryDirty
}

func (s cacheEntryState) String() string {
	switch s {
	case cacheEntryUninitialized:
		return "Uninitialized"
	case cacheEntryInvalidating:
		return "Invalidating"
	case cacheEntryErrored:
		return "Errored"
	case cacheEntryErroredClosed:
		return "ErroredClosed"
	case cacheEntryClean:
		return "Clean"
	case cacheEntryWriteInProgress:
		return "WriteInProgress"
	case cacheEntryDirty:
		return "Dirty"
	case cacheEntryDirtyClosing:
		return "DirtyClosing"
	case cacheEntryClosing:
		return "Closing"
	case cacheEntryClosed:
		return "Closed"
	default:
		return "<unknown>"
	}
}

type CachedBlobEntry struct {
	mu               sync.Mutex
	progressCond     *sync.Cond
	cancelInvalidate func()

	cbs      *CachedBlobStore
	blobpath string
	cachebh  blobstore.BlobHandle

	state cacheEntryState

	bloblen  int64
	validlen int64

	lastUsed  time.Time
	lastWrite time.Time
	lastSync  time.Time
	syncCount int

	handles map[*CachedBlobHandle]struct{}
}

func NewCachedBlobEntry(blobpath string) *CachedBlobEntry {
	be := &CachedBlobEntry{
		state:    cacheEntryUninitialized,
		blobpath: blobpath,
		bloblen:  -1,
	}
	be.progressCond = sync.NewCond(&be.mu)
	return be
}

func (be *CachedBlobEntry) updateState(s cacheEntryState) {
	logger.Debugf(mylog, "Cache state \"%s\": %v -> %v", be.blobpath, be.state, s)
	be.state = s
}

func (be *CachedBlobEntry) AcceptsIO() bool {
	return be.state.AcceptsIO()
}

const invalidateBlockSize int = 32 * 1024

type InvalidateFailedErr struct {
	Blobpath string
}

func (e InvalidateFailedErr) Error() string {
	return fmt.Sprintf("InvalidationFailed for blob \"%s\"", e.Blobpath)
}

/*
func IsInvalidateFailedErr(e error) bool {
	if e == nil {
		return false
	}

	_, ok := e.(InvalidateFailedErr)
	return ok
}
*/

func (be *CachedBlobEntry) waitUntilInvalidateAtLeast(requiredLen int64) error {
	logger.Infof(mylog, "Waiting for cache to be fulfilled: reqlen: %d, validlen: %d", requiredLen, be.validlen)
	for {
		switch be.state {
		case cacheEntryInvalidating:
			break

		case cacheEntryErrored, cacheEntryClosing, cacheEntryErroredClosed:
			return InvalidateFailedErr{be.blobpath}

		case cacheEntryClean, cacheEntryDirty:
			return nil

		case cacheEntryUninitialized, cacheEntryDirtyClosing, cacheEntryClosed:
			logger.Panicf(mylog, "Invalid state while in waitUntilInvalidateAtLeast! %+v", be)
		}

		if be.validlen >= requiredLen {
			return nil
		}

		be.progressCond.Wait()
	}
}

func (be *CachedBlobEntry) waitUntilInvalidateDone() error {
	return be.waitUntilInvalidateAtLeast(be.bloblen)
}

// invalidateInternal is invalidate logic but errror handling.
// should only be called from invalidate()
func (be *CachedBlobEntry) invalidateInternal(ctx context.Context) error {
	be.mu.Lock()
	cbs := be.cbs
	ul := util.EnsureUnlocker{&be.mu}
	defer ul.Unlock()
	blobpath := be.blobpath
	if be.state != cacheEntryInvalidating {
		return fmt.Errorf("invalidate: blobentry in invalid state: %+v", be)
	}
	if be.validlen >= be.bloblen {
		logger.Warningf(mylog, "tried to invalidate a blobentry already clean: %+v", be)
		be.updateState(cacheEntryClean)
		return nil
	}

	ctx, cancelfn := context.WithCancel(ctx)
	be.cancelInvalidate = cancelfn
	ul.Unlock()

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
	done := false
	for !done {
		nr, er := cancellable.Read(ctx, backendr, buf)
		if nr > 0 {
			nw, ew := cachew.Write(buf[:nr])
			if nw > 0 {
				be.mu.Lock()
				be.validlen += int64(nw)
				be.progressCond.Broadcast()
				if be.validlen == be.bloblen {
					be.updateState(cacheEntryClean)
					be.cancelInvalidate = nil
					done = true
				} else if be.validlen > be.bloblen {
					logger.Panicf(mylog, "wrote more than bloblen: %d > %d", be.validlen, be.bloblen)
				}
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
			return er
		}
	}

	return nil
}

// invalidate fetches new version of the blob from backendbs.
// This func should be only called from be.invalidateCache()
func (be *CachedBlobEntry) invalidate(ctx context.Context) error {
	if err := be.invalidateInternal(ctx); err != nil {
		be.mu.Lock()
		be.validlen = 0
		be.updateState(cacheEntryErrored)
		go be.CloseWithLogErr(abandonAndClose)
		be.progressCond.Broadcast()
		be.mu.Unlock()
		if !cancellable.IsCancelledErr(err) {
			logger.Criticalf(mylog, "Failed to invalidate entry \"%s\". err: %v", be.blobpath, err)
		}
		return err
	}
	return nil
}

func (be *CachedBlobEntry) initializeWithLock(cbs *CachedBlobStore) error {
	cachebh, err := cbs.cachebs.Open(be.blobpath, fl.O_RDWRCREATE)
	if err != nil {
		be.updateState(cacheEntryErrored)
		go be.CloseWithLogErr(abandonAndClose)
		return fmt.Errorf("Failed to open cache blob: %v", err)
	}
	cachever, err := cbs.queryVersion(&blobstore.OffsetReader{cachebh, 0})
	if err != nil {
		be.updateState(cacheEntryErrored)
		go be.CloseWithLogErr(abandonAndClose)
		return fmt.Errorf("Failed to query cached blob ver: %v", err)
	}
	backendver, err := cbs.bever.Query(be.blobpath)
	if err != nil {
		be.updateState(cacheEntryErrored)
		go be.CloseWithLogErr(abandonAndClose)
		return err
	}

	be.cbs = cbs
	be.cachebh = cachebh
	be.handles = make(map[*CachedBlobHandle]struct{})

	if cachever > backendver {
		logger.Warningf(mylog, "FIXME: cache is newer than backend when open")
		be.updateState(cacheEntryDirty)
		be.bloblen = cachebh.Size()
		be.validlen = be.bloblen
	} else if cachever == backendver {
		be.updateState(cacheEntryClean)
		be.bloblen = cachebh.Size()
		be.validlen = be.bloblen
	} else {
		blobsizer := cbs.backendbs.(blobstore.BlobSizer)
		be.bloblen, err = blobsizer.BlobSize(be.blobpath)
		if err != nil {
			be.updateState(cacheEntryErrored)
			go be.CloseWithLogErr(abandonAndClose)
			return fmt.Errorf("Failed to query backend blobsize: %v", err)
		}
		if be.bloblen == 0 {
			be.updateState(cacheEntryClean)
		} else {
			be.updateState(cacheEntryInvalidating)
			be.validlen = 0
			cbs.s.RunImmediately(&InvalidateCacheTask{be}, nil)
		}
	}
	if be.state == cacheEntryUninitialized {
		panic("be.state should be set above")
	}

	return nil
}

func (be *CachedBlobEntry) OpenHandle(cbs *CachedBlobStore, flags int) (*CachedBlobHandle, error) {
	be.mu.Lock()
	defer be.mu.Unlock()

Loop:
	for {
		switch be.state {
		case cacheEntryClosed, cacheEntryUninitialized:
			if err := be.initializeWithLock(cbs); err != nil {
				return nil, err
			}

		case cacheEntryErrored:
			return nil, fmt.Errorf("Cache entry is in errored state.")

		case cacheEntryErroredClosed:
			return nil, fmt.Errorf("Previous attempt to open the entry has failed. Declining to OpenHandle.")

		case cacheEntryInvalidating, cacheEntryClean, cacheEntryWriteInProgress, cacheEntryDirty:
			break Loop

		case cacheEntryClosing:
			be.progressCond.Wait()
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
	be.mu.Lock()
	be.lastUsed = time.Now()
	requiredLen := util.Int64Min(offset+int64(len(p)), be.bloblen)
	if err := be.waitUntilInvalidateAtLeast(requiredLen); err != nil {
		be.mu.Unlock()
		return err
	}
	be.mu.Unlock()

	return be.cachebh.PRead(p, offset)
}

func (be *CachedBlobEntry) LastWrite() time.Time { return be.lastWrite }
func (be *CachedBlobEntry) LastSync() time.Time  { return be.lastSync }

func (be *CachedBlobEntry) markWriteInProgressWithLock() {
	now := time.Now()
	be.lastUsed = now
	be.lastWrite = now

	if be.state != cacheEntryClean && be.state != cacheEntryDirty {
		logger.Panicf(mylog, "markWriteInProgressWithLock called from unexpected state: %+v", be.infoWithLock())
	}
	be.updateState(cacheEntryWriteInProgress)

	if be.lastSync.IsZero() {
		be.lastSync = time.Now()
	}
}

func (be *CachedBlobEntry) markDirtyWithLock() {
	if be.state != cacheEntryWriteInProgress {
		logger.Panicf(mylog, "markDirtyWithLock called from unexpected state: %+v", be.infoWithLock())
	}

	be.updateState(cacheEntryDirty)
	be.progressCond.Broadcast()
}

func (be *CachedBlobEntry) PWrite(p []byte, offset int64) error {
	be.mu.Lock()
	ul := util.EnsureUnlocker{&be.mu}
	defer ul.Unlock()

	// Avoid any write when in invalidating state.
	// FIXME: maybe allow when offset+len(p) < be.validlen
	if err := be.waitUntilInvalidateDone(); err != nil {
		return err
	}

	if len(p) == 0 {
		return nil
	}
	right := offset + int64(len(p))
	if right > be.bloblen {
		be.bloblen = right
		be.validlen = right
	}

	be.markWriteInProgressWithLock()
	ul.Unlock()

	err := be.cachebh.PWrite(p, offset)

	be.mu.Lock()
	be.markDirtyWithLock()
	be.mu.Unlock()
	return err
}

func (be *CachedBlobEntry) Size() int64 {
	return be.bloblen
}

func (be *CachedBlobEntry) Truncate(newsize int64) error {
	be.mu.Lock()
	ul := util.EnsureUnlocker{&be.mu}
	defer ul.Unlock()

	// Avoid truncate when in invalidating state.
	// FIXME: maybe allow if newsize < be.validlen
	if err := be.waitUntilInvalidateDone(); err != nil {
		logger.Infof(mylog, "Waiting for cache to be fully invalidated before truncate.")
	}

	if be.bloblen == newsize {
		return nil
	}
	be.bloblen = newsize
	be.validlen = newsize

	be.markWriteInProgressWithLock()
	ul.Unlock()

	err := be.cachebh.Truncate(newsize)

	be.mu.Lock()
	be.markDirtyWithLock()
	be.mu.Unlock()
	return err
}

func (be *CachedBlobEntry) writeBackWithLock() error {
	logger.Debugf(mylog, "writeBackWithLock called for \"%s\" state %v", be.blobpath, be.state)
	switch be.state {
	case cacheEntryUninitialized, cacheEntryInvalidating, cacheEntryWriteInProgress, cacheEntryErrored, cacheEntryErroredClosed, cacheEntryClosed:
		logger.Panicf(mylog, "writeBackWithLock called for \"%s\" in state %v", be.blobpath, be.state)

	case cacheEntryClean, cacheEntryClosing:
		// no need to writeback
		return nil
	case cacheEntryDirty, cacheEntryDirtyClosing:
		break
	}

	// queryVersion is issued while holding be.mu, so it is guaranteed that no racing writes to be.cbs are issued after this query.
	cachever, err := be.cbs.queryVersion(&blobstore.OffsetReader{be.cachebh, 0})
	if err != nil {
		return fmt.Errorf("Failed to query cached blob ver: %v", err)
	}

	bever, err := be.cbs.bever.Query(be.blobpath)
	if err != nil {
		return fmt.Errorf("Failed to query backend blob ver: %v", err)
	}
	if bever == cachever {
		logger.Debugf(mylog, "writeBackWithLock \"%s\" write operations to the cache didn't increment its version.", be.blobpath, bever)
		be.updateState(cacheEntryClean)
		return nil
	} else if bever > cachever {
		return fmt.Errorf("backend version %d is newer than cached version %d when writeBack \"%s\"", bever, cachever, be.blobpath)
	}

	logger.Infof(mylog, "writeBack blob \"%s\" cache ver %d overwriting backend ver %d.", be.blobpath, cachever, bever)

	// unlock be.mu while writeback
	be.mu.Unlock()

	w, err := be.cbs.backendbs.OpenWriter(be.blobpath)
	if err != nil {
		be.mu.Lock()
		return fmt.Errorf("Failed to open backend blob writer: %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			fmt.Printf("Failed to close backend blob writer: %v", err)
		}
	}()
	r := io.LimitReader(&blobstore.OffsetReader{be.cachebh, 0}, be.cachebh.Size())
	if _, err := io.Copy(w, r); err != nil {
		be.mu.Lock()
		return fmt.Errorf("Failed to copy dirty data to backend blob writer: %v", err)
	}

	be.mu.Lock()
	be.cbs.bever.Set(be.blobpath, cachever)

	switch be.state {
	case cacheEntryUninitialized:
		logger.Panicf(mylog, "Sync shouldn't get into this state: %+v", be.infoWithLock())

	case cacheEntryInvalidating, cacheEntryErrored, cacheEntryErroredClosed, cacheEntryClosing, cacheEntryClosed:
		return nil

	case cacheEntryClean:
		logger.Infof(mylog, "cache entry already clean after writeBack. Possibly two writeBack ran concurrently?")
		return nil

	case cacheEntryDirty:
		cacheverAfterWriteback, err := be.cbs.queryVersion(&blobstore.OffsetReader{be.cachebh, 0})
		if err != nil {
			return fmt.Errorf("Failed to query cached blob ver: %v", err)
		}
		if cacheverAfterWriteback == cachever {
			be.updateState(cacheEntryClean)
		}
	}
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

Loop:
	for {
		switch be.state {
		case cacheEntryUninitialized, cacheEntryInvalidating, cacheEntryErrored, cacheEntryErroredClosed:
			logger.Panicf(mylog, "Sync shouldn't get into this state: %+v", be.infoWithLock())

		case cacheEntryClean:
			return nil

		case cacheEntryWriteInProgress:
			logger.Debugf(mylog, "Sync for \"%s\" waiting for write to finish", be.blobpath)
			be.progressCond.Wait()

		case cacheEntryDirty:
			break Loop

		case cacheEntryClosing, cacheEntryClosed:
			logger.Warningf(mylog, "Attempted sync on closed entry: %+v", be.infoWithLock())
			return nil
		}
	}

	logger.Infof(mylog, "Sync entry \"%s\"", be.blobpath)
	start := time.Now()

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
	logger.Infof(mylog, "Sync entry \"%s\" took %v", be.blobpath, be.lastSync.Sub(start))
	return util.ToErrors(errs)
}

const (
	abandonAndClose   = true
	writebackAndClose = false
)

func (be *CachedBlobEntry) CloseWithLogErr(abandon bool) {
	if err := be.Close(abandon); err != nil {
		logger.Warningf(mylog, "Close blobentry \"%s\" failed with err: %v", be.blobpath, err)
	}
}

func (be *CachedBlobEntry) Close(abandon bool) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.state != cacheEntryErrored {
		if nhandles := len(be.handles); nhandles > 0 {
			return fmt.Errorf("Entry has %d handles", nhandles)
		}
	}

	if fn := be.cancelInvalidate; fn != nil {
		logger.Debugf(mylog, "cancelInvalidate triggered for blob cache \"%s\"", be.blobpath)
		fn()
	}

	be.waitUntilInvalidateDone()
	wasErrored := be.state == cacheEntryErrored
	switch be.state {
	case cacheEntryUninitialized, cacheEntryWriteInProgress:
		return fmt.Errorf("logicerr: cacheBlobEntry \"%s\" of state %v shouldn't be Close()-d", be.blobpath, be.state)

	case cacheEntryDirtyClosing, cacheEntryClosing, cacheEntryErroredClosed, cacheEntryClosed:
		logger.Debugf(mylog, "blob cache \"%s\" already (being) closed: %v", be.blobpath, be.state)
		return nil

	case cacheEntryInvalidating:
		if abandon != abandonAndClose {
			return fmt.Errorf("invalidating entry \"%s\" can be only closed if going to be abandoned", be.blobpath)
		}
		be.updateState(cacheEntryClosing)

	case cacheEntryErrored:
		if abandon != abandonAndClose {
			logger.Warningf(mylog, "errored entry \"%s\" should be abandoned", be.blobpath)
		}
		be.updateState(cacheEntryClosing)

	case cacheEntryClean:
		be.updateState(cacheEntryClosing)

	case cacheEntryDirty:
		be.updateState(cacheEntryDirtyClosing)
	}

	logger.Infof(mylog, "Close entry: %+v", be.infoWithLock())

	if abandon == writebackAndClose {
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

	if wasErrored {
		br := be.cbs.cachebs.(blobstore.BlobRemover)
		if err := br.RemoveBlob(be.blobpath); err != nil {
			logger.Criticalf(mylog, "Failed to remove errored blob cache \"%v\"", be.blobpath)
		}
		be.updateState(cacheEntryErroredClosed)
	} else {
		be.updateState(cacheEntryClosed)
	}
	be.progressCond.Broadcast()
	return nil
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

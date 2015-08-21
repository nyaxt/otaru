package cachedblobstore

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/version"
	"github.com/nyaxt/otaru/btncrypt"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
	"github.com/nyaxt/otaru/util/cancellable"
)

var mylog = logger.Registry().Category("cachedbs")

const (
	EPERM  = syscall.Errno(syscall.EPERM)
	ENFILE = syscall.Errno(syscall.ENFILE)
)

type CachedBlobStore struct {
	backendbs blobstore.BlobStore
	cachebs   blobstore.RandomAccessBlobStore
	s         *scheduler.Scheduler

	flags int

	queryVersion version.QueryFunc
	bever        *CachedBackendVersion

	entriesmgr CachedBlobEntriesManager
	usagestats *CacheUsageStats
}

const maxEntries = 128

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

func (cbs *CachedBlobStore) Sync() error {
	return cbs.entriesmgr.SyncAll()
}

const (
	syncTimeoutDuration  = 300 * time.Second
	writeTimeoutDuration = 3 * time.Second
)

func (cbs *CachedBlobStore) SyncOneEntry() error {
	be := cbs.entriesmgr.ChooseSyncEntry()
	if be == nil {
		return ENOENT
	}

	return be.Sync()
}

func New(backendbs blobstore.BlobStore, cachebs blobstore.RandomAccessBlobStore, s *scheduler.Scheduler, flags int, queryVersion version.QueryFunc) (*CachedBlobStore, error) {
	if fl.IsWriteAllowed(flags) {
		if fr, ok := backendbs.(fl.FlagsReader); ok {
			if !fl.IsWriteAllowed(fr.Flags()) {
				return nil, fmt.Errorf("Writable CachedBlobStore requested, but backendbs doesn't allow writes")
			}
		}
	}
	if !fl.IsWriteAllowed(cachebs.Flags()) {
		return nil, fmt.Errorf("CachedBlobStore requested, but cachebs doesn't allow writes")
	}

	cbs := &CachedBlobStore{
		backendbs:    backendbs,
		cachebs:      cachebs,
		s:            s,
		flags:        flags,
		queryVersion: queryVersion,
		bever:        NewCachedBackendVersion(backendbs, queryVersion),
		entriesmgr:   NewCachedBlobEntriesManager(),
		usagestats:   NewCacheUsageStats(),
	}

	if lister, ok := cachebs.(blobstore.BlobLister); ok {
		bps, err := lister.ListBlobs()
		if err != nil {
			return nil, fmt.Errorf("Failed to list blobs to init CacheUsageStats: %v", err)
		}
		cbs.usagestats.ImportBlobList(bps)
	}

	go cbs.entriesmgr.Run()
	return cbs, nil
}

func (cbs *CachedBlobStore) RestoreState(c btncrypt.Cipher) error {
	errs := []error{}

	if err := cbs.bever.RestoreStateFromBlobstore(c, cbs.backendbs); err != nil {
		errs = append(errs, err)
	}

	return util.ToErrors(errs)
}

func (cbs *CachedBlobStore) SaveState(c btncrypt.Cipher) error {
	errs := []error{}

	if err := cbs.bever.SaveStateToBlobstore(c, cbs.backendbs); err != nil {
		errs = append(errs, err)
	}

	return util.ToErrors(errs)
}

var _ = blobstore.BlobStore(&CachedBlobStore{})

func (cbs *CachedBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	bh, err := cbs.Open(blobpath, fl.O_RDONLY)
	if err != nil {
		return nil, err
	}
	return &struct {
		blobstore.OffsetReader // for io.Reader
		blobstore.BlobHandle   // for io.Closer
	}{
		blobstore.OffsetReader{bh, 0},
		bh,
	}, nil
}

func (cbs *CachedBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	bh, err := cbs.Open(blobpath, fl.O_WRONLY|fl.O_CREATE)
	if err != nil {
		return nil, err
	}
	if err := bh.Truncate(0); err != nil {
		return nil, err
	}
	return &struct {
		blobstore.OffsetWriter // for io.Writer
		blobstore.BlobHandle   // for io.Closer
	}{
		blobstore.OffsetWriter{bh, 0},
		bh,
	}, nil
}

var _ = blobstore.RandomAccessBlobStore(&CachedBlobStore{})

func (cbs *CachedBlobStore) Flags() int {
	return cbs.flags
}

func (cbs *CachedBlobStore) Open(blobpath string, flags int) (blobstore.BlobHandle, error) {
	if !fl.IsWriteAllowed(cbs.flags) && fl.IsWriteAllowed(flags) {
		return nil, EPERM
	}

	cbs.usagestats.ObserveOpen(blobpath, flags)
	be, err := cbs.entriesmgr.OpenEntry(blobpath)
	if err != nil {
		return nil, err
	}
	return be.OpenHandle(cbs, flags)
}

func (cbs *CachedBlobStore) DumpEntriesInfo() []*CachedBlobEntryInfo {
	return cbs.entriesmgr.DumpEntriesInfo()
}

func (*CachedBlobStore) ImplName() string { return "CachedBlobStore" }

var _ = blobstore.BlobLister(&CachedBlobStore{})

func (cbs *CachedBlobStore) ListBlobs() ([]string, error) {
	belister, ok := cbs.backendbs.(blobstore.BlobLister)
	if !ok {
		return nil, fmt.Errorf("Backendbs \"%v\" doesn't support listing blobs.", util.TryGetImplName(cbs.backendbs))
	}

	belist, err := belister.ListBlobs()
	if err != nil {
		return nil, fmt.Errorf("Backendbs failed to ListBlobs: %v", err)
	}
	cachelist := cbs.entriesmgr.ListBlobs()
	cachelistset := make(map[string]struct{})
	for _, blobpath := range cachelist {
		cachelistset[blobpath] = struct{}{}
	}

	// list = append(cachelist, ...belist but entries already in cachelist)
	list := cachelist
	for _, blobpath := range belist {
		if _, ok := cachelistset[blobpath]; ok {
			// already in cachelist
			continue
		}
		list = append(list, blobpath)
	}

	return list, nil
}

var _ = blobstore.BlobRemover(&CachedBlobStore{})

func (cbs *CachedBlobStore) RemoveBlob(blobpath string) error {
	backendrm, ok := cbs.backendbs.(blobstore.BlobRemover)
	if !ok {
		return fmt.Errorf("Backendbs \"%v\" doesn't support removing blobs.", util.TryGetImplName(cbs.backendbs))
	}
	cacherm, ok := cbs.cachebs.(blobstore.BlobRemover)
	if !ok {
		return fmt.Errorf("Cachebs \"%v\" doesn't support removing blobs.", util.TryGetImplName(cbs.cachebs))
	}

	if err := cbs.entriesmgr.RemoveBlob(blobpath); err != nil {
		return err
	}
	cbs.bever.Delete(blobpath)
	if err := backendrm.RemoveBlob(blobpath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Backendbs RemoveBlob failed: %v", err)
	}
	cbs.usagestats.ObserveRemoveBlob(blobpath)
	if err := cacherm.RemoveBlob(blobpath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Cachebs RemoveBlob failed: %v", err)
	}

	return nil
}

func (cbs *CachedBlobStore) ReduceCache(ctx context.Context, desiredSize int64, dryrun bool) error {
	start := time.Now()

	tsizer, ok := cbs.cachebs.(blobstore.TotalSizer)
	if !ok {
		return fmt.Errorf("Cache backend \"%s\" doesn't support TotalSize() method, required to ReduceCache(). aborting.", util.TryGetImplName(cbs.cachebs))
	}

	blobsizer, ok := cbs.cachebs.(blobstore.BlobSizer)
	if !ok {
		return fmt.Errorf("Cache backend \"%s\" doesn't support BlobSize() method, required to ReduceCache(). aborting.", util.TryGetImplName(cbs.cachebs))
	}

	blobremover, ok := cbs.cachebs.(blobstore.BlobRemover)
	if !ok {
		return fmt.Errorf("Cache backend \"%s\" doesn't support RemoveBlob() method, required to ReduceCache(). aborting.", util.TryGetImplName(cbs.cachebs))
	}

	totalSizeBefore, err := tsizer.TotalSize()
	if err != nil {
		return fmt.Errorf("Failed to query current total cache size: %v", err)
	}

	needsReduce := totalSizeBefore - desiredSize
	if needsReduce < 0 {
		logger.Infof(mylog, "ReduceCache: No need to reduce cache as its already under desired size! No-op.")
		return nil
	}
	logger.Infof(mylog, "ReduceCache: Current cache bs total size: %s. Desired size: %s. Needs to reduce %s.",
		humanize.IBytes(uint64(totalSizeBefore)), humanize.IBytes(uint64(desiredSize)), humanize.IBytes(uint64(needsReduce)))

	bps := cbs.usagestats.FindLeastUsed()
	for _, bp := range bps {
		size, err := blobsizer.BlobSize(bp)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Infof(mylog, "Attempted to drop blob cache \"%s\", but not found. Maybe it's already removed.", bp)
				continue
			}
			return fmt.Errorf("Failed to query size for cache blob \"%s\": %v", bp, err)
		}

		logger.Infof(mylog, "ReduceCache: Drop entry \"%s\" to release %s", bp, humanize.IBytes(uint64(size)))

		if !dryrun {
			if err := cbs.entriesmgr.DropCacheEntry(bp, blobremover); err != nil {
				return fmt.Errorf("Failed to remove cache blob \"%s\": %v", bp, err)
			}
		}

		needsReduce -= size
		if needsReduce < 0 {
			break
		}
	}

	totalSizeAfter, err := tsizer.TotalSize()
	if err != nil {
		return fmt.Errorf("Failed to query current total cache size: %v", err)
	}

	logger.Infof(mylog, "ReduceCache done. Cache bs total size: %s -> %s. Dryrun: %t. Took: %s",
		humanize.IBytes(uint64(totalSizeBefore)), humanize.IBytes(uint64(totalSizeAfter)),
		dryrun, time.Since(start))
	return nil
}

type Stats struct {
	CacheUsageStatsView `json:"usage_stats"`
	CbvStats            `json:"cbv_stats"`
}

func (cbs *CachedBlobStore) GetStats() Stats {
	return Stats{
		CacheUsageStatsView: cbs.usagestats.View(),
		CbvStats:            cbs.bever.GetStats(),
	}
}

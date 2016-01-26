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

type CacheEntryState int

const (
	CacheEntryUninitialized CacheEntryState = iota
	CacheEntryInvalidating
	CacheEntryErrored
	CacheEntryErroredClosed
	CacheEntryClean
	CacheEntryWriteInProgress
	CacheEntryWritebackInProgress
	CacheEntryStaleWritebackInProgress
	CacheEntryDirty
	CacheEntryDirtyClosing
	CacheEntryClosing
	CacheEntryClosed
)

func (s CacheEntryState) ShouldBeListed() bool {
	return s == CacheEntryInvalidating ||
		s == CacheEntryClean ||
		s == CacheEntryWriteInProgress ||
		s == CacheEntryWritebackInProgress ||
		s == CacheEntryStaleWritebackInProgress ||
		s == CacheEntryDirty
}

func (s CacheEntryState) NeedsSync() bool {
	return s == CacheEntryDirty || s == CacheEntryWriteInProgress
}

func (s CacheEntryState) String() string {
	switch s {
	case CacheEntryUninitialized:
		return "Uninitialized"
	case CacheEntryInvalidating:
		return "Invalidating"
	case CacheEntryErrored:
		return "Errored"
	case CacheEntryErroredClosed:
		return "ErroredClosed"
	case CacheEntryClean:
		return "Clean"
	case CacheEntryWriteInProgress:
		return "WriteInProgress"
	case CacheEntryWritebackInProgress:
		return "WritebackInProgress"
	case CacheEntryStaleWritebackInProgress:
		return "StaleWritebackInProgress"
	case CacheEntryDirty:
		return "Dirty"
	case CacheEntryDirtyClosing:
		return "DirtyClosing"
	case CacheEntryClosing:
		return "Closing"
	case CacheEntryClosed:
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

	state CacheEntryState

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
		state:    CacheEntryUninitialized,
		blobpath: blobpath,
		bloblen:  -1,
	}
	be.progressCond = sync.NewCond(&be.mu)
	return be
}

func (be *CachedBlobEntry) updateState(newState CacheEntryState) {
	logger.Debugf(mylog, "Cache state \"%s\": %v -> %v", be.blobpath, be.state, newState)
	switch be.state {
	case CacheEntryUninitialized:
		switch newState {
		case CacheEntryInvalidating, CacheEntryClean, CacheEntryErrored:
			break
		default:
			goto Unexpected
		}
	case CacheEntryInvalidating:
		switch newState {
		case CacheEntryClean, CacheEntryErrored:
			break
		default:
			goto Unexpected
		}
	case CacheEntryErrored:
		switch newState {
		case CacheEntryErroredClosed:
			break
		default:
			goto Unexpected
		}
	case CacheEntryErroredClosed:
		goto Unexpected
	case CacheEntryClean:
		switch newState {
		case CacheEntryWriteInProgress, CacheEntryClosing:
			break
		default:
			goto Unexpected
		}
	case CacheEntryWriteInProgress:
		switch newState {
		case CacheEntryDirty:
			break
		default:
			goto Unexpected
		}
	case CacheEntryWritebackInProgress:
		switch newState {
		case CacheEntryClean, CacheEntryStaleWritebackInProgress, CacheEntryErrored:
			break
		default:
			goto Unexpected
		}
	case CacheEntryStaleWritebackInProgress:
		switch newState {
		case CacheEntryDirty, CacheEntryStaleWritebackInProgress:
			break
		default:
			goto Unexpected
		}
	case CacheEntryDirty:
		switch newState {
		case CacheEntryWriteInProgress, CacheEntryWritebackInProgress, CacheEntryDirtyClosing:
			break
		default:
			goto Unexpected
		}
	case CacheEntryDirtyClosing:
		switch newState {
		case CacheEntryClosed:
			break
		default:
			goto Unexpected
		}
	case CacheEntryClosing:
		switch newState {
		case CacheEntryClosed:
			break
		default:
			goto Unexpected
		}
	case CacheEntryClosed:
		switch newState {
		case CacheEntryClean, CacheEntryErrored:
			break
		default:
			goto Unexpected
		}
	}
	be.state = newState
	return

Unexpected:
	logger.Panicf(mylog, "Unexpected cache state \"%s\": %v -> %v", be.blobpath, be.state, newState)

	be.state = newState
	return
}

func (be *CachedBlobEntry) ShouldBeListed() bool {
	return be.state.ShouldBeListed()
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
	for {
		switch be.state {
		case CacheEntryInvalidating:
			break

		case CacheEntryErrored, CacheEntryClosing, CacheEntryErroredClosed:
			return InvalidateFailedErr{be.blobpath}

		case CacheEntryClean, CacheEntryDirty, CacheEntryWriteInProgress, CacheEntryWritebackInProgress, CacheEntryStaleWritebackInProgress:
			return nil

		case CacheEntryUninitialized, CacheEntryDirtyClosing, CacheEntryClosed:
			logger.Panicf(mylog, "Invalid state while in waitUntilInvalidateAtLeast! %+v", be)
		}

		if be.validlen >= requiredLen {
			return nil
		}
		logger.Debugf(mylog, "Waiting for cache to be fulfilled: reqlen: %d, validlen: %d", requiredLen, be.validlen)

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
	if be.state != CacheEntryInvalidating {
		return fmt.Errorf("invalidate: blobentry in invalid state: %+v", be)
	}
	if be.validlen >= be.bloblen {
		logger.Warningf(mylog, "tried to invalidate a blobentry already clean: %+v", be)
		be.updateState(CacheEntryClean)
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
					be.updateState(CacheEntryClean)
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
		be.updateState(CacheEntryErrored)
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

func (be *CachedBlobEntry) InitializeForTesting(state CacheEntryState, lastWrite, lastSync time.Time) {
	be.mu.Lock()
	defer be.mu.Unlock()

	be.state = state
	be.lastWrite = lastWrite
	be.lastSync = lastSync
}

func (be *CachedBlobEntry) initializeWithLock(cbs *CachedBlobStore) error {
	cachebh, err := cbs.cachebs.Open(be.blobpath, fl.O_RDWRCREATE)
	if err != nil {
		be.updateState(CacheEntryErrored)
		go be.CloseWithLogErr(abandonAndClose)
		return fmt.Errorf("Failed to open cache blob: %v", err)
	}
	cachever, err := cbs.queryVersion(&blobstore.OffsetReader{cachebh, 0})
	if err != nil {
		be.updateState(CacheEntryErrored)
		go be.CloseWithLogErr(abandonAndClose)
		return fmt.Errorf("Failed to query cached blob ver: %v", err)
	}
	backendver, err := cbs.bever.Query(be.blobpath)
	if err != nil {
		be.updateState(CacheEntryErrored)
		go be.CloseWithLogErr(abandonAndClose)
		return err
	}

	be.cbs = cbs
	be.cachebh = cachebh
	be.handles = make(map[*CachedBlobHandle]struct{})

	if cachever > backendver {
		logger.Warningf(mylog, "Cache for blob \"%s\" ver %v is newer than backend %v when open. Previous sync stopped?",
			be.blobpath, cachever, backendver)
		be.updateState(CacheEntryClean)
		be.updateState(CacheEntryWriteInProgress)
		be.updateState(CacheEntryDirty)
		be.bloblen = cachebh.Size()
		be.validlen = be.bloblen
	} else if cachever == backendver {
		be.updateState(CacheEntryClean)
		be.bloblen = cachebh.Size()
		be.validlen = be.bloblen
	} else {
		blobsizer := cbs.backendbs.(blobstore.BlobSizer)
		be.bloblen, err = blobsizer.BlobSize(be.blobpath)
		if err != nil {
			be.updateState(CacheEntryErrored)
			go be.CloseWithLogErr(abandonAndClose)
			return fmt.Errorf("Failed to query backend blobsize: %v", err)
		}
		if be.bloblen == 0 {
			be.updateState(CacheEntryClean)
		} else {
			be.updateState(CacheEntryInvalidating)
			be.validlen = 0
			cbs.s.RunImmediately(&InvalidateCacheTask{be}, nil)
		}
	}
	if be.state == CacheEntryUninitialized {
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
		case CacheEntryClosed, CacheEntryUninitialized:
			if err := be.initializeWithLock(cbs); err != nil {
				return nil, err
			}

		case CacheEntryErrored:
			return nil, fmt.Errorf("Cache entry is in errored state.")

		case CacheEntryErroredClosed:
			return nil, fmt.Errorf("Previous attempt to open the entry has failed. Declining to OpenHandle.")

		case CacheEntryInvalidating, CacheEntryClean, CacheEntryWriteInProgress, CacheEntryWritebackInProgress, CacheEntryStaleWritebackInProgress, CacheEntryDirty:
			break Loop

		case CacheEntryClosing:
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

	switch be.state {
	case CacheEntryClean, CacheEntryDirty:
		be.updateState(CacheEntryWriteInProgress)
	case CacheEntryWritebackInProgress, CacheEntryStaleWritebackInProgress:
		be.updateState(CacheEntryStaleWritebackInProgress)
	default:
		logger.Panicf(mylog, "markWriteInProgressWithLock called from unexpected state: %+v", be.infoWithLock())
	}
	be.progressCond.Broadcast()

	if be.lastSync.IsZero() {
		be.lastSync = time.Now()
	}
}

func (be *CachedBlobEntry) markWriteEndWithLock() {
	switch be.state {
	case CacheEntryWriteInProgress:
		be.updateState(CacheEntryDirty)
	case CacheEntryStaleWritebackInProgress:
		break
	default:
		logger.Panicf(mylog, "markWriteEndWithLock called from unexpected state: %+v", be.infoWithLock())
	}

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
	be.markWriteEndWithLock()
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
	be.markWriteEndWithLock()
	be.mu.Unlock()
	return err
}

type writeBackCaller int

const (
	callerSync writeBackCaller = iota
	callerClose
)

func (be *CachedBlobEntry) writeBackWithLock(wbc writeBackCaller) error {
	logger.Debugf(mylog, "writeBackWithLock called for \"%s\" state %v", be.blobpath, be.state)
	switch be.state {
	case CacheEntryClean, CacheEntryClosing:
		// no need to writeback
		return nil

	case CacheEntryDirty, CacheEntryDirtyClosing:
		break

	default:
		logger.Panicf(mylog, "writeBackWithLock called for \"%s\" in state %v", be.blobpath, be.state)
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
		logger.Debugf(mylog, "writeBackWithLock \"%s\" write operations to the cache didn't increment its version: %v", be.blobpath, bever)

		// Note: use intermediate WritebackInProgress to simplify FSM
		be.updateState(CacheEntryWritebackInProgress)
		be.updateState(CacheEntryClean)
		return nil
	} else if bever > cachever {
		return fmt.Errorf("backend version %d is newer than cached version %d when writeBack \"%s\"", bever, cachever, be.blobpath)
	}

	if be.state == CacheEntryDirty {
		be.updateState(CacheEntryWritebackInProgress)
	} else {
		if be.state != CacheEntryDirtyClosing {
			logger.Panicf(mylog, "Unexpected state %s", be.state)
		}
	}
	logger.Infof(mylog, "writeBack blob \"%s\" cache ver %d overwriting backend ver %d.", be.blobpath, cachever, bever)
	if wbc == callerClose {
		be.cbs.stats.NumWritebackOnClose++
	}

	// unlock be.mu while writeback
	be.mu.Unlock()

	w, err := be.cbs.backendbs.OpenWriter(be.blobpath)
	if err != nil {
		be.mu.Lock()
		be.updateState(CacheEntryErrored)
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
		be.updateState(CacheEntryErrored)
		return fmt.Errorf("Failed to copy dirty data to backend blob writer: %v", err)
	}

	be.mu.Lock()
	be.cbs.bever.Set(be.blobpath, cachever)

	switch be.state {
	case CacheEntryDirtyClosing:
		return nil

	case CacheEntryWritebackInProgress:
		cacheverAfterWriteback, err := be.cbs.queryVersion(&blobstore.OffsetReader{be.cachebh, 0})
		if err != nil {
			logger.Criticalf(mylog, "Failed to query cached blob ver: %v", err)
			be.updateState(CacheEntryErrored)
			return fmt.Errorf("Failed to query cached blob ver: %v", err)
		}
		if cacheverAfterWriteback == cachever {
			be.updateState(CacheEntryClean)
			be.progressCond.Broadcast()
		} else {
			logger.Criticalf(mylog, "Entry version has changed while cachedEntryWritebackInProgress. was %d -> now %d.", cachever, cacheverAfterWriteback)
			be.updateState(CacheEntryDirty)
			be.progressCond.Broadcast()
		}
		return nil

	case CacheEntryStaleWritebackInProgress:
		be.updateState(CacheEntryDirty)
		be.progressCond.Broadcast()
		return nil

	default:
		logger.Criticalf(mylog, "Sync shouldn't get into this state: %+v", be.infoWithLock())
		return nil
	}
}

var _ = util.Syncer(&CachedBlobEntry{})

func (be *CachedBlobEntry) Sync() error {
	be.mu.Lock()
	defer be.mu.Unlock()

Loop:
	for {
		switch be.state {
		default:
			logger.Panicf(mylog, "Sync shouldn't get into this state: %+v", be.infoWithLock())

		case CacheEntryInvalidating:
			// Wait for invalidation to complete
			if err := be.waitUntilInvalidateDone(); err != nil {
				return err
			}

		case CacheEntryErrored, CacheEntryErroredClosed:
			logger.Warningf(mylog, "Attempted sync on errored entry: %+v", be.infoWithLock())
			return nil

		case CacheEntryClean:
			return nil

		case CacheEntryWriteInProgress:
			logger.Debugf(mylog, "Sync for \"%s\" waiting for write to finish.", be.blobpath)
			be.progressCond.Wait()

		case CacheEntryWritebackInProgress, CacheEntryStaleWritebackInProgress:
			logger.Debugf(mylog, "Sync for \"%s\" waiting for previous writeback to finish.", be.blobpath)
			be.progressCond.Wait()

		case CacheEntryDirty:
			break Loop

		case CacheEntryDirtyClosing, CacheEntryClosing, CacheEntryClosed:
			logger.Warningf(mylog, "Attempted sync on closing/closed entry: %+v", be.infoWithLock())
			return nil
		}
	}

	logger.Infof(mylog, "Sync entry \"%s\"", be.blobpath)
	start := time.Now()

	errC := make(chan error)

	go func() {
		if err := be.writeBackWithLock(callerSync); err != nil {
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

type CloseMode int

const (
	abandonAndClose CloseMode = iota
	writebackAndClose
	writebackIfNeededAndClose
)

func (be *CachedBlobEntry) CloseWithLogErr(mode CloseMode) {
	if err := be.Close(mode); err != nil {
		logger.Warningf(mylog, "Close blobentry \"%s\" failed with err: %v", be.blobpath, err)
	}
}

func (be *CachedBlobEntry) Close(mode CloseMode) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.state != CacheEntryErrored {
		if nhandles := len(be.handles); nhandles > 0 {
			return fmt.Errorf("Entry has %d handles", nhandles)
		}
	}
	if mode == writebackIfNeededAndClose {
		switch be.state {
		case CacheEntryErroredClosed:
			mode = writebackAndClose
		default:
			mode = abandonAndClose
		}
	}

	if fn := be.cancelInvalidate; fn != nil {
		logger.Debugf(mylog, "cancelInvalidate triggered for blob cache \"%s\"", be.blobpath)
		fn()
	}

	be.waitUntilInvalidateDone()
	var wasErrored bool
Loop:
	for {
		wasErrored = be.state == CacheEntryErrored

		switch be.state {
		case CacheEntryUninitialized, CacheEntryWriteInProgress:
			return fmt.Errorf("logicerr: cacheBlobEntry \"%s\" of state %v shouldn't be Close()-d", be.blobpath, be.state)

		case CacheEntryErroredClosed, CacheEntryClosed:
			logger.Debugf(mylog, "blob cache \"%s\" already closed: %v", be.blobpath, be.state)
			return nil

		case CacheEntryInvalidating:
			if mode != abandonAndClose {
				return fmt.Errorf("invalidating entry \"%s\" can be only closed if going to be abandoned", be.blobpath)
			}
			be.updateState(CacheEntryClosing)
			break Loop

		case CacheEntryErrored:
			if mode != abandonAndClose {
				logger.Warningf(mylog, "errored entry \"%s\" should be abandoned", be.blobpath)
			}
			mode = abandonAndClose
			break Loop

		case CacheEntryClean:
			be.updateState(CacheEntryClosing)
			break Loop

		case CacheEntryDirty:
			be.updateState(CacheEntryDirtyClosing)
			break Loop

		case CacheEntryWritebackInProgress, CacheEntryStaleWritebackInProgress, CacheEntryDirtyClosing, CacheEntryClosing:
			be.progressCond.Wait()
		}
	}

	logger.Infof(mylog, "Close entry: %+v", be.infoWithLock())

	if mode == writebackAndClose {
		if err := be.writeBackWithLock(callerClose); err != nil {
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
		be.updateState(CacheEntryErroredClosed)
	} else {
		be.updateState(CacheEntryClosed)
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

func (be *CachedBlobEntry) String() string {
	return fmt.Sprintf("CachedBlobEntry{\"%s\", %v}", be.blobpath, be.state)
}

func (be *CachedBlobEntry) BlobPath() string { return be.blobpath }

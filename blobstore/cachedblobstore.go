package blobstore

import (
	"fmt"
	"io"
	"log"
	"sync"
	"syscall"
	"time"

	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/util"
)

const (
	EPERM  = syscall.Errno(syscall.EPERM)
	ENFILE = syscall.Errno(syscall.ENFILE)
)

// FIXME: handle overflows
type BlobVersion int64

type QueryVersionFunc func(r io.Reader) (BlobVersion, error)

type CachedBlobEntriesManager struct {
	reqC chan interface{}

	entries map[string]*CachedBlobEntry
}

type SyncAllRequest struct {
	resultC chan error
}

type ChooseSyncEntryRequest struct {
	resultC chan *CachedBlobEntry
}

type DumpEntriesInfoRequest struct {
	resultC chan []*CachedBlobEntryInfo
}

type OpenEntryRequest struct {
	blobpath string
	resultC  chan interface{}
}

func NewCachedBlobEntriesManager() CachedBlobEntriesManager {
	return CachedBlobEntriesManager{
		reqC:    make(chan interface{}),
		entries: make(map[string]*CachedBlobEntry),
	}
}

func (mgr *CachedBlobEntriesManager) Run() {
	for req := range mgr.reqC {
		switch req.(type) {
		case *SyncAllRequest:
			req := req.(*SyncAllRequest)
			req.resultC <- mgr.doSyncAll()
		case *ChooseSyncEntryRequest:
			req := req.(*ChooseSyncEntryRequest)
			req.resultC <- mgr.doChooseSyncEntry()
		case *DumpEntriesInfoRequest:
			req := req.(*DumpEntriesInfoRequest)
			req.resultC <- mgr.doDumpEntriesInfo()
		case *OpenEntryRequest:
			req := req.(*OpenEntryRequest)
			be, err := mgr.doOpenEntry(req.blobpath)
			if err != nil {
				req.resultC <- err
			} else {
				req.resultC <- be
			}
		}
	}
}

func (mgr *CachedBlobEntriesManager) doSyncAll() error {
	errs := []error{}
	for blobpath, be := range mgr.entries {
		if err := be.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("Failed to sync \"%s\": %v", blobpath, err))
		}
	}
	return util.ToErrors(errs)
}

func (mgr *CachedBlobEntriesManager) SyncAll() error {
	req := &SyncAllRequest{resultC: make(chan error)}
	mgr.reqC <- req
	return <-req.resultC
}

func (mgr *CachedBlobEntriesManager) doChooseSyncEntry() *CachedBlobEntry {
	// Sync priorities:
	//   1. >300 sec since last sync
	//   2. >3 sec since last write

	now := time.Now()

	var oldestSync, oldestWrite *CachedBlobEntry
	oldestSyncT := now
	oldestWriteT := now

	for _, be := range mgr.entries {
		if be.state == cacheEntryDirty {
			continue
		}

		if oldestSyncT.After(be.LastSync()) {
			oldestSyncT = be.LastSync()
			oldestSync = be
		}
		if oldestWriteT.After(be.LastWrite()) {
			oldestWriteT = be.LastWrite()
			oldestWrite = be
		}
	}

	if now.Sub(oldestWriteT) > writeTimeoutDuration {
		return oldestWrite
	}
	if now.Sub(oldestSyncT) > syncTimeoutDuration {
		return oldestSync
	}
	return nil
}

func (mgr *CachedBlobEntriesManager) ChooseSyncEntry() *CachedBlobEntry {
	req := &ChooseSyncEntryRequest{resultC: make(chan *CachedBlobEntry)}
	mgr.reqC <- req
	return <-req.resultC
}

func (mgr *CachedBlobEntriesManager) doDumpEntriesInfo() []*CachedBlobEntryInfo {
	infos := make([]*CachedBlobEntryInfo, 0, len(mgr.entries))
	for _, be := range mgr.entries {
		infos = append(infos, be.Info())
	}
	return infos
}

func (mgr *CachedBlobEntriesManager) DumpEntriesInfo() []*CachedBlobEntryInfo {
	req := &DumpEntriesInfoRequest{resultC: make(chan []*CachedBlobEntryInfo)}
	mgr.reqC <- req
	return <-req.resultC
}

func (mgr *CachedBlobEntriesManager) doOpenEntry(blobpath string) (*CachedBlobEntry, error) {
	be, ok := mgr.entries[blobpath]
	if ok {
		return be, nil
	}

	if err := mgr.closeOldCacheEntriesIfNeeded(); err != nil {
		return be, err
	}

	be = &CachedBlobEntry{state: cacheEntryUninitialized, blobpath: blobpath}
	mgr.entries[blobpath] = be
	return be, nil
}

func (mgr *CachedBlobEntriesManager) OpenEntry(blobpath string) (*CachedBlobEntry, error) {
	req := &OpenEntryRequest{
		blobpath: blobpath,
		resultC:  make(chan interface{}),
	}
	mgr.reqC <- req
	res := <-req.resultC
	if err, ok := res.(error); ok {
		return nil, err
	}
	return res.(*CachedBlobEntry), nil
}

func (mgr *CachedBlobEntriesManager) tryCloseEntry(be *CachedBlobEntry) {
	if err := be.Close(); err != nil {
		log.Printf("Failed to close cache entry \"%s\": %v", be.blobpath, err)
		return
	}

	delete(mgr.entries, be.blobpath)
}

const inactiveCloseTimeout = 10 * time.Second

func (mgr *CachedBlobEntriesManager) closeOldCacheEntriesIfNeeded() error {
	if len(mgr.entries) <= maxEntries {
		return nil
	}

	threshold := time.Now().Add(-inactiveCloseTimeout)

	oldEntries := make([]*CachedBlobEntry, 0)
	var oldestEntry *CachedBlobEntry

	for _, be := range mgr.entries {
		if len(be.handles) != 0 {
			continue
		}

		if oldestEntry == nil || be.lastUsed.Before(oldestEntry.lastUsed) {
			oldestEntry = be
		}
		if be.lastUsed.Before(threshold) {
			oldEntries = append(oldEntries, be)
		}
	}

	for _, be := range oldEntries {
		mgr.tryCloseEntry(be)
	}

	if len(mgr.entries) > maxEntries {
		if oldestEntry != nil {
			mgr.tryCloseEntry(oldestEntry)
		}
	}

	if len(mgr.entries) > maxEntries {
		return ENFILE // give up
	}

	return nil
}

type CachedBlobStore struct {
	backendbs BlobStore
	cachebs   RandomAccessBlobStore

	flags int

	mu sync.Mutex

	queryVersion QueryVersionFunc
	beVerCache   map[string]BlobVersion

	entriesmgr CachedBlobEntriesManager
}

const maxEntries = 128

type cacheEntryState int

const (
	cacheEntryUninitialized cacheEntryState = iota
	cacheEntryClean         cacheEntryState = iota
	cacheEntryDirty         cacheEntryState = iota
	cacheEntryClosed        cacheEntryState = iota
)

func (s cacheEntryState) IsActive() bool {
	return s == cacheEntryClean || s == cacheEntryDirty
}

func (s cacheEntryState) String() string {
	switch s {
	case cacheEntryUninitialized:
		return "cacheEntryUninitialized"
	case cacheEntryClean:
		return "cacheEntryClean"
	case cacheEntryDirty:
		return "cacheEntryDirty"
	case cacheEntryClosed:
		return "cacheEntryClosed"
	default:
		return "<unknown cacheEntryState>"
	}
}

type CachedBlobEntry struct {
	mu sync.Mutex

	cbs      *CachedBlobStore
	blobpath string
	cachebh  BlobHandle

	state cacheEntryState

	lastUsed  time.Time
	lastWrite time.Time
	lastSync  time.Time
	syncCount int

	handles map[*CachedBlobHandle]struct{}
}

func (be *CachedBlobEntry) initializeWithLock(cbs *CachedBlobStore) error {
	cachebh, err := cbs.cachebs.Open(be.blobpath, fl.O_RDWRCREATE)
	if err != nil {
		return fmt.Errorf("Failed to open cache blob: %v", err)
	}
	cachever, err := cbs.queryVersion(&OffsetReader{cachebh, 0})
	if err != nil {
		return fmt.Errorf("Failed to query cached blob ver: %v", err)
	}
	backendver, err := cbs.queryBackendVersion(be.blobpath)
	if err != nil {
		return err
	}

	be.cbs = cbs
	be.cachebh = cachebh
	be.handles = make(map[*CachedBlobHandle]struct{})

	if cachever > backendver {
		log.Printf("FIXME: cache is newer than backend when open")
		be.state = cacheEntryDirty
	} else if cachever == backendver {
		// ok
	} else {
		if err := cbs.invalidateCache(be.blobpath); err != nil {
			return err
		}

		// reopen cachebh
		if err := be.cachebh.Close(); err != nil {
			return fmt.Errorf("Failed to close cache blob for re-opening: %v", err)
		}
		var err error
		be.cachebh, err = cbs.cachebs.Open(be.blobpath, fl.O_RDWRCREATE)
		if err != nil {
			return fmt.Errorf("Failed to reopen cache blob: %v", err)
		}
	}

	be.state = cacheEntryClean
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

func (be *CachedBlobEntry) PRead(offset int64, p []byte) error {
	// FIXME: may be we should allow stale reads w/o lock
	be.mu.Lock()
	defer be.mu.Unlock()

	be.lastUsed = time.Now()

	return be.cachebh.PRead(offset, p)
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
		log.Fatalf("markDirty called from unexpected state: %+v", be.infoWithLock())
	}
	be.state = cacheEntryDirty

	if be.lastSync.IsZero() {
		be.lastSync = time.Now()
	}
}

func (be *CachedBlobEntry) PWrite(offset int64, p []byte) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if len(p) == 0 {
		return nil
	}
	be.markDirtyWithLock()
	return be.cachebh.PWrite(offset, p)
}

func (be *CachedBlobEntry) Size() int64 {
	be.mu.Lock()
	defer be.mu.Unlock()

	if !be.state.IsActive() {
		return 0
	}

	return be.cachebh.Size()
}

func (be *CachedBlobEntry) Truncate(newsize int64) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.cachebh.Size() == newsize {
		return nil
	}
	be.markDirtyWithLock()
	return be.cachebh.Truncate(newsize)
}

func (be *CachedBlobEntry) writeBackWithLock() error {
	if be.state != cacheEntryDirty {
		return nil
	}

	cachever, err := be.cbs.queryVersion(&OffsetReader{be.cachebh, 0})
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
	r := io.LimitReader(&OffsetReader{be.cachebh, 0}, be.Size())
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("Failed to copy dirty data to backend blob writer: %v", err)
	}

	be.cbs.beVerCache[be.blobpath] = cachever
	be.state = cacheEntryClean
	return nil
}

var _ = util.Syncer(&CachedBlobEntry{})

func (be *CachedBlobEntry) Sync() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if !be.state.IsActive() {
		log.Printf("Attempted to sync already uninitialized/closed entry: %+v", be.infoWithLock())
		return nil
	}
	if be.state == cacheEntryClean {
		return nil
	}

	log.Printf("Sync entry: %+v", be.infoWithLock())

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

func (be *CachedBlobEntry) Close() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if !be.state.IsActive() {
		log.Printf("Attempted to close uninitialized/already closed entry: %+v", be.infoWithLock())
		return nil
	}

	log.Printf("Close entry: %+v", be.infoWithLock())

	if err := be.writeBackWithLock(); err != nil {
		return fmt.Errorf("Failed to writeback dirty: %v", err)
	}

	if err := be.cachebh.Close(); err != nil {
		return fmt.Errorf("Failed to close cache bh: %v", err)
	}

	be.state = cacheEntryClosed
	be.syncCount++
	be.lastSync = time.Now()
	return nil
}

type CachedBlobEntryInfo struct {
	BlobPath              string    `json:"blobpath"`
	State                 string    `json:"is_closed"`
	SyncCount             int       `json:"sync_count"`
	LastUsed              time.Time `json:"last_used"`
	LastWrite             time.Time `json:"last_write"`
	LastSync              time.Time `json:"last_sync"`
	NumberOfHandles       int       `json:"number_of_handles"`
	NumberOfWriterHandles int       `json:"number_of_writer_handles"`
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
		SyncCount:             be.syncCount,
		LastUsed:              be.lastUsed,
		LastWrite:             be.lastWrite,
		LastSync:              be.lastSync,
		NumberOfHandles:       len(be.handles),
		NumberOfWriterHandles: numWriters,
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

func (cbs *CachedBlobStore) invalidateCache(blobpath string) error {
	backendr, err := cbs.backendbs.OpenReader(blobpath)
	if err != nil {
		return fmt.Errorf("Failed to open backend blob for cache invalidation: %v", err)
	}
	defer func() {
		if err := backendr.Close(); err != nil {
			log.Printf("Failed to close backend blob reader for cache invalidation: %v", err)
		}
	}()

	bs, ok := cbs.cachebs.(BlobStore)
	if !ok {
		return fmt.Errorf("FIXME: only cachebs supporting OpenWriter is currently supported")
	}

	cachew, err := bs.OpenWriter(blobpath)
	defer func() {
		if err := cachew.Close(); err != nil {
			log.Printf("Failed to close cache blob writer for cache invalidation: %v", err)
		}
	}()
	if _, err := io.Copy(cachew, backendr); err != nil {
		return fmt.Errorf("Failed to copy blob from backend: %v", err)
	}

	// FIXME: check integrity here?

	return nil
}

func (cbs *CachedBlobStore) queryBackendVersion(blobpath string) (BlobVersion, error) {
	if ver, ok := cbs.beVerCache[blobpath]; ok {
		log.Printf("return cached ver for \"%s\" -> %d", blobpath, ver)
		return ver, nil
	}

	r, err := cbs.backendbs.OpenReader(blobpath)
	if err != nil {
		if err == ENOENT {
			cbs.beVerCache[blobpath] = 0
			return 0, nil
		}
		return -1, fmt.Errorf("Failed to open backend blob for ver query: %v", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("Failed to close backend blob handle for querying version: %v", err)
		}
	}()
	ver, err := cbs.queryVersion(r)
	if err != nil {
		return -1, fmt.Errorf("Failed to query backend blob ver: %v", err)
	}

	cbs.beVerCache[blobpath] = ver
	return ver, nil
}

func NewCachedBlobStore(backendbs BlobStore, cachebs RandomAccessBlobStore, flags int, queryVersion QueryVersionFunc) (*CachedBlobStore, error) {
	if fl.IsWriteAllowed(flags) {
		if fr, ok := backendbs.(FlagsReader); ok {
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
		flags:        flags,
		queryVersion: queryVersion,
		beVerCache:   make(map[string]BlobVersion),
		entriesmgr:   NewCachedBlobEntriesManager(),
	}
	go cbs.entriesmgr.Run()
	return cbs, nil
}

func (cbs *CachedBlobStore) Flags() int {
	return cbs.flags
}

func (cbs *CachedBlobStore) Open(blobpath string, flags int) (BlobHandle, error) {
	if !fl.IsWriteAllowed(cbs.flags) && fl.IsWriteAllowed(flags) {
		return nil, EPERM
	}

	be, err := cbs.entriesmgr.OpenEntry(blobpath)
	if err != nil {
		return nil, err
	}
	return be.OpenHandle(cbs, flags)
}

func (cbs *CachedBlobStore) DumpEntriesInfo() []*CachedBlobEntryInfo {
	return cbs.entriesmgr.DumpEntriesInfo()
}

type CachedBlobHandle struct {
	be    *CachedBlobEntry
	flags int
}

func (bh *CachedBlobHandle) Flags() int { return bh.flags }

func (bh *CachedBlobHandle) PRead(offset int64, p []byte) error {
	if !fl.IsReadAllowed(bh.flags) {
		return EPERM
	}

	return bh.be.PRead(offset, p)
}

func (bh *CachedBlobHandle) PWrite(offset int64, p []byte) error {
	if !fl.IsWriteAllowed(bh.flags) {
		return EPERM
	}

	return bh.be.PWrite(offset, p)
}

func (bh *CachedBlobHandle) Size() int64 {
	return bh.be.Size()
}

func (bh *CachedBlobHandle) Truncate(newsize int64) error {
	if !fl.IsWriteAllowed(bh.flags) {
		return EPERM
	}

	return bh.be.Truncate(newsize)
}

var _ = util.Syncer(&CachedBlobHandle{})

func (bh *CachedBlobHandle) Sync() error {
	if !fl.IsWriteAllowed(bh.flags) {
		return nil
	}

	return bh.be.Sync()
}

func (bh *CachedBlobHandle) Close() error {
	bh.be.CloseHandle(bh)

	return nil
}

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
	EPERM = syscall.Errno(syscall.EPERM)
)

// FIXME: handle overflows
type BlobVersion int64

type QueryVersionFunc func(r io.Reader) (BlobVersion, error)

type CachedBlobStore struct {
	backendbs BlobStore
	cachebs   RandomAccessBlobStore

	flags int

	mu sync.Mutex

	queryVersion QueryVersionFunc
	beVerCache   map[string]BlobVersion

	entries map[string]*CachedBlobEntry
}

const maxEntries = 512

type CachedBlobEntry struct {
	mu sync.Mutex

	cbs      *CachedBlobStore
	blobpath string
	cachebh  BlobHandle

	isDirty bool

	lastUsed  time.Time
	lastWrite time.Time
	lastSync  time.Time
	syncCount int

	handles map[*CachedBlobHandle]struct{}
}

func (be *CachedBlobEntry) OpenHandle(flags int) *CachedBlobHandle {
	be.mu.Lock()
	defer be.mu.Unlock()

	be.lastUsed = time.Now()

	bh := &CachedBlobHandle{be, flags}
	be.handles[bh] = struct{}{}

	return bh
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

func (be *CachedBlobEntry) markDirty() {
	now := time.Now()
	be.lastUsed = now
	be.lastWrite = now

	if be.isDirty {
		return
	}

	be.isDirty = true
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
	be.markDirty()
	return be.cachebh.PWrite(offset, p)
}

func (be *CachedBlobEntry) Size() int64 {
	return be.cachebh.Size()
}

func (be *CachedBlobEntry) Truncate(newsize int64) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.cachebh.Size() == newsize {
		return nil
	}
	be.markDirty()
	return be.cachebh.Truncate(newsize)
}

func (be *CachedBlobEntry) writeBack() error {
	if !be.isDirty {
		return nil
	}

	be.mu.Lock()
	defer be.mu.Unlock()

	be.syncCount++
	be.lastSync = time.Now()

	cachever, err := be.cbs.queryVersion(&OffsetReader{be.cachebh, 0})
	if err != nil {
		return fmt.Errorf("Failed to query cached blob ver: %v", err)
	}

	w, err := be.cbs.backendbs.OpenWriter(be.blobpath)
	if err != nil {
		return fmt.Errorf("Failed to open backend blob writer: %v", err)
	}
	r := io.LimitReader(&OffsetReader{be.cachebh, 0}, be.Size())
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("Failed to copy dirty data to backend blob writer: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("Failed to close backend blob writer: %v", err)
	}

	be.cbs.beVerCache[be.blobpath] = cachever
	be.isDirty = false
	return nil
}

func (be *CachedBlobEntry) IsDirty() bool { return be.isDirty }

var _ = util.Syncer(&CachedBlobEntry{})

func (be *CachedBlobEntry) Sync() error {
	// don't need to lock be.mu here. writeBack() will take it.

	errC := make(chan error)

	go func() {
		if be.isDirty {
			if err := be.writeBack(); err != nil {
				errC <- fmt.Errorf("Failed to writeback dirty: %v", err)
			} else {
				errC <- nil
			}
		} else {
			errC <- nil
		}
	}()

	go func() {
		if cs, ok := be.cachebh.(util.Syncer); ok {
			if err := cs.Sync(); err != nil {
				errC <- fmt.Errorf("Failed to sync cache: %v", err)
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
	return util.ToErrors(errs)
}

type CachedBlobEntryInfo struct {
	BlobPath              string    `json:"blobpath"`
	IsDirty               bool      `json:"is_dirty"`
	SyncCount             int       `json:"sync_count"`
	LastUsed              time.Time `json:"last_used"`
	LastWrite             time.Time `json:"last_write"`
	LastSync              time.Time `json:"last_sync"`
	NumberOfHandles       int       `json:"number_of_handles"`
	NumberOfWriterHandles int       `json:"number_of_writer_handles"`
}

func (be *CachedBlobEntry) Info() *CachedBlobEntryInfo {
	be.mu.Lock()
	defer be.mu.Unlock()

	numWriters := 0
	for h, _ := range be.handles {
		if fl.IsWriteAllowed(h.Flags()) {
			numWriters++
		}
	}

	return &CachedBlobEntryInfo{
		BlobPath:              be.blobpath,
		IsDirty:               be.isDirty,
		SyncCount:             be.syncCount,
		LastUsed:              be.lastUsed,
		LastWrite:             be.lastWrite,
		LastSync:              be.lastSync,
		NumberOfHandles:       len(be.handles),
		NumberOfWriterHandles: numWriters,
	}
}

func (cbs *CachedBlobStore) Sync() error {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	errs := []error{}
	for blobpath, be := range cbs.entries {
		log.Printf("Sync entry: %+v", be)

		if err := be.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("Failed to sync \"%s\": %v", blobpath, err))
		}
	}
	return util.ToErrors(errs)
}

const (
	syncTimeoutDuration  = 300 * time.Second
	writeTimeoutDuration = 3 * time.Second
)

func (cbs *CachedBlobStore) chooseSyncEntry() *CachedBlobEntry {
	cbs.mu.Lock()
	eu := util.EnsureUnlocker{&cbs.mu}
	defer eu.Unlock()

	// Sync priorities:
	//   1. >300 sec since last sync
	//   2. >3 sec since last write

	now := time.Now()

	var oldestSync, oldestWrite *CachedBlobEntry
	oldestSyncT := now
	oldestWriteT := now

	for _, be := range cbs.entries {
		if !be.IsDirty() {
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

func (cbs *CachedBlobStore) SyncOneEntry() error {
	be := cbs.chooseSyncEntry()
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

	return &CachedBlobStore{
		backendbs:    backendbs,
		cachebs:      cachebs,
		flags:        flags,
		queryVersion: queryVersion,
		beVerCache:   make(map[string]BlobVersion),
		entries:      make(map[string]*CachedBlobEntry),
	}, nil
}

func (cbs *CachedBlobStore) Flags() int {
	return cbs.flags
}

func (cbs *CachedBlobStore) DumpEntriesInfo() []*CachedBlobEntryInfo {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	infos := make([]*CachedBlobEntryInfo, 0, len(cbs.entries))
	for _, be := range cbs.entries {
		infos = append(infos, be.Info())
	}
	return infos
}

func (cbs *CachedBlobStore) openCacheEntry(blobpath string) (*CachedBlobEntry, error) {
	cbs.mu.Lock()
	defer cbs.mu.Unlock() // FIXME: maybe release this earlier

	be, ok := cbs.entries[blobpath]
	if ok {
		return be, nil
	}

	cachebh, err := cbs.cachebs.Open(blobpath, fl.O_RDWRCREATE)
	if err != nil {
		return nil, fmt.Errorf("Failed to open cache blob: %v", err)
	}
	cachever, err := cbs.queryVersion(&OffsetReader{cachebh, 0})
	if err != nil {
		return nil, fmt.Errorf("Failed to query cached blob ver: %v", err)
	}
	backendver, err := cbs.queryBackendVersion(blobpath)
	if err != nil {
		return nil, err
	}

	be = &CachedBlobEntry{
		cbs: cbs, blobpath: blobpath, cachebh: cachebh,
		isDirty: false,
		handles: make(map[*CachedBlobHandle]struct{}),
	}
	if cachever > backendver {
		log.Printf("FIXME: cache is newer than backend when open")
		be.isDirty = true
	} else if cachever == backendver {
		// ok
	} else {
		if err := cbs.invalidateCache(blobpath); err != nil {
			return nil, err
		}

		// reopen cachebh
		if err := be.cachebh.Close(); err != nil {
			return nil, fmt.Errorf("Failed to close cache blob for re-opening: %v", err)
		}
		var err error
		be.cachebh, err = cbs.cachebs.Open(blobpath, fl.O_RDWRCREATE)
		if err != nil {
			return nil, fmt.Errorf("Failed to reopen cache blob: %v", err)
		}
	}

	cbs.entries[blobpath] = be
	return be, nil
}

func (cbs *CachedBlobStore) Open(blobpath string, flags int) (BlobHandle, error) {
	if !fl.IsWriteAllowed(cbs.flags) && fl.IsWriteAllowed(flags) {
		return nil, EPERM
	}

	be, err := cbs.openCacheEntry(blobpath)
	if err != nil {
		return nil, err
	}

	return be.OpenHandle(flags), nil
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

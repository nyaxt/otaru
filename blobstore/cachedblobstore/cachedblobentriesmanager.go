package cachedblobstore

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nyaxt/otaru/util"
)

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

type ListBlobsRequest struct {
	resultC chan []string
}

type RemoveBlobRequest struct {
	blobpath string
	resultC  chan error
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
		case *ListBlobsRequest:
			req := req.(*ListBlobsRequest)
			req.resultC <- mgr.doListBlobs()
		case *RemoveBlobRequest:
			req := req.(*RemoveBlobRequest)
			req.resultC <- mgr.doRemoveBlob(req.blobpath)
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
		if be.state != cacheEntryDirty {
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

func (mgr *CachedBlobEntriesManager) doListBlobs() []string {
	bpaths := make([]string, 0, len(mgr.entries))
	for _, be := range mgr.entries {
		if !be.state.IsActive() {
			continue
		}
		bpaths = append(bpaths, be.blobpath)
	}
	return bpaths
}

func (mgr *CachedBlobEntriesManager) ListBlobs() []string {
	req := &ListBlobsRequest{resultC: make(chan []string)}
	mgr.reqC <- req
	return <-req.resultC
}

func (mgr *CachedBlobEntriesManager) doRemoveBlob(blobpath string) error {
	be, ok := mgr.entries[blobpath]
	if !ok {
		return nil
	}
	if err := be.Close(abandonAndClose); err != nil {
		return fmt.Errorf("Failed to abandon cache entry to be removed \"%s\": %v", be.blobpath, err)
	}

	delete(mgr.entries, blobpath)
	return nil
}

func (mgr *CachedBlobEntriesManager) RemoveBlob(blobpath string) error {
	req := &RemoveBlobRequest{blobpath: blobpath, resultC: make(chan error)}
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

	be = &CachedBlobEntry{
		state:    cacheEntryUninitialized,
		blobpath: blobpath,
		bloblen:  -1,
	}
	be.validlenExtended = sync.NewCond(&be.mu)
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
	if err := be.Close(writebackAndClose); err != nil {
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

package cachedblobstore

import (
	"fmt"
	"time"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

type CachedBlobEntriesManager struct {
	reqC chan func()

	entries map[string]*CachedBlobEntry
}

const maxEntries = 128

func NewCachedBlobEntriesManager() CachedBlobEntriesManager {
	return CachedBlobEntriesManager{
		reqC:    make(chan func()),
		entries: make(map[string]*CachedBlobEntry),
	}
}

func (mgr *CachedBlobEntriesManager) Run() {
	for f := range mgr.reqC {
		f()
	}
}

func (mgr *CachedBlobEntriesManager) SyncAll() (err error) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		errs := []error{}
		for blobpath, be := range mgr.entries {
			if err := be.Sync(); err != nil {
				errs = append(errs, fmt.Errorf("Failed to sync \"%s\": %v", blobpath, err))
			}
		}
		err = util.ToErrors(errs)
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) ChooseSyncEntry() (cbe *CachedBlobEntry) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

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
			cbe = oldestWrite
			return
		}
		if now.Sub(oldestSyncT) > syncTimeoutDuration {
			cbe = oldestSync
			return
		}
		cbe = nil
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) DumpEntriesInfo() (infos []*CachedBlobEntryInfo) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		infos = make([]*CachedBlobEntryInfo, 0, len(mgr.entries))
		for _, be := range mgr.entries {
			infos = append(infos, be.Info())
		}
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) ListBlobs() (bpaths []string) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		bpaths = make([]string, 0, len(mgr.entries))
		for _, be := range mgr.entries {
			if !be.AcceptsIO() {
				continue
			}
			bpaths = append(bpaths, be.blobpath)
		}
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) RemoveBlob(blobpath string) (err error) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)
		be, ok := mgr.entries[blobpath]
		if !ok {
			return
		}
		if err := be.Close(abandonAndClose); err != nil {
			err = fmt.Errorf("Failed to abandon cache entry to be removed \"%s\": %v", be.blobpath, err)
			return
		}

		delete(mgr.entries, blobpath)
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) OpenEntry(blobpath string) (be *CachedBlobEntry, err error) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		var ok bool
		be, ok = mgr.entries[blobpath]
		if ok {
			return
		}

		if err = mgr.closeOldCacheEntriesIfNeeded(); err != nil {
			return
		}

		be = NewCachedBlobEntry(blobpath)
		mgr.entries[blobpath] = be
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) tryCloseEntry(be *CachedBlobEntry) {
	if err := be.Close(writebackAndClose); err != nil {
		logger.Criticalf(mylog, "Failed to close cache entry \"%s\": %v", be.blobpath, err)
		return
	}

	delete(mgr.entries, be.blobpath)
}

const inactiveCloseTimeout = 10 * time.Second

func (mgr *CachedBlobEntriesManager) closeOldCacheEntriesIfNeeded() error {
	if len(mgr.entries) <= maxEntries {
		return nil
	}

	logger.Infof(mylog, "closeOldCacheEntriesIfNeeded started")
	start := time.Now()
	threshold := start.Add(-inactiveCloseTimeout)

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
		logger.Infof(mylog, "closeOldCacheEntriesIfNeeded giving up. Couldn't reduce to <maxEntries. Took %s.", time.Since(start))
		return ENFILE // give up
	}

	logger.Infof(mylog, "closeOldCacheEntriesIfNeeded finished. Took %s.", time.Since(start))
	return nil
}

func (mgr *CachedBlobEntriesManager) DropCacheEntry(blobpath string, blobremover blobstore.BlobRemover) (err error) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		be, ok := mgr.entries[blobpath]
		if ok {
			if err := be.Close(writebackAndClose); err != nil {
				err = fmt.Errorf("Failed to writeback cache entry to be removed \"%s\": %v", blobpath, err)
				return
			}
		}
		if err := blobremover.RemoveBlob(blobpath); err != nil {
			err = fmt.Errorf("Failed to remove cache blob entry from blobstore \"%s\": %v", blobpath, err)
			return
		}

		delete(mgr.entries, blobpath)
	}
	<-ch
	return
}

package cachedblobstore

import (
	"fmt"
	"sort"
	"time"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

type CachedBlobEntriesManager struct {
	reqC  chan func()
	joinC chan struct{}

	entries map[string]*CachedBlobEntry
}

var _ = SyncCandidatesProvider(&CachedBlobEntriesManager{})

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

	for _, be := range mgr.entries {
		if err := be.Close(writebackIfNeededAndClose); err != nil {
			logger.Warningf(mylog, "Failed to close entry %v. Err: %v", be, err)
		}
	}

	mgr.joinC <- struct{}{}
}

func (mgr *CachedBlobEntriesManager) FindAllSyncable() (scs []util.Syncer) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		scs = make([]util.Syncer, 0, maxEntries)
		for _, be := range mgr.entries {
			if !be.state.NeedsSync() {
				continue
			}

			scs = append(scs, be)
		}
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) Quit() {
	mgr.joinC = make(chan struct{})
	close(mgr.reqC)
	<-mgr.joinC
}

type syncCandidatesSorter struct {
	cs             []*CachedBlobEntry
	writeThreshold time.Time
	syncThreshold  time.Time
}

func (s syncCandidatesSorter) Len() int { return len(s.cs) }

func (s syncCandidatesSorter) Swap(i, j int) {
	s.cs[i], s.cs[j] = s.cs[j], s.cs[i]
}

const (
	syncTimeoutDuration  = 300 * time.Second
	writeTimeoutDuration = 3 * time.Second
)

func (s syncCandidatesSorter) Less(i, j int) bool {
	// Sync priorities:
	//   1. >300 sec since last sync
	//   2. >3 sec since last write

	if s.cs[i].LastSync().Before(s.syncThreshold) {
		if s.cs[j].LastSync().Before(s.syncThreshold) {
			return s.cs[i].LastSync().Before(s.cs[j].LastSync())
		} else {
			return true
		}
	} else {
		if s.cs[j].LastSync().Before(s.syncThreshold) {
			return false
		} else {
			return s.cs[i].LastWrite().Before(s.cs[j].LastWrite())
		}
	}
}

func (mgr *CachedBlobEntriesManager) FindSyncCandidates(n int) (scs []util.Syncer) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		now := time.Now()
		writeThreshold := now.Add(-writeTimeoutDuration)
		syncThreshold := now.Add(-syncTimeoutDuration)

		cs := make([]*CachedBlobEntry, 0, len(mgr.entries))

		for _, be := range mgr.entries {
			if !be.state.NeedsSync() {
				continue
			}
			if be.LastWrite().After(writeThreshold) {
				continue
			}
			if be.LastSync().After(syncThreshold) {
				continue
			}

			cs = append(cs, be)
		}

		sort.Sort(syncCandidatesSorter{
			cs:             cs,
			writeThreshold: writeThreshold,
			syncThreshold:  syncThreshold,
		})

		if len(cs) > n {
			cs = cs[:n]
		}
		scs = make([]util.Syncer, 0, len(cs))
		for i := 0; i < len(cs); i++ {
			scs = append(scs, cs[i])
		}
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
			if !be.ShouldBeListed() {
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

func (mgr *CachedBlobEntriesManager) CloseEntryForTesting(blobpath string) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		be, ok := mgr.entries[blobpath]
		if !ok {
			logger.Warningf(mylog, "CloseEntryForTesting \"\" couldn't find any entry to close", blobpath)
			return
		}

		mgr.tryCloseEntry(be)
	}
	<-ch
	return
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
			// FIXME: shouldn't this be abandonAndClose
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

package cachedblobstore

import (
	"fmt"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/logger"
	oprometheus "github.com/nyaxt/otaru/prometheus"
	"github.com/nyaxt/otaru/util"
)

var (
	openEntryHitMiss = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: oprometheus.Namespace,
			Subsystem: promSubsystem,
			Name:      "open_entry_count",
			Help:      "Counts CachedBlobEntriesManager.OpenEntry() existing entry hit/miss.",
		},
		[]string{"hitmiss"})
	openEntryHitCounter  = openEntryHitMiss.WithLabelValues("hit")
	openEntryMissCounter = openEntryHitMiss.WithLabelValues("miss")

	numCacheEntriesGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: oprometheus.Namespace,
		Subsystem: promSubsystem,
		Name:      "num_cache_entries",
		Help:      "Number of CachedBlobEntry(ies) in CachedBlobEntriesManager.",
	})

	entryStateGaugeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(oprometheus.Namespace, promSubsystem, "entry_state_count"),
		"Number of CachedBlobEntry in each states.",
		[]string{"state"}, nil)
)

type CachedBlobEntriesManager struct {
	reqC  chan func()
	joinC chan struct{}

	entries map[string]*CachedBlobEntry
}

type cbeCollector struct {
	cbemgr *CachedBlobEntriesManager
}

var _ = SyncCandidatesProvider(&CachedBlobEntriesManager{})

const maxEntries = 128

func NewCachedBlobEntriesManager() *CachedBlobEntriesManager {
	cbemgr := &CachedBlobEntriesManager{
		reqC:    make(chan func()),
		entries: make(map[string]*CachedBlobEntry),
	}
	prometheus.MustRegister(cbeCollector{cbemgr})

	return cbemgr
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
			if be.LastWrite().After(writeThreshold) && be.LastSync().After(syncThreshold) {
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

func (c cbeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- entryStateGaugeDesc
}

func (c cbeCollector) Collect(ch chan<- prometheus.Metric) {
	var numUninitialized int
	var numInvalidating int
	var numErrored int
	var numErroredClosed int
	var numClean int
	var numWriteInProgress int
	var numWritebackInProgress int
	var numStaleWritebackInProgress int
	var numDirty int
	var numDirtyClosing int
	var numClosing int
	var numClosed int

	chM := make(chan struct{})
	c.cbemgr.reqC <- func() {
		defer close(chM)

		for _, be := range c.cbemgr.entries {
			switch be.state {
			case CacheEntryUninitialized:
				numUninitialized += 1
			case CacheEntryInvalidating:
				numInvalidating += 1
			case CacheEntryErrored:
				numErrored += 1
			case CacheEntryErroredClosed:
				numErroredClosed += 1
			case CacheEntryClean:
				numClean += 1
			case CacheEntryWriteInProgress:
				numWriteInProgress += 1
			case CacheEntryWritebackInProgress:
				numWritebackInProgress += 1
			case CacheEntryStaleWritebackInProgress:
				numStaleWritebackInProgress += 1
			case CacheEntryDirty:
				numDirty += 1
			case CacheEntryDirtyClosing:
				numDirtyClosing += 1
			case CacheEntryClosing:
				numClosing += 1
			case CacheEntryClosed:
				numClosed += 1
			}
		}
	}
	<-chM

	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numUninitialized), "uninitialized")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numInvalidating), "invalidating")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numErrored), "errored")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numErroredClosed), "erroredClosed")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numClean), "clean")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numWriteInProgress), "writeInProgress")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numWritebackInProgress), "writebackInProgress")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numStaleWritebackInProgress), "staleWritebackInProgress")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numDirty), "dirty")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numDirtyClosing), "dirtyClosing")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numClosing), "closing")
	ch <- prometheus.MustNewConstMetric(entryStateGaugeDesc, prometheus.GaugeValue, float64(numClosed), "closed")

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
		mgr.updateNumCacheEntriesGauge()
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) OpenEntry(blobpath string, cbs *CachedBlobStore) (be *CachedBlobEntry, err error) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		var ok bool
		be, ok = mgr.entries[blobpath]
		if ok {
			openEntryHitCounter.Inc()
			return
		}

		if err = mgr.closeOldCacheEntriesIfNeeded(); err != nil {
			return
		}

		be = NewCachedBlobEntry(blobpath, cbs)
		mgr.entries[blobpath] = be
		openEntryMissCounter.Inc()
		mgr.updateNumCacheEntriesGauge()
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
	mgr.updateNumCacheEntriesGauge()
}

func (mgr *CachedBlobEntriesManager) CloseEntryForTesting(blobpath string) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		be, ok := mgr.entries[blobpath]
		if !ok {
			logger.Warningf(mylog, "CloseEntryForTesting %q couldn't find any entry to close", blobpath)
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
		return util.ENFILE // give up
	}

	logger.Infof(mylog, "closeOldCacheEntriesIfNeeded finished. Took %s.", time.Since(start))
	return nil
}

func (mgr *CachedBlobEntriesManager) DropCacheEntry(blobpath string, cbs *CachedBlobStore, blobremover blobstore.BlobRemover) (err error) {
	ch := make(chan struct{})
	mgr.reqC <- func() {
		defer close(ch)

		be, ok := mgr.entries[blobpath]
		if !ok {
			be = NewCachedBlobEntry(blobpath, cbs)
		}
		if err := be.Close(writebackAndClose); err != nil {
			err = fmt.Errorf("Failed to writeback cache entry to be removed \"%s\": %v", blobpath, err)
			return
		}
		if err := blobremover.RemoveBlob(blobpath); err != nil {
			err = fmt.Errorf("Failed to remove cache blob entry from blobstore \"%s\": %v", blobpath, err)
			return
		}

		delete(mgr.entries, blobpath)
		mgr.updateNumCacheEntriesGauge()
	}
	<-ch
	return
}

func (mgr *CachedBlobEntriesManager) updateNumCacheEntriesGauge() {
	numCacheEntriesGauge.Set(float64(len(mgr.entries)))
}

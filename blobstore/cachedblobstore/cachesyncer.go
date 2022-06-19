package cachedblobstore

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/nyaxt/otaru/util"
)

var CacheSyncerGracePeriod = 1 * time.Second // only modified from test
var DisableAutoSyncForTesting = false

const defaultNumWorkers = 10

type SyncCandidatesProvider interface {
	FindSyncCandidates(int) []util.Syncer
}

const (
	isBusy     int32 = 0
	isStarving int32 = 1
)

type syncWorkerCmd struct {
	s    util.Syncer
	errC chan error
}

type syncAllCmd struct {
	ss   []util.Syncer
	errC chan error
}

type CacheSyncer struct {
	provider         SyncCandidatesProvider
	numWorkers       int
	workerIsStarving []int32

	workerC chan syncWorkerCmd

	syncAllC    chan syncAllCmd
	quitC       chan chan struct{}
	joinWorkerG sync.WaitGroup
}

func NewCacheSyncer(provider SyncCandidatesProvider, numWorkers int) *CacheSyncer {
	cs := &CacheSyncer{
		provider:         provider,
		numWorkers:       numWorkers,
		workerIsStarving: make([]int32, numWorkers),

		workerC:  make(chan syncWorkerCmd, numWorkers),
		syncAllC: make(chan syncAllCmd),
		quitC:    make(chan chan struct{}, 1),
	}

	cs.joinWorkerG.Add(cs.numWorkers)
	for i := 0; i < cs.numWorkers; i++ {
		cs.workerIsStarving[i] = isBusy
		go cs.workerMain(i)
	}
	go cs.producerMain()

	return cs
}

func (cs *CacheSyncer) SyncAll(ss []util.Syncer) error {
	errC := make(chan error, 16)
	cs.syncAllC <- syncAllCmd{ss, errC}
	var me error
	for i := 0; i < len(ss); i++ {
		e := <-errC
		me = multierr.Append(me, e)
	}
	return me
}

func (cs *CacheSyncer) Quit() {
	joinC := make(chan struct{})
	// zap.S().Debugf("CacheSyncer Quit start")
	cs.quitC <- joinC
	close(cs.quitC)
	<-joinC
	// zap.S().Debugf("CacheSyncer Quit end")
}

func (cs *CacheSyncer) workerMain(workerId int) {
	cs.workerIsStarving[workerId] = isStarving
	for cmd := range cs.workerC {
		atomic.StoreInt32(&cs.workerIsStarving[workerId], isBusy)
		zap.S().Debugf("Worker[%d] syncing %v", workerId, cmd.s)
		err := cmd.s.Sync()
		if cmd.errC != nil {
			cmd.errC <- err
		}
		if err != nil {
			zap.S().Warnf("Worker[%d] Sync failed %v err: %v", workerId, cmd.s, err)
		}
		zap.S().Debugf("Worker[%d] Sync done %v", workerId, cmd.s)
		atomic.StoreInt32(&cs.workerIsStarving[workerId], isStarving)
	}
	cs.joinWorkerG.Done()
}

func (cs *CacheSyncer) StarvingWorkerCount() int {
	count := 0
	for i := 0; i < cs.numWorkers; i++ {
		if atomic.LoadInt32(&cs.workerIsStarving[i]) == isStarving {
			count++
		}
	}
	return count
}

func (cs *CacheSyncer) producerMain() {
	var quitJoinC chan struct{}

	pendingSyncCmds := make([]syncWorkerCmd, 0, maxEntries)
	for {
		nextQueryC := time.After(CacheSyncerGracePeriod)
		select {
		case joinC := <-cs.quitC:
			// zap.S().Debugf("CacheSyncer quitC Recved")
			if joinC != nil {
				quitJoinC = joinC
			}
			break

		case syncAll := <-cs.syncAllC:
			for i := 0; i < len(syncAll.ss); i++ {
				pendingSyncCmds = append(pendingSyncCmds, syncWorkerCmd{syncAll.ss[i], syncAll.errC})
			}
			break

		case <-nextQueryC:
			break
		}

		starvingWorkerCount := cs.StarvingWorkerCount()
		nfree := starvingWorkerCount
		if nfree == 0 {
			zap.S().Debugf("All %d workers are busy.", cs.numWorkers)
			continue
		}

		// zap.S().Debugf("%d starving workers, %d pendingSyncCmds", nfree, len(pendingSyncCmds))
		nPopPending := util.IntMin(nfree, len(pendingSyncCmds))
		for i := 0; i < nPopPending; i++ {
			cs.workerC <- pendingSyncCmds[i]
		}
		pendingSyncCmds = pendingSyncCmds[nPopPending:]

		cbes := []util.Syncer{}
		if !DisableAutoSyncForTesting {
			cbes = cs.provider.FindSyncCandidates(nfree)
			zap.S().Debugf("Found %d starving workers, found %d pending syncs, found %d sync candidates",
				starvingWorkerCount, nPopPending, len(cbes))
		}
		for _, be := range cbes {
			cs.workerC <- syncWorkerCmd{be, nil}
		}

		// zap.S().Debugf("CacheSyncer quitJoinC non nil? %t", quitJoinC != nil)
		if nPopPending == 0 && len(cbes) == 0 && quitJoinC != nil {
			close(cs.workerC)
			cs.joinWorkerG.Wait()
			quitJoinC <- struct{}{}
			return
		}
	}
}

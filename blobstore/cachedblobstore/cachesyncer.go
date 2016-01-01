package cachedblobstore

import (
	"sync/atomic"
	"time"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

var CacheSyncerGracePeriod = 1 * time.Second // only modified from test

const defaultNumWorkers = 10

type SyncCandidatesProvider interface {
	FindSyncCandidates(int) []util.Syncer
}

const (
	isBusy     int32 = 0
	isStarving int32 = 1
)

type CacheSyncer struct {
	provider         SyncCandidatesProvider
	numWorkers       int
	workerIsStarving []int32

	candidateC chan util.Syncer
	quitC      chan chan struct{}
}

func NewCacheSyncer(provider SyncCandidatesProvider, numWorkers int) *CacheSyncer {
	cs := &CacheSyncer{
		provider:         provider,
		numWorkers:       numWorkers,
		workerIsStarving: make([]int32, numWorkers),

		candidateC: make(chan util.Syncer, numWorkers),
		quitC:      make(chan chan struct{}, 1),
	}

	for i := 0; i < cs.numWorkers; i++ {
		cs.workerIsStarving[i] = isBusy
		go cs.workerMain(i)
	}
	go cs.producerMain()

	return cs
}

func (cs *CacheSyncer) Quit() {
	joinC := make(chan struct{})
	cs.quitC <- joinC
	<-joinC
}

func (cs *CacheSyncer) workerMain(workerId int) {
	cs.workerIsStarving[workerId] = isStarving
	for be := range cs.candidateC {
		atomic.StoreInt32(&cs.workerIsStarving[workerId], isBusy)
		logger.Debugf(mylog, "Worker[%d] syncing %v", workerId, be)
		if err := be.Sync(); err != nil {
			logger.Warningf(mylog, "Worker[%d] Sync %s failed: %v", workerId, be, err)
		}
		atomic.StoreInt32(&cs.workerIsStarving[workerId], isStarving)
	}
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
	for {
		nextQueryC := time.After(CacheSyncerGracePeriod)
		select {
		case joinC := <-cs.quitC:
			joinC <- struct{}{}
			return

		case <-nextQueryC:
			break
		}

		starvingWorkerCount := cs.StarvingWorkerCount()
		if starvingWorkerCount == 0 {
			logger.Debugf(mylog, "All %d workers are busy.", cs.numWorkers)
			continue
		}

		cbes := cs.provider.FindSyncCandidates(starvingWorkerCount)
		logger.Debugf(mylog, "Found %d starving workers, found %d sync candidates",
			starvingWorkerCount, len(cbes))
		if len(cbes) == 0 {
			continue
		}

		for _, be := range cbes {
			cs.candidateC <- be
		}
	}
}

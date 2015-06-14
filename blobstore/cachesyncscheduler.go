package blobstore

import (
	"log"
	"time"
)

type CacheSyncScheduler struct {
	cbs *CachedBlobStore

	lastSync time.Time
	quitC    chan struct{}
}

func NewCacheSyncScheduler(cbs *CachedBlobStore) *CacheSyncScheduler {
	return &CacheSyncScheduler{
		cbs:   cbs,
		quitC: make(chan struct{}),
	}
}

const schedulerWaitDuration = 200 * time.Millisecond

func (ss *CacheSyncScheduler) Run() {
	go func() {
		ticker := time.NewTicker(schedulerWaitDuration)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := ss.cbs.SyncOneEntry()
				if err != ENOENT {
					log.Printf("SyncOneEntry err: %v", err)
				}

			case <-ss.quitC:
				return
			}
		}
	}()
}

func (ss *CacheSyncScheduler) Stop() {
	ss.quitC <- struct{}{}
}

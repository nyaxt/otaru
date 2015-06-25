package blobstore

import (
	"log"
	"time"
)

type CacheSyncScheduler struct {
	cbs   *CachedBlobStore
	quitC chan struct{}
}

func NewCacheSyncScheduler(cbs *CachedBlobStore) *CacheSyncScheduler {
	css := &CacheSyncScheduler{
		cbs:   cbs,
		quitC: make(chan struct{}),
	}
	go css.run()
	return css
}

const schedulerWaitDuration = 200 * time.Millisecond

func (ss *CacheSyncScheduler) run() {
	ticker := time.NewTicker(schedulerWaitDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ss.cbs.SyncOneEntry()
			if err != nil && err != ENOENT {
				log.Printf("SyncOneEntry err: %v", err)
			}

		case <-ss.quitC:
			return
		}
	}
}

func (ss *CacheSyncScheduler) Stop() {
	ss.quitC <- struct{}{}
}

package cachedblobstore

import (
	"log"
	"time"

	"github.com/nyaxt/otaru/util"
)

const schedulerWaitDuration = 200 * time.Millisecond

func NewCacheSyncScheduler(cbs *CachedBlobStore) *util.PeriodicRunner {
	return util.NewPeriodicRunner(func() {
		err := cbs.SyncOneEntry()
		if err != nil && err != ENOENT {
			log.Printf("SyncOneEntry err: %v", err)
		}
	}, schedulerWaitDuration)
}

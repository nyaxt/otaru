package cachedblobstore

import (
	"time"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

const schedulerWaitDuration = 200 * time.Millisecond

func NewCacheSyncScheduler(cbs *CachedBlobStore) *util.PeriodicRunner {
	return util.NewPeriodicRunner(func() {
		err := cbs.SyncOneEntry()
		if err != nil && err != ENOENT {
			logger.Warningf(mylog, "SyncOneEntry err: %v", err)
		}
	}, schedulerWaitDuration)
}

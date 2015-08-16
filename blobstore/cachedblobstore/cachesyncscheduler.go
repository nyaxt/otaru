package cachedblobstore

import (
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/scheduler"
)

const syncPeriod = 1 * time.Second

type SyncCacheBlobStoreTask struct {
	CBS *CachedBlobStore
}

func (t SyncCacheBlobStoreTask) Run(ctx context.Context) scheduler.Result {
	err := t.CBS.SyncOneEntry()
	if err != nil && err != ENOENT {
		logger.Warningf(mylog, "SyncOneEntry err: %v", err)
		return scheduler.ErrorResult{err}
	}
	return scheduler.ErrorResult{nil}
}

func (t SyncCacheBlobStoreTask) ImplName() string { return "SyncCacheBlobStoreTask" }

func SetupCacheSync(cbs *CachedBlobStore, s *scheduler.RepetitiveJobRunner) scheduler.ID {
	return s.RunEveryPeriod(SyncCacheBlobStoreTask{cbs}, syncPeriod)
}

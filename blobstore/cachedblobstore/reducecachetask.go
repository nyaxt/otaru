package cachedblobstore

import (
	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/scheduler"
)

type ReduceCacheTask struct {
	CBS    *CachedBlobStore
	DryRun bool
}

func (t *ReduceCacheTask) Run(ctx context.Context) scheduler.Result {
	err := t.CBS.ReduceCache(ctx, t.DryRun)
	return scheduler.ErrorResult{err}
}

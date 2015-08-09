package cachedblobstore

import (
	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/scheduler"
)

type InvalidateCacheTask struct {
	CBS *CachedBlobStore
	BE  *CachedBlobEntry
}

func (t *InvalidateCacheTask) Run(ctx context.Context) scheduler.Result {
	err := t.BE.invalidateCache(t.CBS)
	return scheduler.ErrorResult{err}
}

package cachedblobstore

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/scheduler"
)

type InvalidateCacheTask struct {
	CBS *CachedBlobStore
	BE  *CachedBlobEntry
}

func (t *InvalidateCacheTask) Run(ctx context.Context) scheduler.Result {
	err := t.BE.invalidateCache(ctx, t.CBS)
	return scheduler.ErrorResult{err}
}

func (*InvalidateCacheTask) ImplName() string { return "InvalidateCacheTask" }

func (t *InvalidateCacheTask) String() string {
	return fmt.Sprintf("InvalidateCacheTask{blobpath: %s}", t.BE.blobpath)
}

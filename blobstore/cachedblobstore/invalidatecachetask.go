package cachedblobstore

import (
	"fmt"

	"context"

	"github.com/nyaxt/otaru/scheduler"
)

type InvalidateCacheTask struct {
	BE *CachedBlobEntry
}

func (t *InvalidateCacheTask) Run(ctx context.Context) scheduler.Result {
	err := t.BE.invalidate(ctx)
	return scheduler.ErrorResult{err}
}

func (*InvalidateCacheTask) ImplName() string { return "InvalidateCacheTask" }

func (t *InvalidateCacheTask) String() string {
	return fmt.Sprintf("InvalidateCacheTask{blobpath: %s}", t.BE.blobpath)
}

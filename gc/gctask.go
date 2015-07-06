package gc

import (
	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/scheduler"
)

type GCTask struct {
	BS     GCableBlobStore
	IDB    inodedb.DBFscker
	DryRun bool
}

func (t *GCTask) Run(ctx context.Context) scheduler.Result {
	err := GC(ctx, t.BS, t.IDB, t.DryRun)
	return scheduler.ErrorResult{err}
}

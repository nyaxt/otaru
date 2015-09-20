package gc

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
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

func (t *GCTask) String() string {
	return fmt.Sprintf("GCTask{%s, %s}", util.TryGetImplName(t.BS), util.TryGetImplName(t.IDB))
}

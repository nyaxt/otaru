package inodedbssgc

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
)

type INodeDBSSGCer interface {
	DeleteOldSnapshots(ctx context.Context, dryRun bool) error
}

type Task struct {
	Impl   INodeDBSSGCer
	DryRun bool
}

func (t *Task) Run(ctx context.Context) scheduler.Result {
	err := t.Impl.DeleteOldSnapshots(ctx, t.DryRun)
	return scheduler.ErrorResult{err}
}

func (t *Task) String() string {
	return fmt.Sprintf("inodedbssgc.Task{%s}", util.Describe(t.Impl))
}

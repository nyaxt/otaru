package inodedbtxloggc

import (
	"fmt"

	"context"

	"github.com/nyaxt/otaru/scheduler"
	"github.com/nyaxt/otaru/util"
)

type Task struct {
	ThresFinder UnneededTxIDThresholdFinder
	LogDeleter  TransactionLogDeleter
	DryRun      bool
}

func (t *Task) Run(ctx context.Context) scheduler.Result {
	err := GC(ctx, t.ThresFinder, t.LogDeleter, t.DryRun)
	return scheduler.ErrorResult{err}
}

func (t *Task) String() string {
	return fmt.Sprintf("inodedbtxloggc.Task{%s, %s}", util.Describe(t.ThresFinder), util.Describe(t.LogDeleter))
}

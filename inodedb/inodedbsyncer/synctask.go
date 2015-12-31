package inodedbsyncer

import (
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/scheduler"
)

type syncTask struct {
	ts inodedb.TriggerSyncer
}

var _ = scheduler.Task(&syncTask{})
var tzero time.Time

func NewSyncTask(ts inodedb.TriggerSyncer) *syncTask {
	return &syncTask{ts: ts}
}

func (st *syncTask) Run(ctx context.Context) scheduler.Result {
	err := <-st.ts.TriggerSync()
	return scheduler.ErrorResult{err}
}

func (st *syncTask) ImplName() string { return "inodedbsyncer.syncTask" }

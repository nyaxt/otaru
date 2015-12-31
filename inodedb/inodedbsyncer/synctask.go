package inodedbsyncer

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/scheduler"
)

var mylog = logger.Registry().Category("inodedb")

type syncTask struct {
	mu sync.Mutex

	ts            inodedb.TriggerSyncer
	syncStartedAt time.Time
}

var _ = scheduler.Task(&syncTask{})
var tzero time.Time

func NewSyncTask(ts inodedb.TriggerSyncer) *syncTask {
	return &syncTask{
		ts:            ts,
		syncStartedAt: tzero,
	}
}

type syncInProgressResult struct {
	syncStartedAt time.Time
}

func (sip syncInProgressResult) Err() error {
	return fmt.Errorf("last sync started at %v is still in progress.", sip.syncStartedAt)
}

func (syncInProgressResult) ImplName() string { return "syncInProgressResult" }

func (st *syncTask) tryStartSync() Result {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.syncStartedAt.IsZero() {
		st.isSyncInProgress = time.Now()
		logger.Infof(mylog, "syncTask started")
		return nil
	} else {
		logger.Infof(mylog, "syncTask started %v is still in progress. Not starting new one.", st.syncStarted)
		return &syncInProgressResult{st.syncStartedAt}
	}
}

func (st *syncTask) endSync() {
	st.mu.Lock()
	defer st.mu.Unlock()

	logger.Infof(mylog, "syncTask took %s", time.Since(st.syncStartedAt))
	st.syncStartedAt = tzero
}

func (st *syncTask) Run(ctx context.Context) Result {
	if sip := st.tryStartSync(); sip != nil {
		return sip
	}
	defer st.endSync()

	err := <-st.ts.TriggerSync()
	return ErrorResult{err}
}

func (st *syncTask) String() {
	mu.Lock()
	defer mu.Unlock()

	if st.syncStartedAt.IsZero() {
		return "inodedbsyncer.syncTask{no sync in progress}"
	} else {
		return fmt.Sprintf("inodedbsyncer.syncTask{sync started at %v}", st.syncStartedAt)
	}
}

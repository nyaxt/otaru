package scheduler_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/scheduler"
)

func TestScheduler_RunAllAndStop(t *testing.T) {
	s := scheduler.NewScheduler()
	s.RunAllAndStop()
}

func TestScheduler_ZeroTime(t *testing.T) {
	if !scheduler.ZeroTime.IsZero() {
		t.Errorf("err")
	}
}

var counter = 0

type HogeResult struct {
	counterSnapshot int
}

func (h HogeResult) Err() error { return nil }

type HogeTask struct{}

func (HogeTask) Run(context.Context) scheduler.Result {
	counter++
	return HogeResult{counter}
}

func TestScheduler_RunTask(t *testing.T) {
	counter = 0

	s := scheduler.NewScheduler()
	s.RunImmediately(HogeTask{}, nil)
	s.RunAllAndStop()

	if counter != 1 {
		t.Errorf("err")
	}
}

func TestScheduler_RunAt(t *testing.T) {
	counter = 0

	start := time.Now()
	s := scheduler.NewScheduler()

	cbjoin := make(chan struct{})
	cb := func(v *scheduler.JobView) {
		if v.State != scheduler.JobFinished {
			t.Errorf("wrong state")
		}
		if v.CreatedAt.Before(start) {
			t.Errorf("wrong CreatedAt")
		}
		if v.ScheduledAt.Before(v.CreatedAt) {
			t.Errorf("wrong ScheduledAt")
		}
		if v.Result.(HogeResult).counterSnapshot != 1 {
			t.Errorf("wrong result")
		}
		cbjoin <- struct{}{}
	}
	id := s.RunAt(HogeTask{}, time.Now().Add(100*time.Millisecond), cb)
	if id != scheduler.ID(1) {
		t.Errorf("first task not ID 1 but %d", id)
	}

	v := s.Query(id)
	if v == nil {
		t.Errorf("id should be queryable immediately after RunAt return")
		return
	}

	if v.ID != id {
		t.Errorf("ID mismatch")
	}
	if v.State != scheduler.JobScheduled {
		t.Errorf("wrong state")
	}
	if v.Result != nil {
		t.Errorf("result non-nil before run")
	}

	s.RunAllAndStop()

	if counter != 1 {
		t.Errorf("err")
	}
	<-cbjoin
}

type LongTaskResult time.Time

func (LongTaskResult) Err() error { return nil }

type LongTask time.Duration

func (lt LongTask) Run(ctx context.Context) scheduler.Result {
	ticker := time.NewTicker(time.Duration(lt))
	defer ticker.Stop()
	select {
	case <-ticker.C:
		return LongTaskResult(time.Now())
	case <-ctx.Done():
		return nil
	}
}

func TestScheduler_AbortTaskBeforeRun(t *testing.T) {
	s := scheduler.NewScheduler()

	cbjoin := make(chan struct{})
	cb := func(v *scheduler.JobView) {
		if v.State != scheduler.JobAborted {
			t.Errorf("wrong State")
		}
		if v.Result != nil {
			t.Errorf("Aborted task has non-nil Result")
		}
		cbjoin <- struct{}{}
	}
	id := s.RunAt(LongTask(time.Second), time.Now().Add(100*time.Millisecond), cb)
	s.Abort(id)

	s.RunAllAndStop()
	<-cbjoin
}

func TestScheduler_AbortTaskDuringRun(t *testing.T) {
	s := scheduler.NewScheduler()

	cbjoin := make(chan struct{})
	cb := func(v *scheduler.JobView) {
		if v.State != scheduler.JobAborted {
			t.Errorf("wrong State")
		}
		if v.Result != nil {
			t.Errorf("Aborted task has non-nil Result")
		}
		cbjoin <- struct{}{}
	}
	id := s.RunImmediately(LongTask(time.Second), cb)
	time.Sleep(50 * time.Millisecond)
	s.Abort(id)

	s.RunAllAndStop()
	<-cbjoin
}

func TestScheduler_AbortTaskAfterRun(t *testing.T) {
	s := scheduler.NewScheduler()

	cbjoin := make(chan struct{})
	cb := func(v *scheduler.JobView) {
		if v.State != scheduler.JobFinished {
			t.Errorf("wrong State")
		}
		if v.Result == nil {
			t.Errorf("Finished task has nil Result")
		}
		cbjoin <- struct{}{}
	}
	id := s.RunImmediately(HogeTask{}, cb)
	<-cbjoin
	s.Abort(id)

	s.RunAllAndStop()
}

func TestScheduler_AbortAllAndStop(t *testing.T) {
	s := scheduler.NewScheduler()

	s.RunImmediately(LongTask(time.Second), nil)
	s.RunAt(LongTask(time.Second), time.Now().Add(300*time.Millisecond), nil)

	start := time.Now()
	s.AbortAllAndStop()
	if time.Since(start) > time.Second {
		t.Errorf("took >1sec to abort all and stop")
	}
}

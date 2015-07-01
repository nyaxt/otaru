package scheduler_test

import (
	"testing"
	"time"

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

func (HogeTask) Run() scheduler.Result {
	counter++
	return HogeResult{counter}
}

func TestScheduler_RunTask(t *testing.T) {
	counter = 0

	s := scheduler.NewScheduler()
	s.RunImmediately(HogeTask{})
	s.RunAllAndStop()

	if counter != 1 {
		t.Errorf("err")
	}
}

func TestScheduler_RunAt(t *testing.T) {
	counter = 0

	s := scheduler.NewScheduler()
	id := s.RunAt(HogeTask{}, time.Now().Add(100*time.Millisecond))
	if id != scheduler.ID(1) {
		t.Errorf("first task not ID 1 but %d", id)
	}

	var v *scheduler.JobView
	for ; v == nil; v = s.Query(id) {
		// retry until we get valid v
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
}

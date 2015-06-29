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
	s.RunAt(HogeTask{}, time.Now().Add(100*time.Millisecond))
	s.RunAllAndStop()

	if counter != 1 {
		t.Errorf("err")
	}
}

package scheduler_test

import (
	"testing"
	"time"

	"github.com/nyaxt/otaru/scheduler"
)

func TestRepetitiveJobRunner_RunEveryPeriod(t *testing.T) {
	s := scheduler.NewScheduler()
	r := scheduler.NewRepetitiveJobRunner(s)

	counter = 0
	rid := r.RunEveryPeriod(HogeTask{}, 50*time.Millisecond)

	if counter != 0 {
		t.Errorf("err")
	}
	time.Sleep(1 * time.Second)

	if counter < 2 {
		t.Errorf("Should have run at least 2 times: counter %d", counter)
	}

	if err := r.Abort(rid); err != nil {
		t.Errorf("Abort err: %v", err)
	}
	ss := counter
	time.Sleep(1 * time.Second)

	if ss != counter {
		t.Errorf("Detected task run after Abort()")
	}
}

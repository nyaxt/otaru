package util

import "time"

type PeriodicRunner struct {
	cb   func()
	wait time.Duration

	forceRunC chan struct{}
	quitC     chan struct{}
}

const schedulerWaitDuration = 30 * time.Second

func NewPeriodicRunner(cb func(), wait time.Duration) *PeriodicRunner {
	ss := &PeriodicRunner{
		cb:        cb,
		wait:      wait,
		forceRunC: make(chan struct{}),
		quitC:     make(chan struct{}),
	}
	go ss.run()
	return ss
}

func (ss *PeriodicRunner) run() {
	ticker := time.NewTicker(ss.wait)
	defer ticker.Stop()

	runAndScheduleNext := func() {
		ticker.Stop()
		ss.cb()
		ticker = time.NewTicker(ss.wait)
	}

	for {
		select {
		case <-ticker.C:
			runAndScheduleNext()
		case <-ss.forceRunC:
			runAndScheduleNext()

		case <-ss.quitC:
			return
		}
	}
}

func (ss *PeriodicRunner) TriggerImmediately() {
	ss.forceRunC <- struct{}{}
}

func (ss *PeriodicRunner) Stop() {
	ss.quitC <- struct{}{}
}

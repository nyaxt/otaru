package util

import (
	"log"
	"time"
)

type SyncScheduler struct {
	s     Syncer
	wait  time.Duration
	quitC chan struct{}
}

const schedulerWaitDuration = 30 * time.Second

func NewSyncScheduler(s Syncer) *SyncScheduler {
	ss := &SyncScheduler{
		s:     s,
		wait:  schedulerWaitDuration,
		quitC: make(chan struct{}),
	}
	go ss.run()
	return ss
}

func (ss *SyncScheduler) SetSyncWaitDuration(d time.Duration) {
	ss.wait = d
}

func (ss *SyncScheduler) run() {
	ticker := time.NewTicker(ss.wait)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ss.s.Sync()
			if err != nil {
				log.Printf("Sync err: %v", err)
			}

		case <-ss.quitC:
			return
		}
	}
}

func (ss *SyncScheduler) Stop() {
	ss.quitC <- struct{}{}
}

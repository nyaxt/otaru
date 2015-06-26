package util

import (
	"log"
	"time"
)

func NewSyncScheduler(s Syncer, wait time.Duration) *PeriodicRunner {
	return NewPeriodicRunner(func() {
		err := s.Sync()
		if err != nil {
			log.Printf("Sync err: %v", err)
		}
	}, wait)
}

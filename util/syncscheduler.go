package util

import (
	"time"

	"github.com/nyaxt/otaru/logger"
)

var synclog = logger.Registry().Category("syncsched")

func NewSyncScheduler(s Syncer, wait time.Duration) *PeriodicRunner {
	return NewPeriodicRunner(func() {
		err := s.Sync()
		if err != nil {
			logger.Warningf(synclog, "Sync err: %v", err)
		}
	}, wait)
}

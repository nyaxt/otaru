package testutils

import (
	"os"
	"sync"

	"github.com/nyaxt/otaru/logger"
)

var ensureLoggerOnce sync.Once
var ObservedCriticalLog bool

func EnsureLogger() {
	ObservedCriticalLog = false

	ensureLoggerOnce.Do(func() {
		logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
		logger.Registry().AddOutput(logger.HandleCritical(func() {
			ObservedCriticalLog = true
		}))
	})
}

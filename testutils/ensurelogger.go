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
		if false {
			logfile, err := os.OpenFile("/tmp/otaru-test.log", os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				panic(err)
			}
			logger.Registry().AddOutput(logger.WriterLogger{logfile})
		}
		logger.Registry().AddOutput(logger.HandleCritical(func() {
			ObservedCriticalLog = true
		}))
	})
}

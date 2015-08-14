package testutils

import (
	"os"
	"sync"

	"github.com/nyaxt/otaru/logger"
)

var ensureLoggerOnce sync.Once

func EnsureLogger() {
	ensureLoggerOnce.Do(func() {
		logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	})
}

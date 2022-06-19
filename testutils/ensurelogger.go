package testutils

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ensureLoggerOnce sync.Once

var ObservedFatalLog bool

type testFatalHook struct{}

var _ zapcore.CheckWriteHook = testFatalHook{}

func (testFatalHook) OnWrite(*zapcore.CheckedEntry, []zap.Field) {
	ObservedFatalLog = true
}

func EnsureLogger() {
	ObservedFatalLog = false

	ensureLoggerOnce.Do(func() {
		logger, err := zap.NewDevelopment(zap.WithFatalHook(testFatalHook{}))
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(logger)
	})
}

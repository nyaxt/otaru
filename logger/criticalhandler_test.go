package logger_test

import (
	"testing"

	"github.com/nyaxt/otaru/logger"
)

func TestHandleCritical(t *testing.T) {
	called := false
	h := logger.HandleCritical(func() { called = true })

	logger.Debugf(h, "debug")
	if called {
		t.Errorf("Shouldn't be triggered from debug msg")
	}
	logger.Criticalf(h, "critical")
	if !called {
		t.Errorf("Should be triggered from debug msg")
	}

	called = false
	logger.Criticalf(h, "critical2")
	if called {
		t.Errorf("Should be triggered only once")
	}
}

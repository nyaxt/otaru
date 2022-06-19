package logger_test

import (
	"testing"

	"github.com/nyaxt/otaru/logger"
	"go.uber.org/zap"
)

func TestHandleCritical(t *testing.T) {
	called := false
	h := logger.HandleCritical(func() { called = true })

	zap.S().Debugf("debug")
	if called {
		t.Errorf("Shouldn't be triggered from debug msg")
	}
	zap.S().Errorf("critical")
	if !called {
		t.Errorf("Should be triggered from debug msg")
	}

	called = false
	zap.S().Errorf("critical2")
	if called {
		t.Errorf("Should be triggered only once")
	}
}

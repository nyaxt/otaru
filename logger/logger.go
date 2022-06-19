package logger

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

type Level int

const (
	Debug    Level = iota // Debugging logs
	Info                  // What is happening right now
	Warning               // Recoverable errors
	Critical              // Non-recoverable errors which will start graceful shutdown
	Panic                 // Non-recoverable errors with immediate crash
)

func (lv Level) String() string {
	switch lv {
	case Debug:
		return "Debug"
	case Info:
		return "Info"
	case Warning:
		return "Warning"
	case Critical:
		return "Critical"
	case Panic:
		return "Panic"
	default:
		return "<unknown>"
	}
}

func (lv Level) Rune() rune {
	switch lv {
	case Debug:
		return 'D'
	case Info:
		return 'I'
	case Warning:
		return 'W'
	case Critical:
		return 'C'
	case Panic:
		return 'P'
	default:
		return '?'
	}
}

type Logger interface {
	Log(lv Level, data map[string]interface{})
	WillAccept(lv Level) bool
}

func genLocation() string {
	const skip = 3
	_, fullpath, line, ok := runtime.Caller(skip)
	if !ok {
		return "<unknown>:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(fullpath), line)

}

func Logf(l Logger, lv Level, format string, v ...interface{}) {
	if l == nil || !l.WillAccept(lv) {
		return
	}

	logstr := fmt.Sprintf(format, v...)
	l.Log(lv, map[string]interface{}{
		"log":      logstr,
		"time":     time.Now(),
		"location": genLocation(),
	})

	if lv >= Panic {
		panic(logstr)
	}
}

package logger

import (
	"sync"
)

type criticalHandler struct {
	cb   func()
	once sync.Once
}

var _ = &criticalHandler{}

func HandleCritical(cb func()) *criticalHandler {
	return &criticalHandler{cb: cb}
}

func (h criticalHandler) Log(lv Level, data map[string]interface{}) {
	if lv != Critical {
		return
	}

	h.once.Do(h.cb)
}

func (h criticalHandler) WillAccept(lv Level) bool { return lv == Critical }

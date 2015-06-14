package util

import (
	"sync"
)

type EnsureUnlocker struct{ L sync.Locker }

func (eu *EnsureUnlocker) Unlock() {
	if eu.L == nil {
		return
	}

	eu.L.Unlock()
	eu.L = nil
}

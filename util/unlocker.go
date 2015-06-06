package util

import (
	"sync"
)

type EnsureUnlocker struct{ lk sync.Locker }

func (eu *EnsureUnlocker) Unlock() {
	if eu.lk == nil {
		return
	}

	eu.lk.Unlock()
	eu.lk = nil
}

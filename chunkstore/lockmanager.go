package chunkstore

import (
	"sync"
)

type LockManager struct {
	mu              sync.Mutex
	cond            *sync.Cond
	lockedBlobPaths map[string]struct{}
}

func NewLockManager() *LockManager {
	lm := &LockManager{
		lockedBlobPaths: make(map[string]struct{}),
	}
	lm.cond = sync.NewCond(&lm.mu)
	return lm
}

func (lm *LockManager) Lock(blobpath string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for {
		if _, locked := lm.lockedBlobPaths[blobpath]; locked {
			lm.cond.Wait()
		} else {
			lm.lockedBlobPaths[blobpath] = struct{}{}
			return
		}
	}
}

func (lm *LockManager) Unlock(blobpath string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	delete(lm.lockedBlobPaths, blobpath)
	lm.cond.Broadcast()
}

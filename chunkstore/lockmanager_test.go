package chunkstore_test

import (
	"sync"
	"testing"

	"github.com/nyaxt/otaru/chunkstore"
)

func TestLockManager(t *testing.T) {
	lm := chunkstore.NewLockManager()
	lm.Lock("hoge")
	lm.Unlock("hoge")
	lm.Lock("hoge")
	lm.Unlock("hoge")
}

func TestLockManager_MultipleKeys(t *testing.T) {
	lm := chunkstore.NewLockManager()
	lm.Lock("1")
	lm.Lock("2")
	lm.Unlock("2")
	lm.Unlock("1")

	lm.Lock("1")
	lm.Lock("2")
	lm.Unlock("1")
	lm.Unlock("2")
}

func TestLockManager_Mutex(t *testing.T) {
	lm := chunkstore.NewLockManager()

	var wg sync.WaitGroup

	a := make([]int, 0, 10)
	work := func(base int) {
		lm.Lock("hoge")
		defer lm.Unlock("hoge")

		for i := 0; i < 10; i++ {
			a = append(a, base+i)
		}

		wg.Done()
	}

	wg.Add(3)
	go work(10)
	go work(20)
	go work(30)
	wg.Wait()

	if len(a) != 30 {
		t.Errorf("skipped some")
	}
	for i := 0; i < 30; i++ {
		if a[i]%10 != i%10 {
			t.Errorf("corrupt!")
			return
		}
	}
}

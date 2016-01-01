package cachedblobstore_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/logger"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

func init() { tu.EnsureLogger() }

var mylog = logger.Registry().Category("cachedbs_test")

var muConcurrency sync.Mutex
var currConcurrency, maxConcurrency int

type syncable struct {
	id        int
	isSynced  bool
	syncDelay time.Duration
}

func (s *syncable) Sync() error {
	logger.Debugf(mylog, "Start sync %d", s.id)
	{
		muConcurrency.Lock()
		currConcurrency++
		if currConcurrency > maxConcurrency {
			maxConcurrency = currConcurrency
		}
		muConcurrency.Unlock()
	}
	time.Sleep(s.syncDelay)
	s.isSynced = true
	{
		muConcurrency.Lock()
		currConcurrency++
		muConcurrency.Unlock()
	}
	logger.Debugf(mylog, "End sync %d", s.id)
	return nil
}

func (s *syncable) String() string {
	return fmt.Sprintf("syncable{%d, %v delay}", s.id, s.syncDelay)
}

type testProvider struct {
	ss []*syncable
}

func (tp *testProvider) FindSyncCandidates(n int) []util.Syncer {
	if n > len(tp.ss) {
		n = len(tp.ss)
	}

	ret := make([]util.Syncer, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, tp.ss[i])
	}

	tp.ss = tp.ss[n:]

	return ret
}

func TestCacheSyncer(t *testing.T) {
	prov := &testProvider{[]*syncable{}}
	numWorker := 4
	cs := cachedblobstore.NewCacheSyncer(prov, numWorker)
	cs.Quit()
}

func TestCacheSyncer_Concurrent(t *testing.T) {
	// set aggressive value for testing
	cachedblobstore.CacheSyncerGracePeriod = 50 * time.Millisecond

	// make rand seq. deterministic for tests
	rand.Seed(1234)

	numSyncables := 16
	ss := make([]*syncable, 0, numSyncables)
	for i := 0; i < numSyncables; i++ {
		ss = append(ss, &syncable{i, false, time.Duration(20+rand.Intn(8)*10) * time.Millisecond})
	}
	prov := &testProvider{ss}

	numWorker := 4
	maxConcurrency = 0
	currConcurrency = 0
	cs := cachedblobstore.NewCacheSyncer(prov, numWorker)
WaitLoop:
	for {
		time.Sleep(50 * time.Millisecond)

		for _, s := range ss {
			if !s.isSynced {
				continue WaitLoop
			}
		}
		break WaitLoop
	}
	logger.Debugf(mylog, "all syncable synced")
	cs.Quit()

	if maxConcurrency < 2 {
		t.Errorf("MaxConcurrency should be >= 2, but got %d", maxConcurrency)
	}
}

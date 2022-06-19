package cachedblobstore_test

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/logger"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
	"go.uber.org/zap"
)

func init() {
	tu.EnsureLogger()

	// set aggressive value for testing
	cachedblobstore.CacheSyncerGracePeriod = 50 * time.Millisecond
}

var mylog = logger.Registry().Category("cachedbs_test")

const numWorker = 4

var muConcurrency sync.Mutex
var currConcurrency, maxConcurrency int

type syncable struct {
	id        int
	isSynced  bool
	syncDelay time.Duration
}

func (s *syncable) Sync() error {
	{
		muConcurrency.Lock()
		currConcurrency++
		if currConcurrency > maxConcurrency {
			maxConcurrency = currConcurrency
		}
		muConcurrency.Unlock()
	}
	zap.S().Debugf("Start sync %d currConcurrency %d", s.id, currConcurrency)
	time.Sleep(s.syncDelay)
	s.isSynced = true
	zap.S().Debugf("End sync %d", s.id)
	{
		muConcurrency.Lock()
		currConcurrency--
		muConcurrency.Unlock()
	}
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

func TestCacheSyncer_Quit(t *testing.T) {
	prov := &testProvider{[]*syncable{}}
	cs := cachedblobstore.NewCacheSyncer(prov, numWorker)
	cs.Quit()
}

func TestCacheSyncer_Concurrent(t *testing.T) {
	// make rand seq. deterministic for tests
	rand.Seed(1234)

	numSyncables := 16
	ss := make([]*syncable, 0, numSyncables)
	for i := 0; i < numSyncables; i++ {
		ss = append(ss, &syncable{i, false, time.Duration(20+rand.Intn(8)*10) * time.Millisecond})
	}
	prov := &testProvider{ss}

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
	zap.S().Debugf("all syncable synced")
	cs.Quit()

	if maxConcurrency < 2 {
		t.Errorf("MaxConcurrency should be >= 2, but got %d", maxConcurrency)
	}
}

type syncerr struct{ e error }

func (s syncerr) Sync() error { return s.e }

func TestCacheSyncer_SyncAll(t *testing.T) {
	testE := errors.New("hogeErr")

	prov := &testProvider{[]*syncable{}}
	cs := cachedblobstore.NewCacheSyncer(prov, numWorker)

	ss := []util.Syncer{
		syncerr{testE},
		syncerr{nil},
	}
	err := cs.SyncAll(ss)
	if err != testE {
		t.Errorf("unexpected err: %v", err)
	}

	cs.Quit()
}

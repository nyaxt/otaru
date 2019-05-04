package cachedblobstore_test

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/util"
)

func init() {
	cachedblobstore.PrometheusRegisterer = nil
}

func TestCachedBlobEntriesManager_RunQuit(t *testing.T) {
	mgr := cachedblobstore.NewCachedBlobEntriesManager()
	go mgr.Run()
	mgr.Quit()
}

func syncable2blobpaths(scs []util.Syncer) []string {
	bps := make([]string, 0, len(scs))
	for _, s := range scs {
		bp := s.(*cachedblobstore.CachedBlobEntry).BlobPath()
		bps = append(bps, bp)
	}
	return bps
}

func TestCachedBlobEntriesManager_FindSync(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	mgr := cachedblobstore.NewCachedBlobEntriesManager()
	go mgr.Run()
	// defer mgr.Quit()

	now := time.Now()
	var tzero time.Time

	{
		be, err := mgr.OpenEntry("dirtyRecentWrite", nil)
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now, now)
	}
	{
		be, err := mgr.OpenEntry("dirtyRecentSync", nil)
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now, now.Add(-180*time.Second))
	}
	{
		be, err := mgr.OpenEntry("dirtySuperOld", nil)
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, tzero, tzero)
	}
	{
		be, err := mgr.OpenEntry("writeInProgressOldSync", nil)
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now, now.Add(-330*time.Second))
	}
	{
		be, err := mgr.OpenEntry("dirtyWrite5", nil)
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now.Add(-5*time.Second), now.Add(-5*time.Second))
	}
	{
		be, err := mgr.OpenEntry("writebackInProgress", nil)
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryWritebackInProgress, now.Add(-330*time.Second), now.Add(-330*time.Second))
	}

	{
		scs := mgr.FindSyncCandidates(10)
		expected := []string{"dirtySuperOld", "writeInProgressOldSync", "dirtyWrite5"}
		bps := syncable2blobpaths(scs)
		if !reflect.DeepEqual(expected, bps) {
			t.Errorf("FindSyncCandidates(10) Expected: %+v, Actual: %+v", expected, bps)
		}
	}
	{
		scs := mgr.FindSyncCandidates(1)
		expected := []string{"dirtySuperOld"}
		bps := syncable2blobpaths(scs)
		if !reflect.DeepEqual(expected, bps) {
			t.Errorf("FindSyncCandidates(1) Expected: %+v, Actual: %+v", expected, bps)
		}
	}
	{
		scs := mgr.FindAllSyncable()
		expected := []string{"dirtySuperOld", "writeInProgressOldSync", "dirtyWrite5", "dirtyRecentWrite", "dirtyRecentSync"}
		sort.Strings(expected)
		bps := syncable2blobpaths(scs)
		sort.Strings(bps)
		if !reflect.DeepEqual(expected, bps) {
			t.Errorf("FindAllSyncable Expected: %+v, Actual: %+v", expected, bps)
		}
	}
}

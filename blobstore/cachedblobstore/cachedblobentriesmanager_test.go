package cachedblobstore_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
)

func TestCachedBlobEntriesManager_RunQuit(t *testing.T) {
	mgr := cachedblobstore.NewCachedBlobEntriesManager()
	go mgr.Run()
	mgr.Quit()
}

func TestCachedBlobEntriesManager_FindAllSyncable(t *testing.T) {
	cachedblobstore.DisableAutoSyncForTesting = true
	defer func() { cachedblobstore.DisableAutoSyncForTesting = false }()

	mgr := cachedblobstore.NewCachedBlobEntriesManager()
	go mgr.Run()
	// defer mgr.Quit()

	now := time.Now()
	var tzero time.Time

	{
		be, err := mgr.OpenEntry("dirtyRecentWrite")
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now, now)
	}
	{
		be, err := mgr.OpenEntry("dirtyRecentSync")
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now, now.Add(-180*time.Second))
	}
	{
		be, err := mgr.OpenEntry("dirtySuperOld")
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, tzero, tzero)
	}
	{
		be, err := mgr.OpenEntry("writeInProgressOldSync")
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now, now.Add(-330*time.Second))
	}
	{
		be, err := mgr.OpenEntry("dirtyWrite5")
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryDirty, now.Add(-5*time.Second), now.Add(-5*time.Second))
	}
	{
		be, err := mgr.OpenEntry("writebackInProgress")
		if err != nil {
			t.Errorf("OpenEntry err: %v", err)
			return
		}
		be.InitializeForTesting(cachedblobstore.CacheEntryWritebackInProgress, now.Add(-330*time.Second), now.Add(-330*time.Second))
	}

	scs := mgr.FindSyncCandidates(10)
	bps := make([]string, 0, len(scs))
	for _, s := range scs {
		bp := s.(*cachedblobstore.CachedBlobEntry).BlobPath()
		bps = append(bps, bp)
	}
	expected := []string{}
	if !reflect.DeepEqual(expected, bps) {
		t.Errorf("Expected: %+v, Actual: %+v", expected, bps)
	}
}

package cachedblobstore_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	"github.com/nyaxt/otaru/flags"
	tu "github.com/nyaxt/otaru/testutils"
)

func TestCachedBlobStore(t *testing.T) {
	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")

	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	bs, err := cachedblobstore.New(backendbs, cachebs, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	if err := tu.AssertBlobVersion(backendbs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}
	// assert cache not yet filled
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 0); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersionRA(bs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	// assert cache fill
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 5); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := tu.WriteVersionedBlobRA(bs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}

	if err := tu.AssertBlobVersionRA(bs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(cachebs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(backendbs, "backendonly", 10); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestCachedBlobStore_Invalidate(t *testing.T) {
	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")

	if err := tu.WriteVersionedBlob(cachebs, "backendnewer", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(backendbs, "backendnewer", 3); err != nil {
		t.Errorf("%v", err)
		return
	}

	bs, err := cachedblobstore.New(backendbs, cachebs, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}
	if err := tu.AssertBlobVersionRA(bs, "backendnewer", 3); err != nil {
		t.Errorf("%v", err)
		return
	}

	// assert cache fill
	if err := tu.AssertBlobVersion(cachebs, "backendnewer", 3); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := tu.WriteVersionedBlobRA(bs, "backendnewer", 4); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := tu.AssertBlobVersionRA(bs, "backendnewer", 4); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}
	if err := tu.AssertBlobVersion(cachebs, "backendnewer", 4); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(backendbs, "backendnewer", 4); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestCachedBlobStore_NewEntry(t *testing.T) {
	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")

	bs, err := cachedblobstore.New(backendbs, cachebs, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}

	if err := tu.WriteVersionedBlobRA(bs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersionRA(bs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}
	if err := tu.AssertBlobVersion(cachebs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.AssertBlobVersion(backendbs, "newentry", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestCachedBlobStore_AutoExpandLen(t *testing.T) {
	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")

	bs, err := cachedblobstore.New(backendbs, cachebs, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}

	bh, err := bs.Open("hoge", flags.O_RDWRCREATE)
	if err != nil {
		t.Errorf("Failed to open blobhandle")
	}
	defer bh.Close()

	if size := bh.Size(); size != 0 {
		t.Errorf("New bh size non-zero: %d", size)
	}

	if err := bh.PWrite([]byte("Hello"), 0); err != nil {
		t.Errorf("PWrite failed: %v", err)
	}

	if size := bh.Size(); size != 5 {
		t.Errorf("bh size not auto expanded! size  %d", size)
	}
}

func TestCachedBlobStore_ListBlobs(t *testing.T) {
	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")

	bs, err := cachedblobstore.New(backendbs, cachebs, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}

	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(cachebs, "cacheonly", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "synced", 3); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "unsynced", 4); err != nil {
		t.Errorf("%v", err)
		return
	}

	bpaths, err := bs.ListBlobs()
	if err != nil {
		t.Errorf("ListBlobs failed: %v", err)
		return
	}
	sort.Strings(bpaths)
	if !reflect.DeepEqual([]string{"backendonly", "synced", "unsynced"}, bpaths) {
		t.Errorf("ListBlobs returned unexpected result: %v", bpaths)
	}
}

func TestCachedBlobStore_RemoveBlob(t *testing.T) {
	backendbs := tu.TestFileBlobStoreOfName("backend")
	cachebs := tu.TestFileBlobStoreOfName("cache")

	bs, err := cachedblobstore.New(backendbs, cachebs, flags.O_RDWRCREATE, tu.TestQueryVersion)
	if err != nil {
		t.Errorf("Failed to create CachedBlobStore: %v", err)
		return
	}

	if err := tu.WriteVersionedBlob(backendbs, "backendonly", 1); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(cachebs, "cacheonly", 2); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "synced", 3); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := bs.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
		return
	}
	if err := tu.WriteVersionedBlobRA(bs, "unsynced", 4); err != nil {
		t.Errorf("%v", err)
		return
	}

	if err := bs.RemoveBlob("backendonly"); err != nil {
		t.Errorf("RemoveBlob failed: %v", err)
		return
	}
	if err := bs.RemoveBlob("synced"); err != nil {
		t.Errorf("RemoveBlob failed: %v", err)
		return
	}
	if err := bs.RemoveBlob("unsynced"); err != nil {
		t.Errorf("RemoveBlob failed: %v", err)
		return
	}

	bpaths, err := bs.ListBlobs()
	if err != nil {
		t.Errorf("ListBlobs failed: %v", err)
		return
	}
	if len(bpaths) > 0 {
		t.Errorf("Left over blobs: %v", bpaths)
	}

	for _, bp := range []string{"backendonly", "synced", "unsynced"} {
		if err := tu.AssertBlobVersionRA(bs, bp, 0); err != nil {
			t.Errorf("left over blob in bs: %s", bp)
		}
		if err := tu.AssertBlobVersion(cachebs, bp, 0); err != nil {
			t.Errorf("left over blob in cachebs: %s", bp)
		}
		if err := tu.AssertBlobVersion(backendbs, bp, 0); err != nil {
			t.Errorf("left over blob in backendbs: %s", bp)
		}
	}
}

package blobstore_test

import (
	"os"
	"path"
	"reflect"
	"sort"
	"testing"

	tu "github.com/nyaxt/otaru/testutils"
)

func TestFileBlobStore_ListBlobs(t *testing.T) {
	bs := tu.TestFileBlobStoreOfName("filebstest_list")

	if err := tu.WriteVersionedBlob(bs, "hoge", 5); err != nil {
		t.Errorf("Failed to write blob: %v", err)
		return
	}
	if err := tu.WriteVersionedBlob(bs, "fuga", 5); err != nil {
		t.Errorf("Failed to write blob: %v", err)
		return
	}
	if err := os.Mkdir(path.Join(bs.GetBase(), "piyo"), 0755); err != nil {
		t.Errorf("Failed to mkdir dummy blob: %v", err)
		return
	}

	blobs, err := bs.ListBlobs()
	if err != nil {
		t.Errorf("ListBlobs failed: %v", err)
		return
	}
	sort.Strings(blobs)
	if !reflect.DeepEqual(blobs, []string{"fuga", "hoge"}) {
		t.Errorf("ListBlobs wrong result: %v", blobs)
	}
}

func TestFileBlobStore_RemoveBlob(t *testing.T) {
	bs := tu.TestFileBlobStoreOfName("filebstest_rm")

	if err := tu.WriteVersionedBlob(bs, "hoge", 5); err != nil {
		t.Errorf("Failed to write blob: %v", err)
		return
	}
	if err := tu.AssertBlobVersion(bs, "hoge", 5); err != nil {
		t.Errorf("Failed to read back blob: %v", err)
		return
	}
	if err := tu.WriteVersionedBlob(bs, "safe", 3); err != nil {
		t.Errorf("Failed to write blob: %v", err)
		return
	}

	if err := bs.RemoveBlob("hoge"); err != nil {
		t.Errorf("Failed to remove blob: %v", err)
	}

	blobs, err := bs.ListBlobs()
	if err != nil {
		t.Errorf("ListBlobs failed: %v", err)
		return
	}
	if !reflect.DeepEqual(blobs, []string{"safe"}) {
		t.Errorf("ListBlobs wrong result: %v", blobs)
	}

	if _, err := bs.OpenReader("hoge"); err == nil {
		t.Errorf("Open removed file succeeded???")
	}
}

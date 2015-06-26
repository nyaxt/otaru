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
	bs := tu.TestFileBlobStoreOfName("filebstest")

	if err := tu.WriteVersionedBlob(bs, "hoge", 5); err != nil {
		t.Errorf("Failed to write to bs: %v", err)
		return
	}
	if err := tu.WriteVersionedBlob(bs, "fuga", 5); err != nil {
		t.Errorf("Failed to write to bs: %v", err)
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

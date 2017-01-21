package blobstore_test

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/flags"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

func TestFileBlobStore_MultiPRead(t *testing.T) {
	bs := tu.TestFileBlobStoreOfName("filebstest_multipread")

	w, err := bs.OpenWriter("hoge")
	if err != nil {
		t.Errorf("Failed to open writer: %v", err)
		return
	}
	for i := byte(0); i < 128; i++ {
		if _, err := w.Write([]byte{i}); err != nil {
			t.Errorf("Failed to write: %v", err)
			return
		}
	}
	if err := w.Close(); err != nil {
		t.Errorf("Failed to close: %v", err)
		return
	}

	r, err := bs.OpenReader("hoge")
	if err != nil {
		t.Errorf("Failed to open reader: %v", err)
		return
	}
	r2, err := bs.OpenReader("hoge")
	if err != nil {
		t.Errorf("Failed to open reader: %v", err)
		return
	}
	buf := make([]byte, 1)
	for i := byte(0); i < 64; i++ {
		if _, err := r.Read(buf); err != nil {
			t.Errorf("Failed to read: %v", err)
			return
		}
		if buf[0] != i {
			t.Errorf("mismatch! read %d, expected %d", buf[0], i)
		}
	}
	for i := byte(0); i < 64; i++ {
		if _, err := r2.Read(buf); err != nil {
			t.Errorf("Failed to read: %v", err)
			return
		}
		if buf[0] != i {
			t.Errorf("mismatch! read %d, expected %d", buf[0], i)
		}
	}
	for i := byte(64); i < 128; i++ {
		if _, err := r.Read(buf); err != nil {
			t.Errorf("Failed to read: %v", err)
			return
		}
		if buf[0] != i {
			t.Errorf("mismatch! read %d, expected %d", buf[0], i)
		}
	}

	if err := r.Close(); err != nil {
		t.Errorf("Failed to close: %v", err)
		return
	}
	if err := r2.Close(); err != nil {
		t.Errorf("Failed to close: %v", err)
		return
	}
}

func TestFileBlobStore_TotalSize(t *testing.T) {
	bs := tu.TestFileBlobStoreOfName("filebstest_totalsize")

	w, err := bs.OpenWriter("hoge")
	if err != nil {
		t.Errorf("Failed to open writer: %v", err)
		return
	}
	if _, err := w.Write([]byte("01234567")); err != nil {
		t.Errorf("Failed to write: %v", err)
		return
	}
	if err := w.Close(); err != nil {
		t.Errorf("Failed to close: %v", err)
		return
	}

	n, err := bs.TotalSize()
	if err != nil {
		t.Errorf("TotalSize() err: %v", err)
	}
	if n != 8 {
		t.Errorf("TotalSize returned unexpected result: %v", n)
	}
}

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

func TestFileBlobStore_ReadOnly(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "blobstoretest_ro")
	if err != nil {
		t.Errorf("failed to create tmpdir: %v", err)
		return
	}
	wbs, err := blobstore.NewFileBlobStore(tempdir, flags.O_RDWRCREATE)
	if err != nil {
		t.Errorf("failed to create wbs: %v", err)
		return
	}
	if err := tu.WriteVersionedBlob(wbs, "hoge", 5); err != nil {
		t.Errorf("Failed to write blob: %v", err)
		return
	}
	if err := tu.WriteVersionedBlob(wbs, "safe", 3); err != nil {
		t.Errorf("Failed to write blob: %v", err)
		return
	}

	rbs, err := blobstore.NewFileBlobStore(tempdir, flags.O_RDONLY)
	if err != nil {
		t.Errorf("failed to create rbs: %v", err)
		return
	}

	err = rbs.RemoveBlob("hoge")
	if err == nil {
		t.Errorf("Unexpected RemoveBlob success.")
		return
	}
	if err != util.EPERM {
		t.Errorf("Expected EPERM. got %v", err)
		return
	}

	blobs, err := rbs.ListBlobs()
	if err != nil {
		t.Errorf("ListBlobs failed: %v", err)
		return
	}
	sort.Strings(blobs)
	if !reflect.DeepEqual(blobs, []string{"hoge", "safe"}) {
		t.Errorf("ListBlobs wrong result: %v", blobs)
	}

	if _, err := rbs.OpenReader("hoge"); err != nil {
		t.Errorf("Open remove failed file success???")
	}
	if _, err := wbs.OpenReader("hoge"); err != nil {
		t.Errorf("Open remove failed file success???")
	}

	_, err = rbs.OpenWriter("new")
	if err == nil {
		t.Errorf("Unexpected OpenWriter on ro fbs success")
	}
	if err != util.EPERM {
		t.Errorf("Expected EPERM. got %v", err)
		return
	}
	_, err = rbs.OpenWriter("safe")
	if err == nil {
		t.Errorf("Unexpected OpenWriter on ro fbs success")
	}
	if err != util.EPERM {
		t.Errorf("Expected EPERM. got %v", err)
		return
	}
}

package gcs_test

import (
	"bytes"
	"io"
	"log"
	"reflect"
	"sort"
	"testing"

	"github.com/nyaxt/otaru/flags"
	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/gcs"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

func testGCSBlobStore(f int) *gcs.GCSBlobStore {
	bs, err := gcs.NewGCSBlobStore(
		authtu.TestConfig().ProjectName,
		authtu.TestBucketName(),
		authtu.TestTokenSource(),
		f,
	)
	if err != nil {
		log.Fatalf("Failed to create GCSBlobStore: %v", err)
	}
	return bs
}

func TestGCSBlobStore_WriteReadDelete(t *testing.T) {
	bs := testGCSBlobStore(flags.O_RDWR)

	// Write
	{
		w, err := bs.OpenWriter("hoge")
		if err != nil {
			t.Errorf("Failed to open writer: %v", err)
			return
		}

		n, err := w.Write(tu.HelloWorld)
		if err != nil {
			t.Errorf("Write failed: %v", err)
			return
		}
		if n != len(tu.HelloWorld) {
			t.Errorf("Write returned unexpected len: %d", n)
			return
		}
		if err := w.Close(); err != nil {
			t.Errorf("Failed to close writer: %v", err)
			return
		}
	}

	// Read
	{
		r, err := bs.OpenReader("hoge")
		if err != nil {
			t.Errorf("Failed to open reader: %v", err)
			return
		}

		buf := make([]byte, len(tu.HelloWorld))
		if _, err = io.ReadFull(r, buf); err != nil {
			t.Errorf("ReadFull failed: %v", err)
			return
		}
		if !bytes.Equal(tu.HelloWorld, buf) {
			t.Errorf("Read content != Write content")
		}

		if err := r.Close(); err != nil {
			t.Errorf("Failed to close reader: %v", err)
			return
		}
	}

	// ListBlobs
	{
		bpaths, err := bs.ListBlobs()
		if err != nil {
			t.Errorf("Failed to ListBlobs(): %v", err)
			return
		}

		if !reflect.DeepEqual([]string{"hoge"}, bpaths) {
			t.Errorf("Unexpected BlobList: %v", bpaths)
		}
	}

	// Delete
	if err := bs.RemoveBlob("hoge"); err != nil {
		t.Errorf("Failed to remove blob: %v", err)
	}
}

func TestGCSBlobStore_ReadOnly(t *testing.T) {
	wbs := testGCSBlobStore(flags.O_RDWR)
	defer func() {
		_ = wbs.RemoveBlob("hoge")
		_ = wbs.RemoveBlob("fuga")
	}()
	if err := tu.WriteVersionedBlob(wbs, "hoge", 42); err != nil {
		t.Errorf("%v", err)
		return
	}
	if err := tu.WriteVersionedBlob(wbs, "fuga", 123); err != nil {
		t.Errorf("%v", err)
		return
	}

	// Read should work just fine
	rbs := testGCSBlobStore(flags.O_RDONLY)
	if err := tu.AssertBlobVersion(rbs, "hoge", 42); err != nil {
		t.Errorf("assert hoge 42. err: %v", err)
	}
	if err := tu.AssertBlobVersion(rbs, "fuga", 123); err != nil {
		t.Errorf("assert fuga 123. err: %v", err)
	}

	// Delete should fail
	err := rbs.RemoveBlob("hoge")
	if err == nil {
		t.Errorf("Unexpected RemoveBlob success.")
		return
	}
	if err != util.EACCES {
		t.Errorf("Expected EACCES. got %v", err)
		return
	}

	// Write should fail
	_, err = rbs.OpenWriter("new")
	if err == nil {
		t.Errorf("Unexpected OpenWriter success.")
		return
	}
	if err != util.EACCES {
		t.Errorf("Expected EACCES. got %v", err)
		return
	}

	// ListBlobs should work
	{
		bpaths, err := rbs.ListBlobs()
		if err != nil {
			t.Errorf("Failed to ListBlobs(): %v", err)
			return
		}
		sort.Strings(bpaths)
		if !reflect.DeepEqual([]string{"fuga", "hoge"}, bpaths) {
			t.Errorf("Unexpected BlobList: %v", bpaths)
		}
	}
}

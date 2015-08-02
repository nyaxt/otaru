package gcs_test

import (
	"bytes"
	"io"
	"log"
	"reflect"
	"testing"

	"github.com/nyaxt/otaru/flags"
	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/gcs"
	tu "github.com/nyaxt/otaru/testutils"
)

func testGCSBlobStore() *gcs.GCSBlobStore {
	bs, err := gcs.NewGCSBlobStore(
		authtu.TestConfig().ProjectName,
		authtu.TestBucketName(),
		authtu.TestTokenSource(),
		flags.O_RDWR,
	)
	if err != nil {
		log.Fatalf("Failed to create GCSBlobStore: %v", err)
	}
	return bs
}

func TestGCSBlobStore_WriteReadDelete(t *testing.T) {
	bs := testGCSBlobStore()

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

package gcs_test

import (
	"bytes"
	"io"
	"log"
	"os"
	"path"
	"testing"

	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/gcs"
	. "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

func testClientSource() auth.ClientSource {
	homedir := os.Getenv("HOME")
	clisrc, err := auth.GetGCloudClientSource(
		path.Join(homedir, ".otaru", "credentials.json"),
		path.Join(homedir, ".otaru", "tokencache.json"),
		false)
	if err != nil {
		log.Fatalf("Failed to create testClientSource: %v", err)
	}
	return clisrc
}

func testGCSBlobStore() *gcs.GCSBlobStore {
	homedir := os.Getenv("HOME")
	projectName := util.StringFromFileOrDie(path.Join(homedir, ".otaru", "projectname.txt"), "projectName")
	bs, err := gcs.NewGCSBlobStore(
		projectName,
		"otaru-test",
		testClientSource(),
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

		n, err := w.Write(HelloWorld)
		if err != nil {
			t.Errorf("Write failed: %v", err)
			return
		}
		if n != len(HelloWorld) {
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

		buf := make([]byte, len(HelloWorld))
		if _, err = io.ReadFull(r, buf); err != nil {
			t.Errorf("ReadFull failed: %v", err)
			return
		}
		if !bytes.Equal(HelloWorld, buf) {
			t.Errorf("Read content != Write content")
		}

		if err := r.Close(); err != nil {
			t.Errorf("Failed to close reader: %v", err)
			return
		}
	}

	// Delete
	if err := bs.RemoveBlob("hoge"); err != nil {
		t.Errorf("Failed to remove blob: %v", err)
	}
}

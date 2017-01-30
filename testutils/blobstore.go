package testutils

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/blobstore/version"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/util"
)

func TestFileBlobStore() *blobstore.FileBlobStore {
	return TestFileBlobStoreOfName("")
}

func TestFileBlobStoreOfName(name string) *blobstore.FileBlobStore {
	tempdir, err := ioutil.TempDir("", fmt.Sprintf("blobstoretest%s", name))
	if err != nil {
		log.Fatalf("failed to create tmpdir: %v", err)
	}
	fbs, err := blobstore.NewFileBlobStore(tempdir, flags.O_RDWRCREATE)
	if err != nil {
		log.Fatalf("failed to create blobstore: %v", err)
	}
	return fbs
}

func TestQueryVersion(r io.Reader) (version.Version, error) {
	b := make([]byte, 1)
	if _, err := r.Read(b); err != nil {
		if err == io.EOF {
			// no data -> ver 0.
			return 0, nil
		}
		return -1, fmt.Errorf("Failed to read 1 byte: %v", err)
	}

	return version.Version(b[0]), nil
}

func AssertBlobVersion(bs blobstore.BlobStore, blobpath string, expected version.Version) error {
	r, err := bs.OpenReader(blobpath)
	if err != nil {
		if expected == 0 && util.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("Failed to open reader: %v", err)
	}
	actual, err := TestQueryVersion(r)
	if err != nil {
		return fmt.Errorf("Failed to query version: %v", err)
	}
	if err := r.Close(); err != nil {
		return fmt.Errorf("Failed to close reader: %v", err)
	}

	if actual != expected {
		return fmt.Errorf("Expected version %d, but got %d", expected, actual)
	}

	return nil
}

func AssertBlobVersionRA(bs blobstore.RandomAccessBlobStore, blobpath string, expected version.Version) error {
	h, err := bs.Open(blobpath, flags.O_RDONLY)
	if err != nil {
		return fmt.Errorf("Failed to open reader: %v", err)
	}
	actual, err := TestQueryVersion(&blobstore.OffsetReader{h, 0})
	if err != nil {
		return fmt.Errorf("Failed to query version: %v", err)
	}
	if err := h.Close(); err != nil {
		return fmt.Errorf("Failed to close reader: %v", err)
	}

	if actual != expected {
		return fmt.Errorf("Expected version %d, but got %d", expected, actual)
	}

	return nil
}

func WriteVersionedBlob(bs blobstore.BlobStore, blobpath string, version byte) error {
	w, err := bs.OpenWriter(blobpath)
	if err != nil {
		return fmt.Errorf("Failed to open writer: %v", err)
	}

	if _, err := w.Write([]byte{version}); err != nil {
		return fmt.Errorf("Failed to blob write: %v", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("Failed to close writer: %v", err)
	}

	return nil
}

func WriteVersionedBlobRA(bs blobstore.RandomAccessBlobStore, blobpath string, version byte) error {
	bh, err := bs.Open(blobpath, flags.O_RDWRCREATE)
	if err != nil {
		return fmt.Errorf("Failed to open handle: %v", err)
	}
	if err := bh.PWrite([]byte{version}, 0); err != nil {
		return fmt.Errorf("Failed to blob write: %v", err)
	}
	if err := bh.Close(); err != nil {
		return fmt.Errorf("Failed to close handle: %v", err)
	}

	return nil
}

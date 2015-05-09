package otaru_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/nyaxt/otaru"
)

func TestFileBlobStore() *otaru.FileBlobStore {
	return TestFileBlobStoreOfName("")
}

func TestFileBlobStoreOfName(name string) *otaru.FileBlobStore {
	tempdir, err := ioutil.TempDir("", fmt.Sprintf("otarutest%s", name))
	if err != nil {
		log.Fatalf("failed to create tmpdir: %v", err)
	}
	fbs, err := otaru.NewFileBlobStore(tempdir, otaru.O_RDWRCREATE)
	if err != nil {
		log.Fatalf("failed to create blobstore: %v", err)
	}
	return fbs
}

func TestQueryVersion(r io.Reader) (int, error) {
	b := make([]byte, 1)
	if _, err := r.Read(b); err != nil {
		if err == io.EOF {
			// no data -> ver 0.
			return 0, nil
		}
		return -1, fmt.Errorf("Failed to read 1 byte: %v", err)
	}

	return int(b[0]), nil
}

func AssertBlobVersion(bs otaru.BlobStore, blobpath string, expected int) error {
	r, err := bs.OpenReader(blobpath)
	if err != nil {
		if expected == 0 && os.IsNotExist(err) {
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

func AssertBlobVersionRA(bs otaru.RandomAccessBlobStore, blobpath string, expected int) error {
	h, err := bs.Open(blobpath, otaru.O_RDONLY)
	if err != nil {
		return fmt.Errorf("Failed to open reader: %v", err)
	}
	actual, err := TestQueryVersion(&otaru.OffsetReader{h, 0})
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

func WriteVersionedBlob(bs otaru.BlobStore, blobpath string, version byte) error {
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

func WriteVersionedBlobRA(bs otaru.RandomAccessBlobStore, blobpath string, version byte) error {
	bh, err := bs.Open(blobpath, otaru.O_RDWRCREATE)
	if err != nil {
		return fmt.Errorf("Failed to open handle: %v", err)
	}
	if err := bh.PWrite(0, []byte{version}); err != nil {
		return fmt.Errorf("Failed to blob write: %v", err)
	}
	if err := bh.Close(); err != nil {
		return fmt.Errorf("Failed to close handle: %v", err)
	}

	return nil
}

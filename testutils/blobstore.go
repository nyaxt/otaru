package otaru_test

import (
	. "github.com/nyaxt/otaru"

	"io/ioutil"
	"log"
)

func TestFileBlobStore() *FileBlobStore {
	tempdir, err := ioutil.TempDir("", "otarutest")
	if err != nil {
		log.Fatalf("failed to create tmpdir: %v", err)
	}
	fbs, err := NewFileBlobStore(tempdir, O_RDWR)
	if err != nil {
		log.Fatalf("failed to create blobstore: %v", err)
	}
	return fbs
}

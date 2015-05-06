package gcs_test

import (
	"os"
	"path"
	"testing"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/gcs"
)

func TestGCSBlobStore_Interface(t *testing.T) {
	homedir := os.Getenv("HOME")

	var bs otaru.RandomAccessBlobStore
	bs = gcs.NewGCSBlobStore(
		path.Join(homedir, ".otaru", "credentials.json"),
		path.Join(homedir, ".otaru", "tokencache.json"),
	)
	bs.Open("hoge", otaru.O_RDONLY)
}

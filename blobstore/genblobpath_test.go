package blobstore_test

import (
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/flags"
	. "github.com/nyaxt/otaru/testutils"

	"testing"
)

func TestGenerateNewBlobPath_Unique(t *testing.T) {
	n := 200
	bs := blobstore.NewMockBlobStore()

	for i := 0; i < n; i++ {
		bpath, err := blobstore.GenerateNewBlobPath(bs)
		if err != nil {
			t.Errorf("Failed to GenerateNewBlobPath on %d iter: %v", i, err)
		}

		bh, err := bs.Open(bpath, flags.O_RDONLY)
		if err != nil {
			t.Errorf("open bpath \"%s\" failed: %v", bpath, err)
		}
		if err := bh.PWrite(HelloWorld, 0); err != nil {
			t.Errorf("write helloworld to bpath \"%s\" failed: %v", bpath, err)
		}
		if err := bh.Close(); err != nil {
			t.Errorf("close bpath \"%s\" failed: %v", bpath, err)
		}
	}

	if len(bs.Paths) != n {
		t.Errorf("Expected %d unique entries, but found %d entries", n, len(bs.Paths))
	}
}

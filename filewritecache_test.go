package otaru_test

import (
	"testing"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
)

func TestRegression_PWriteAfterSync(t *testing.T) {
	bh := blobstore.NewMockBlobHandle()

	wc := otaru.NewFileWriteCache()
	if err := wc.PWrite([]byte{1, 2, 3}, 0); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}

	if err := wc.Sync(bh); err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	if err := wc.PWrite([]byte{4, 5, 6}, 3); err != nil {
		t.Errorf("PWrite failed: %v", err)
		return
	}
	// Test PASS if no panic
}

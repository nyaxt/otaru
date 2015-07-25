package cachedblobstore_test

import (
	"reflect"
	"testing"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	fl "github.com/nyaxt/otaru/flags"
)

func TestCacheUsageStats_FindLeastUsed(t *testing.T) {
	s := cachedblobstore.NewCacheUsageStats()
	s.ImportBlobList([]string{"nevertouched", "touched", "removed"})
	s.ObserveOpen("touched", fl.O_RDONLY)
	s.ObserveRemoveBlob("removed")
	s.ObserveOpen("new", fl.O_WRONLY)

	leastUsed := s.FindLeastUsed()
	if !reflect.DeepEqual([]string{"nevertouched", "touched", "new"}, leastUsed) {
		t.Errorf("Unexpected result: %v", leastUsed)
	}
}

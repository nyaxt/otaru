package gc_test

import (
	"reflect"
	"testing"

	"github.com/nyaxt/otaru/gc"

	"golang.org/x/net/context"
)

type MockGCBlobStore struct {
	bs        []string
	removedbs []string
}

var _ = gc.GCableBlobStore(&MockGCBlobStore{})

func (bs *MockGCBlobStore) ListBlobs() ([]string, error) { return bs.bs, nil }
func (bs *MockGCBlobStore) RemoveBlob(b string) error {
	bs.removedbs = append(bs.removedbs, b)
	return nil
}

type MockFscker struct {
	usedbs []string
}

func (idb *MockFscker) Fsck() ([]string, []error) { return idb.usedbs, nil }

func TestGC_Basic(t *testing.T) {
	bs := &MockGCBlobStore{
		bs:        []string{"a", "b", "x", "y", "z", "INODEDB_SNAPSHOT"},
		removedbs: []string{},
	}
	idb := &MockFscker{
		usedbs: []string{"x", "y", "z"},
	}

	if err := gc.GC(context.TODO(), bs, idb, false); err != nil {
		t.Errorf("GC err: %v", err)
	}

	if !reflect.DeepEqual([]string{"a", "b"}, bs.removedbs) {
		t.Errorf("GC removed unexpected blobs: %v", bs.removedbs)
	}
}

func TestGC_EmptyRun(t *testing.T) {
	bs := &MockGCBlobStore{
		bs:        []string{"x", "y", "z"},
		removedbs: []string{},
	}
	idb := &MockFscker{
		usedbs: []string{"x", "y", "z"},
	}

	// vvv should not panic.
	if err := gc.GC(context.TODO(), bs, idb, false); err != nil {
		t.Errorf("GC err: %v", err)
	}
	if len(bs.removedbs) > 0 {
		t.Errorf("GC removed unexpected blobs: %v", bs.removedbs)
	}
}

package datastore_test

/*
import (
	"reflect"
	"sort"
	"testing"

	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/datastore"
)

func TestINodeDBSSLocator_PutLocate(t *testing.T) {
	loc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(authtu.TestBucketName()))

	if _, err := loc.DeleteAll(); err != nil {
		t.Errorf("DeleteAll failed unexpectedly: %v", err)
		return
	}

	bp, err := loc.Locate(0)
	if err != datastore.EEMPTY {
		t.Errorf("Locate() when no entry should fail, but succeeded.")
	}

	if err := loc.Put("META-snapshot123", 123); err != nil {
		t.Errorf("Put failed unexpectedly: %v", err)
		return
	}
	if err := loc.Put("META-snapshot231", 231); err != nil {
		t.Errorf("Put failed unexpectedly: %v", err)
		return
	}

	bp, err = loc.Locate(0)
	if err != nil {
		t.Errorf("Locate failed unexpectedly: %v", err)
		return
	}
	if bp != "META-snapshot231" {
		t.Errorf("Locate returned unexpected bp: %v", bp)
		return
	}

	bp, err = loc.Locate(1)
	if err != nil {
		t.Errorf("Locate failed unexpectedly: %v", err)
		return
	}
	if bp != "META-snapshot123" {
		t.Errorf("Locate returned unexpected bp: %v", bp)
		return
	}

	if err := loc.Put("META-snapshot345", 345); err != nil {
		t.Errorf("Put failed unexpectedly: %v", err)
		return
	}

	bp, err = loc.Locate(0)
	if err != nil {
		t.Errorf("Locate failed unexpectedly: %v", err)
		return
	}
	if bp != "META-snapshot345" {
		t.Errorf("Locate returned unexpected bp: %v", bp)
		return
	}

	bps, err := loc.DeleteAll()
	if err != nil {
		t.Errorf("DeleteAll failed unexpectedly: %v", err)
		return
	}
	sort.Strings(bps)
	if !reflect.DeepEqual([]string{
		"META-snapshot123",
		"META-snapshot231",
		"META-snapshot345",
	}, bps) {
		t.Errorf("DeleteAll returned unexpected blobpaths: %v", bps)
		return
	}
}
*/

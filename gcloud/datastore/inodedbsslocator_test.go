package datastore_test

import (
	"reflect"
	"sort"
	"testing"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/flags"
	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/util"
)

func TestINodeDBSSLocator_PutLocate(t *testing.T) {
	muTest.Lock()
	defer muTest.Unlock()

	loc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(authtu.TestBucketName()), flags.O_RDWRCREATE)

	if _, err := loc.DeleteAll(context.Background(), false); err != nil {
		t.Errorf("DeleteAll failed unexpectedly: %v", err)
		return
	}

	bp, _, err := loc.Locate(0)
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

	bp, txid, err := loc.Locate(0)
	if err != nil {
		t.Errorf("Locate failed unexpectedly: %v", err)
		return
	}
	if txid != 231 {
		t.Errorf("Locate returned unexpected txid: %v", txid)
		return
	}
	if bp != "META-snapshot231" {
		t.Errorf("Locate returned unexpected bp: %v", bp)
		return
	}

	bp, txid, err = loc.Locate(1)
	if err != nil {
		t.Errorf("Locate failed unexpectedly: %v", err)
		return
	}
	if txid != 123 {
		t.Errorf("Locate returned unexpected txid: %v", txid)
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

	bp, txid, err = loc.Locate(0)
	if err != nil {
		t.Errorf("Locate failed unexpectedly: %v", err)
		return
	}
	if txid != 345 {
		t.Errorf("Locate returned unexpected txid: %v", txid)
		return
	}
	if bp != "META-snapshot345" {
		t.Errorf("Locate returned unexpected bp: %v", bp)
		return
	}

	bps, err := loc.DeleteAll(context.Background(), false)
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

func TestINodeDBSSLocator_ReadOnly(t *testing.T) {
	muTest.Lock()
	defer muTest.Unlock()

	// prep
	wloc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(authtu.TestBucketName()), flags.O_RDWRCREATE)
	deleteAll := func() {
		if _, err := wloc.DeleteAll(context.Background(), false); err != nil {
			t.Errorf("DeleteAll failed unexpectedly: %v", err)
			return
		}
	}
	deleteAll()
	defer deleteAll()

	if err := wloc.Put("META-snapshot123", 123); err != nil {
		t.Errorf("Put failed unexpectedly: %v", err)
		return
	}

	rloc := datastore.NewINodeDBSSLocator(authtu.TestDSConfig(authtu.TestBucketName()), flags.O_RDONLY)

	// Locate should succeed
	bp, txid, err := rloc.Locate(0)
	if err != nil {
		t.Errorf("Locate failed unexpectedly: %v", err)
		return
	}
	if txid != 123 {
		t.Errorf("Locate returned unexpected txid: %v", txid)
		return
	}
	if bp != "META-snapshot123" {
		t.Errorf("Locate returned unexpected bp: %v", bp)
		return
	}

	// Put should fail
	err = rloc.Put("META-snapshot231", 231)
	if err == nil {
		t.Errorf("Unexpected Put success")
		return
	}
	if err != util.EACCES {
		t.Errorf("Expected EACCES. got %v", err)
		return
	}

	// DeleteAll should fail
	_, err = rloc.DeleteAll(context.Background(), false)
	if err == nil {
		t.Errorf("Unexpected DeleteAll success")
		return
	}
	if err != util.EACCES {
		t.Errorf("Expected EACCES. got %v", err)
		return
	}
}

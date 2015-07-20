package datastore_test

import (
	"testing"

	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/datastore"
)

func TestGlobalLocker_LockUnlock(t *testing.T) {
	l := datastore.NewGlobalLocker(testConfig(authtu.TestBucketName()), "otaru-unittest", "unittest desuyo-")

	if err := l.ForceUnlock(); err != nil {
		t.Errorf("ForceUnlock() failed: %v", err)
		return
	}

	if err := l.Lock(); err != nil {
		t.Errorf("Lock() failed: %v", err)
	}

	if err := l.Unlock(); err != nil {
		t.Errorf("Unlock() failed: %v", err)
	}
}

func TestGlobalLocker_ActAsMutex(t *testing.T) {
	l1 := datastore.NewGlobalLocker(testConfig(authtu.TestBucketName()), "otaru-unittest-1", "hogefuga")
	l2 := datastore.NewGlobalLocker(testConfig(authtu.TestBucketName()), "otaru-unittest-2", "foobar")

	if err := l1.ForceUnlock(); err != nil {
		t.Errorf("ForceUnlock() failed: %v", err)
		return
	}

	// l1 takes lock. l2 lock should fail.
	if err := l1.Lock(); err != nil {
		t.Errorf("l1.Lock() failed: %v", err)
	}
	err := l2.Lock()
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("l2.Lock() unexpected (no) err: %v", err)
	}

	if err := l1.Unlock(); err != nil {
		t.Errorf("l1.Unlock() failed: %v", err)
	}
}

func TestGlobalLocker_ForceUnlock(t *testing.T) {
	l1 := datastore.NewGlobalLocker(testConfig(authtu.TestBucketName()), "otaru-unittest-1", "hogefuga")
	l2 := datastore.NewGlobalLocker(testConfig(authtu.TestBucketName()), "otaru-unittest-2", "foobar")

	if err := l1.ForceUnlock(); err != nil {
		t.Errorf("ForceUnlock() failed: %v", err)
		return
	}

	// l1 takes lock. l2 lock should fail.
	if err := l1.Lock(); err != nil {
		t.Errorf("l1.Lock() failed: %v", err)
	}
	err := l2.Lock()
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("l2.Lock() unexpected (no) err: %v", err)
	}

	// l2 force unlocks and takes lock
	if err := l2.ForceUnlock(); err != nil {
		t.Errorf("l2.ForceUnlock() failed: %v", err)
	}
	if err := l2.Lock(); err != nil {
		t.Errorf("l2.Lock() failed: %v", err)
	}

	// l1 unlock should fail
	err = l1.Unlock()
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("l1.Unlock() failed with unexpected (no) err: %v", err)
	}

	// l2 unlock should succeed
	if err := l2.Unlock(); err != nil {
		t.Errorf("l2.Unlock() failed: %v", err)
	}
}

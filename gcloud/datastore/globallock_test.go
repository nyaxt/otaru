package datastore_test

import (
	"context"
	"testing"

	authtu "github.com/nyaxt/otaru/gcloud/auth/testutils"
	"github.com/nyaxt/otaru/gcloud/datastore"
)

const (
	readOnly    = true
	notReadOnly = false
)

func TestGlobalLocker_LockUnlock(t *testing.T) {
	muTest.Lock()
	defer muTest.Unlock()

	ctx := context.Background()

	l := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest", "unittest desuyo-")

	if err := l.ForceUnlock(ctx); err != nil {
		t.Errorf("ForceUnlock() failed: %v", err)
		return
	}

	if err := l.Lock(ctx, notReadOnly); err != nil {
		t.Errorf("Lock() failed: %v", err)
	}

	if err := l.Unlock(ctx); err != nil {
		t.Errorf("Unlock() failed: %v", err)
	}
}

func TestGlobalLocker_ActAsMutex(t *testing.T) {
	muTest.Lock()
	defer muTest.Unlock()

	ctx := context.Background()

	l1 := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest-1", "hogefuga")
	l2 := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest-2", "foobar")
	l3 := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest-3", "readonly")

	if err := l1.ForceUnlock(ctx); err != nil {
		t.Errorf("ForceUnlock() failed: %v", err)
		return
	}

	// l1 takes lock. l2/l3 lock should fail.
	if err := l1.Lock(ctx, notReadOnly); err != nil {
		t.Errorf("l1.Lock() failed: %v", err)
	}
	err := l2.Lock(ctx, notReadOnly)
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("l2.Lock() unexpected (no) err: %v", err)
	}
	err = l3.Lock(ctx, notReadOnly)
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("l2.Lock() unexpected (no) err: %v", err)
	}

	if err := l1.Unlock(ctx); err != nil {
		t.Errorf("l1.Unlock() failed: %v", err)
	}
}

func TestGlobalLocker_ForceUnlock(t *testing.T) {
	muTest.Lock()
	defer muTest.Unlock()

	ctx := context.Background()

	l1 := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest-1", "hogefuga")
	l2 := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest-2", "foobar")

	if err := l1.ForceUnlock(ctx); err != nil {
		t.Errorf("ForceUnlock() failed: %v", err)
		return
	}

	// l1 takes lock. l2 lock should fail.
	if err := l1.Lock(ctx, notReadOnly); err != nil {
		t.Errorf("l1.Lock() failed: %v", err)
	}
	err := l2.Lock(ctx, notReadOnly)
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("l2.Lock() unexpected (no) err: %v", err)
	}

	// l2 force unlocks and takes lock
	if err := l2.ForceUnlock(ctx); err != nil {
		t.Errorf("l2.ForceUnlock() failed: %v", err)
	}
	if err := l2.Lock(ctx, notReadOnly); err != nil {
		t.Errorf("l2.Lock() failed: %v", err)
	}

	// l1 unlock should fail
	err = l1.Unlock(ctx)
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("l1.Unlock() failed with unexpected (no) err: %v", err)
	}

	// l2 unlock should succeed
	if err := l2.Unlock(ctx); err != nil {
		t.Errorf("l2.Unlock() failed: %v", err)
	}
}

func TestGlobalLocker_ReadOnlyLock(t *testing.T) {
	muTest.Lock()
	defer muTest.Unlock()

	ctx := context.Background()

	lw := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest-1", "hogefuga")
	lr := datastore.NewGlobalLocker(authtu.TestDSConfig(authtu.TestBucketName()), "otaru-unittest-ro", "readonly")

	forceUnlock := func() {
		if err := lw.ForceUnlock(ctx); err != nil {
			t.Errorf("ForceUnlock() failed: %v", err)
			return
		}
	}
	forceUnlock()
	defer forceUnlock()

	// lr should succeed if there is no lw
	if err := lr.Lock(ctx, readOnly); err != nil {
		t.Errorf("lr.Lock() failed: %v", err)
	}

	// lw should succeed even with lr. o_O
	if err := lw.Lock(ctx, notReadOnly); err != nil {
		t.Errorf("lw.Lock() failed: %v", err)
	}

	// lr should fail w/ lw
	err := lr.Lock(ctx, notReadOnly)
	if _, ok := err.(*datastore.ErrLockTaken); !ok {
		t.Errorf("lr.Lock() unexpected (no) err: %v", err)
	}
}

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
	}

	if err := l.Lock(); err != nil {
		t.Errorf("Lock() failed: %v", err)
	}

	if err := l.Unlock(); err != nil {
		t.Errorf("Unlock() failed: %v", err)
	}
}

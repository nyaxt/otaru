package cachedblobstore_test

import (
	"testing"

	"github.com/nyaxt/otaru/blobstore/cachedblobstore"
	tu "github.com/nyaxt/otaru/testutils"
)

func TestCachedBackendVersion_UseCached(t *testing.T) {
	bs := tu.TestFileBlobStore()
	cbv := cachedblobstore.NewCachedBackendVersion(bs, tu.TestQueryVersion)

	cbv.Set("foobar", 123)
	v, err := cbv.Query("foobar")
	if err != nil {
		t.Errorf("Unexpected Query() err: %v", err)
		return
	}
	if v != 123 {
		t.Errorf("Unexpected Query() result. v: %d", v)
		return
	}
}

func TestCachedBackendVersion_FillCache(t *testing.T) {
	bs := tu.TestFileBlobStore()
	if err := tu.WriteVersionedBlob(bs, "uncached", 42); err != nil {
		t.Errorf("%v", err)
		return
	}

	cbv := cachedblobstore.NewCachedBackendVersion(bs, tu.TestQueryVersion)
	v, err := cbv.Query("uncached")
	if err != nil {
		t.Errorf("Unexpected Query() err: %v", err)
		return
	}
	if v != 42 {
		t.Errorf("Unexpected Query() result. v: %d", v)
		return
	}
}

func TestCachedBackedVersion_SaveRestore(t *testing.T) {
	bs := tu.TestFileBlobStore()

	cbv := cachedblobstore.NewCachedBackendVersion(bs, tu.TestQueryVersion)
	cbv.Set("foobar", 123)
	if err := cbv.SaveStateToBlobstore(tu.TestCipher(), bs); err != nil {
		t.Errorf("Failed to save state: %v", err)
		return
	}

	cbv2 := cachedblobstore.NewCachedBackendVersion(bs, tu.TestQueryVersion)
	if err := cbv2.RestoreStateFromBlobstore(tu.TestCipher(), bs); err != nil {
		t.Errorf("Failed to restore  state: %v", err)
	}
	v, err := cbv2.Query("foobar")
	if err != nil {
		t.Errorf("Unexpected Query() err: %v", err)
		return
	}
	if v != 123 {
		t.Errorf("Unexpected Query() result. v: %d", v)
		return
	}
}

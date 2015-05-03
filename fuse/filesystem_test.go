package fuse_test

import (
	"log"
	"testing"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/fuse"
	. "github.com/nyaxt/otaru/testutils"
)

func TestServeFUSE(t *testing.T) {
	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}

	mountpoint := "/tmp/hoge"

	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	done := make(chan bool)
	ready := make(chan bool)
	go func() {
		if err := fuse.ServeFUSE(mountpoint, fs, ready); err != nil {
			t.Errorf("ServeFUSE err: %v", err)
			close(ready)
		}
		close(done)
	}()

	<-ready
	if err := bfuse.Unmount(mountpoint); err != nil {
		t.Errorf("umount failed: %v", err)
	}

	<-done
}

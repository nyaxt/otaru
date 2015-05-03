package fuse_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"path"
	"testing"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/fuse"
	. "github.com/nyaxt/otaru/testutils"
)

func fusetestCommon(t *testing.T, fs *otaru.FileSystem, f func(mountpoint string)) {
	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}

	mountpoint := "/tmp/hoge"

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

	f(mountpoint)

	if err := bfuse.Unmount(mountpoint); err != nil {
		t.Errorf("umount failed: %v", err)
	}
	<-done
}

func TestServeFUSE_DoNothing(t *testing.T) {
	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	fusetestCommon(t, fs, func(mountpoint string) {})
}

func TestServeFUSE_WriteReadFile(t *testing.T) {
	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	fusetestCommon(t, fs, func(mountpoint string) {
		if err := ioutil.WriteFile(path.Join(mountpoint, "hello.txt"), HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}
	})

	fusetestCommon(t, fs, func(mountpoint string) {
		b, err := ioutil.ReadFile(path.Join(mountpoint, "hello.txt"))
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !bytes.Equal(HelloWorld, b) {
			t.Errorf("Content mismatch!: %v", err)
		}
	})
}

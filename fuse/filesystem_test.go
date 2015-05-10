package fuse_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"syscall"
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

	if err := os.Mkdir(mountpoint, 0777); err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create mountpoint: %v", err)
	}

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

		b, err := ioutil.ReadFile(path.Join(mountpoint, "hello.txt"))
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !bytes.Equal(HelloWorld, b) {
			t.Errorf("Content mismatch!: %v", err)
		}
	})

	// Check that it persists
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

func TestServeFUSE_RenameFile(t *testing.T) {
	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	fusetestCommon(t, fs, func(mountpoint string) {
		before := path.Join(mountpoint, "aaa.txt")
		after := path.Join(mountpoint, "bbb.txt")

		if err := ioutil.WriteFile(before, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}

		if err := os.Rename(before, after); err != nil {
			t.Errorf("failed to rename file: %v", err)
		}

		b, err := ioutil.ReadFile(after)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !bytes.Equal(HelloWorld, b) {
			t.Errorf("Content mismatch!: %v", err)
		}
	})
}

func TestServeFUSE_RemoveFile(t *testing.T) {
	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	fusetestCommon(t, fs, func(mountpoint string) {
		filepath := path.Join(mountpoint, "hello.txt")

		if err := ioutil.WriteFile(filepath, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}

		if err := os.Remove(filepath); err != nil {
			t.Errorf("Failed to remove file: %v", err)
		}

		_, err := ioutil.ReadFile(filepath)
		if pe, ok := err.(*os.PathError); !ok || pe.Err != syscall.ENOENT {
			t.Errorf("File still exists: %v", err)
		}
	})

	fusetestCommon(t, fs, func(mountpoint string) {
		filepath := path.Join(mountpoint, "hello.txt")
		_, err := ioutil.ReadFile(filepath)
		if pe, ok := err.(*os.PathError); !ok || pe.Err != syscall.ENOENT {
			t.Errorf("File still exists: %v", err)
		}
	})
}

func TestServeFUSE_Mkdir(t *testing.T) {
	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	fusetestCommon(t, fs, func(mountpoint string) {
		dirpath := path.Join(mountpoint, "hokkaido")
		if err := os.Mkdir(dirpath, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
		}

		filepath := path.Join(dirpath, "otaru.txt")
		if err := ioutil.WriteFile(filepath, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}
	})

	fusetestCommon(t, fs, func(mountpoint string) {
		filepath := path.Join(mountpoint, "hokkaido", "otaru.txt")
		b, err := ioutil.ReadFile(filepath)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !bytes.Equal(HelloWorld, b) {
			t.Errorf("Content mismatch!: %v", err)
		}
	})
}

func TestServeFUSE_MoveFile(t *testing.T) {
	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	fusetestCommon(t, fs, func(mountpoint string) {
		dir1 := path.Join(mountpoint, "dir1")
		if err := os.Mkdir(dir1, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
		}

		dir2 := path.Join(mountpoint, "dir2")
		if err := os.Mkdir(dir2, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
		}

		before := path.Join(dir1, "aaa.txt")
		after := path.Join(dir2, "bbb.txt")

		if err := ioutil.WriteFile(before, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}

		if err := os.Rename(before, after); err != nil {
			t.Errorf("failed to rename file: %v", err)
		}

		b, err := ioutil.ReadFile(after)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !bytes.Equal(HelloWorld, b) {
			t.Errorf("Content mismatch!: %v", err)
		}
	})
}

func TestServeFUSE_Rmdir(t *testing.T) {
	bs := TestFileBlobStore()
	fs := otaru.NewFileSystemEmpty(bs, TestCipher())

	fusetestCommon(t, fs, func(mountpoint string) {
		dirpath := path.Join(mountpoint, "hokkaido")
		if err := os.Mkdir(dirpath, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
		}

		filepath := path.Join(dirpath, "otaru.txt")
		if err := ioutil.WriteFile(filepath, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}

		err := os.Remove(dirpath)
		if err == nil {
			t.Errorf("Removed non-empty dir without err")
		} else {
			if en, ok := err.(*os.PathError).Err.(syscall.Errno); !ok || en != syscall.ENOTEMPTY {
				t.Errorf("Expected ENOTEMPTY err when trying to remove non-empty dir: %v, %d", err, en)
			}
		}

		if err := os.Remove(filepath); err != nil {
			t.Errorf("Failed to remove file: %v", err)
		}

		if err := os.Remove(dirpath); err != nil {
			t.Errorf("Failed to remove empty dir: %v", err)
		}
	})
}

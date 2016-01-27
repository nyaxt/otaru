package fuse_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"
	"testing"
	"time"

	bfuse "github.com/nyaxt/fuse"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/fuse"
	"github.com/nyaxt/otaru/inodedb"
	. "github.com/nyaxt/otaru/testutils"
)

func init() { EnsureLogger() }

func maybeSkipTest(t *testing.T) {
	if os.Getenv("SKIP_FUSE_TEST") == "1" {
		t.Skip("Skipping FUSE test")
	}
}

func fusetestFileSystem() *otaru.FileSystem {
	sio := inodedb.NewSimpleDBStateSnapshotIO()
	txio := inodedb.NewSimpleDBTransactionLogIO()

	idb, err := inodedb.NewEmptyDB(sio, txio)
	if err != nil {
		log.Fatalf("NewEmptyDB failed: %v", err)
	}

	bs := TestFileBlobStore()
	fs := otaru.NewFileSystem(idb, bs, TestCipher())

	return fs
}

func fusetestCommon(t *testing.T, fs *otaru.FileSystem, f func(mountpoint string)) {
	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}

	mountpoint := "/tmp/hoge"

	if err := os.Mkdir(mountpoint, 0777); err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create mountpoint: %v", err)
	}

	bfuse.Unmount(mountpoint)

	done := make(chan bool)
	ready := make(chan bool)
	go func() {
		if err := fuse.ServeFUSE("otaru-test", mountpoint, fs, ready); err != nil {
			t.Errorf("ServeFUSE err: %v", err)
			close(ready)
		}
		close(done)
	}()
	<-ready

	f(mountpoint)

	time.Sleep(100 * time.Millisecond)
	if err := bfuse.Unmount(mountpoint); err != nil {
		t.Errorf("umount failed: %v", err)
	}
	<-done
}

func TestServeFUSE_DoNothing(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()
	fusetestCommon(t, fs, func(mountpoint string) {})
}

func TestServeFUSE_WriteReadFile(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

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
			t.Errorf("Content mismatch!: %v", b)
		}
	})
}

func TestServeFUSE_WriteAppend(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

	var exp bytes.Buffer
	fusetestCommon(t, fs, func(mountpoint string) {
		if err := ioutil.WriteFile(path.Join(mountpoint, "foobar.log"), HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}
		exp.Write(HelloWorld)
	})
	fusetestCommon(t, fs, func(mountpoint string) {
		// According to POSIX:
		// O_APPEND
		//     If set, the file offset will be set to the end of the file prior to each write.
		// ref: http://pubs.opengroup.org/onlinepubs/7908799/xsh/open.html
		f, err := os.OpenFile(path.Join(mountpoint, "foobar.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			t.Errorf("failed to open file: %v", err)
		}
		defer f.Close()

		for _, seek := range []int64{0, 10, 5, 3} {
			if _, err := f.Seek(seek, 0); err != nil {
				t.Errorf("Seek failed: %v", err)
			}
			l := fmt.Sprintf("Write after seek %d", seek)
			if _, err := f.WriteString(l); err != nil {
				t.Errorf("Failed write: %v", err)
			}
			exp.WriteString(l)
		}
	})
	fusetestCommon(t, fs, func(mountpoint string) {
		b, err := ioutil.ReadFile(path.Join(mountpoint, "foobar.log"))
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !bytes.Equal(exp.Bytes(), b) {
			t.Errorf("Content mismatch!: %v", string(b))
		}
	})
}

func TestServeFUSE_RenameFile(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

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

func TestServeFUSE_RenameFile_Overwrite(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

	fusetestCommon(t, fs, func(mountpoint string) {
		src := path.Join(mountpoint, "aaa.txt")
		tgt := path.Join(mountpoint, "bbb.txt")

		if err := ioutil.WriteFile(src, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}
		if err := ioutil.WriteFile(tgt, HogeFugaPiyo, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
		}

		if err := os.Rename(src, tgt); err != nil {
			t.Errorf("failed to rename file: %v", err)
		}

		b, err := ioutil.ReadFile(tgt)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !bytes.Equal(HelloWorld, b) {
			t.Errorf("Content mismatch!: %v", err)
		}
	})
}

func TestServeFUSE_RemoveFile(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

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
	maybeSkipTest(t)
	fs := fusetestFileSystem()

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
	maybeSkipTest(t)
	fs := fusetestFileSystem()

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
	maybeSkipTest(t)
	fs := fusetestFileSystem()

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

func TestServeFUSE_LsCmd(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

	fusetestCommon(t, fs, func(mountpoint string) {
		dirpath := path.Join(mountpoint, "hokkaido")
		if err := os.Mkdir(dirpath, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
			return
		}

		filepath := path.Join(dirpath, "otaru.txt")
		if err := ioutil.WriteFile(filepath, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
			return
		}

		// We need to use "ls -a" cmd here, as golang Readdir automatically omits "." and ".." entry, which we want to check
		cmd := exec.Command("ls", "-a", dirpath)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("ls failed: %v", err)
			return
		}

		foundOtaruTxt := false
		foundCurrentDir := false
		foundParentDir := false
		sc := bufio.NewScanner(&out)
		for sc.Scan() {
			switch l := sc.Text(); l {
			case "otaru.txt":
				foundOtaruTxt = true
			case ".":
				foundCurrentDir = true
			case "..":
				foundParentDir = true
			default:
				t.Errorf("Found unexpected entry: %s", l)
			}
		}
		if !foundOtaruTxt {
			t.Errorf("otaru.txt not found in the dir")
		}
		if !foundCurrentDir {
			t.Errorf("\".\"not found in the dir")
		}
		if !foundParentDir {
			t.Errorf("\"..\"not found in the dir")
		}
	})
}

func TestServeFUSE_Chmod(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

	fusetestCommon(t, fs, func(mountpoint string) {
		dirpath := path.Join(mountpoint, "hokkaido")
		if err := os.Mkdir(dirpath, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
			return
		}

		filepath := path.Join(dirpath, "otaru.txt")
		if err := ioutil.WriteFile(filepath, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
			return
		}

		fi, err := os.Stat(dirpath)
		if err != nil {
			t.Errorf("Failed to stat dir: %v", err)
			return
		}
		if !fi.IsDir() {
			t.Errorf("dir isn't dir!")
		}
		if fi.Mode()&os.ModePerm != 0755 {
			t.Errorf("invalid initial perm!")
		}

		fi, err = os.Stat(filepath)
		if err != nil {
			t.Errorf("Failed to stat file: %v", err)
			return
		}
		if fi.IsDir() {
			t.Errorf("file is dir!")
		}
		if fi.Mode()&os.ModePerm != 0644 {
			t.Errorf("invalid initial perm!")
		}

		if err := os.Chmod(dirpath, 0700); err != nil {
			t.Errorf("Failed to chmod dir: %v", err)
			return
		}
		if err := os.Chmod(filepath, 0764); err != nil {
			t.Errorf("Failed to chmod file: %v", err)
			return
		}

		fi, err = os.Stat(dirpath)
		if err != nil {
			t.Errorf("Failed to stat dir: %v", err)
			return
		}
		if !fi.IsDir() {
			t.Errorf("dir isn't dir!")
		}
		if fi.Mode()&os.ModePerm != 0700 {
			t.Errorf("invalid perm! %o", fi.Mode())
		}

		fi, err = os.Stat(filepath)
		if err != nil {
			t.Errorf("Failed to stat file: %v", err)
			return
		}
		if fi.IsDir() {
			t.Errorf("file is dir!")
		}
		if fi.Mode()&os.ModePerm != 0764 {
			t.Errorf("invalid perm! %o", fi.Mode())
		}
	})
}

func TestServeFUSE_Chtimes(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

	stableT := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	fusetestCommon(t, fs, func(mountpoint string) {
		dirpath := path.Join(mountpoint, "hokkaido")
		if err := os.Mkdir(dirpath, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
			return
		}

		filepath := path.Join(dirpath, "otaru.txt")
		if err := ioutil.WriteFile(filepath, HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
			return
		}

		if err := os.Chtimes(dirpath, stableT, stableT); err != nil {
			t.Errorf("Failed to chtimes dir: %v", err)
			return
		}
		if err := os.Chtimes(filepath, stableT, stableT); err != nil {
			t.Errorf("Failed to chtimes file: %v", err)
			return
		}

		fi, err := os.Stat(dirpath)
		if err != nil {
			t.Errorf("Failed to stat dir: %v", err)
			return
		}
		if !fi.IsDir() {
			t.Errorf("dir isn't dir!")
		}
		if fi.ModTime().Sub(stableT) > time.Second {
			t.Errorf("invalid modifiedT! %v", fi.ModTime())
		}

		fi, err = os.Stat(filepath)
		if err != nil {
			t.Errorf("Failed to stat file: %v", err)
			return
		}
		if fi.IsDir() {
			t.Errorf("file is dir!")
		}
		if fi.ModTime().Sub(stableT) > time.Second {
			t.Errorf("invalid modifiedT! %v", fi.ModTime())
		}
	})
}

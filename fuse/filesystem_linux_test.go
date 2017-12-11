package fuse_test

import (
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"

	tu "github.com/nyaxt/otaru/testutils"
)

// Note: invoke test by: docker build -t otaru-dev . && docker run -ti --rm -privileged=true otaru-dev go test ./fuse -run TestServe_ChUGid -v
func TestServe_ChUGid(t *testing.T) {
	maybeSkipTest(t)
	fs := fusetestFileSystem()

	if os.Getuid() != 0 {
		t.Skip("ChUGid test is only possible when root")
		return
	}

	fusetestCommon(t, fs, func(mountpoint string) {
		dirpath := path.Join(mountpoint, "hokkaido")
		if err := os.Mkdir(dirpath, 0755); err != nil {
			t.Errorf("Failed to mkdir: %v", err)
			return
		}

		filepath := path.Join(dirpath, "otaru.txt")
		if err := ioutil.WriteFile(filepath, tu.HelloWorld, 0644); err != nil {
			t.Errorf("failed to write file: %v", err)
			return
		}

		if err := os.Chown(filepath, 123, 456); err != nil {
			t.Errorf("Failed to chown file: %v", err)
			return
		}
		if err := os.Chown(dirpath, 234, 567); err != nil {
			t.Errorf("Failed to chown dir: %v", err)
			return
		}

		fi, err := os.Stat(filepath)
		if err != nil {
			t.Errorf("Failed to stat dir: %v", err)
			return
		}
		if fi.IsDir() {
			t.Errorf("file is dir!")
		}
		if uid := fi.Sys().(*syscall.Stat_t).Uid; uid != 123 {
			t.Errorf("Invalid UID: %d", uid)
		}
		if gid := fi.Sys().(*syscall.Stat_t).Gid; gid != 456 {
			t.Errorf("Invalid GID: %d", gid)
		}

		fi, err = os.Stat(dirpath)
		if err != nil {
			t.Errorf("Failed to stat dir: %v", err)
			return
		}
		if !fi.IsDir() {
			t.Errorf("dir isn't dir!")
		}
		if uid := fi.Sys().(*syscall.Stat_t).Uid; uid != 234 {
			t.Errorf("Invalid UID: %d", uid)
		}
		if gid := fi.Sys().(*syscall.Stat_t).Gid; gid != 567 {
			t.Errorf("Invalid GID: %d", gid)
		}
	})
}

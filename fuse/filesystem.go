package fuse

import (
	"context"
	"fmt"
	"math"
	"time"

	bfuse "github.com/nyaxt/fuse"
	bfs "github.com/nyaxt/fuse/fs"
	"go.uber.org/zap"

	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("fuse")
var bfuseLogger = logger.Registry().Category("bfuse")

func init() {
	bfuse.Debug = func(msg interface{}) { zap.S().Debugf("%v", msg) }
}

type FileSystem struct {
	ofs *filesystem.FileSystem
}

func (fs FileSystem) Root() (bfs.Node, error) {
	return DirNode{fs: fs.ofs, id: inodedb.RootDirID}, nil
}

func (fs FileSystem) Statfs(ctx context.Context, req *bfuse.StatfsRequest, resp *bfuse.StatfsResponse) error {
	// fill dummy
	resp.Blocks = 0
	if tsize, err := fs.ofs.TotalSize(); err != nil {
		resp.Blocks = uint64(tsize)
	}
	resp.Bfree = math.MaxUint64
	resp.Bavail = math.MaxUint64
	resp.Files = 0
	resp.Ffree = 0
	resp.Bsize = 32 * 1024
	resp.Namelen = 32 * 1024
	resp.Frsize = 1

	return nil
}

func Serve(ctx context.Context, bucketName string, mountpoint string, ofs *filesystem.FileSystem, ready chan<- bool) error {
	fsName := fmt.Sprintf("otaru+gs://%s", bucketName)
	volName := fmt.Sprintf("Otaru %s", bucketName)

	c, err := bfuse.Mount(
		mountpoint,
		bfuse.FSName(fsName),
		bfuse.Subtype("otarufs"),
		bfuse.VolumeName(volName),
		bfuse.MaxReadahead(math.MaxUint32),
	)
	if err != nil {
		return fmt.Errorf("bfuse.Mount failed: %v", err)
	}
	defer c.Close()

	serveC := make(chan error)
	go func() {
		if err := bfs.Serve(c, FileSystem{ofs}); err != nil {
			serveC <- err
		}
		close(serveC)
	}()

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}

	zap.S().Infof("Mountpoint \"%s\" should be ready now!", mountpoint)
	if ready != nil {
		close(ready)
	}

	go func() {
		<-ctx.Done()
		Unmount(mountpoint)
	}()

	return <-serveC
}

func Unmount(mountpoint string) {
	doneC := make(chan struct{})
	go func() {
		bfuse.Unmount(mountpoint)
		close(doneC)
	}()
	timeoutC := time.After(time.Second * 3)
	select {
	case <-doneC:
		zap.S().Infof("Successfully unmounted: %v", mountpoint)
	case <-timeoutC:
		zap.S().Warnf("Timeout reached while trying to unmount: %v", mountpoint)
	}
}

package fuse

import (
	"fmt"
	"math"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"

	bfuse "github.com/nyaxt/fuse"
	bfs "github.com/nyaxt/fuse/fs"
	"golang.org/x/net/context"
)

var mylog = logger.Registry().Category("fuse")

type FileSystem struct {
	ofs *otaru.FileSystem
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

func ServeFUSE(bucketName string, mountpoint string, ofs *otaru.FileSystem, ready chan<- bool) error {
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
			close(serveC)
			return
		}
		close(serveC)
	}()

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}

	logger.Infof(mylog, "Mountpoint \"%s\" should be ready now!", mountpoint)
	if ready != nil {
		close(ready)
	}

	if err := <-serveC; err != nil {
		return nil
	}
	if err := ofs.Sync(); err != nil {
		return fmt.Errorf("Failed to Sync fs: %v", err)
	}
	return nil
}

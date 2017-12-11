package fuse

import (
	"fmt"
	"math"
	"os"
	"syscall"
	"time"

	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"

	bfuse "github.com/nyaxt/fuse"
	bfs "github.com/nyaxt/fuse/fs"
	"golang.org/x/net/context"
)

const (
	EBADF = syscall.Errno(syscall.EBADF)
)

type FileNode struct {
	fs *filesystem.FileSystem
	id inodedb.ID
}

func (n FileNode) Attr(ctx context.Context, a *bfuse.Attr) error {
	const blockSize = 512

	attr, err := n.fs.Attr(n.id)
	if err != nil {
		panic("fs.Attr failed for FileNode")
	}

	a.Valid = 1 * time.Minute
	a.Inode = uint64(n.id)
	a.Size = uint64(attr.Size)
	a.Blocks = a.Size / blockSize
	a.Mode = os.FileMode(attr.PermMode) & os.ModePerm
	a.Atime = attr.ModifiedT
	a.Mtime = attr.ModifiedT
	a.Ctime = attr.ModifiedT
	a.Crtime = attr.ModifiedT
	a.Nlink = 1
	a.Uid = attr.Uid
	a.Gid = attr.Gid
	return nil
}

func (n FileNode) Getattr(ctx context.Context, req *bfuse.GetattrRequest, resp *bfuse.GetattrResponse) error {
	return n.Attr(ctx, &resp.Attr)
}

func (n FileNode) Setattr(ctx context.Context, req *bfuse.SetattrRequest, resp *bfuse.SetattrResponse) error {
	if req.Valid.Size() {
		logger.Debugf(mylog, "Setattr size %d", req.Size)
		if req.Size > math.MaxInt64 {
			return fmt.Errorf("specified size too big: %d", req.Size)
		}
		if err := n.fs.TruncateFile(n.id, int64(req.Size)); err != nil {
			return err
		}
	}

	if err := otaruSetattr(n.fs, n.id, req); err != nil {
		return err
	}

	if err := n.Attr(ctx, &resp.Attr); err != nil {
		return err
	}

	return nil
}

func (n FileNode) Open(ctx context.Context, req *bfuse.OpenRequest, resp *bfuse.OpenResponse) (bfs.Handle, error) {
	logger.Debugf(mylog, "Open flags: %s", req.Flags.String())

	fh, err := n.fs.OpenFile(n.id, Bazil2OtaruFlags(req.Flags))
	if err != nil {
		return nil, err
	}

	return FileHandle{fh}, nil
}

func (n FileNode) Fsync(ctx context.Context, req *bfuse.FsyncRequest) error {
	if err := n.fs.SyncFile(n.id); err != nil {
		return err
	}
	return nil
}

type FileHandle struct {
	h *filesystem.FileHandle
}

func (fh FileHandle) Read(ctx context.Context, req *bfuse.ReadRequest, resp *bfuse.ReadResponse) error {
	logger.Debugf(mylog, "Read offset %d size %d", req.Offset, req.Size)

	if fh.h == nil {
		return EBADF
	}

	resp.Data = resp.Data[:req.Size]
	n, err := fh.h.ReadAt(resp.Data, req.Offset)
	if err != nil {
		return err
	}
	resp.Data = resp.Data[:n]

	return nil
}

func (fh FileHandle) Write(ctx context.Context, req *bfuse.WriteRequest, resp *bfuse.WriteResponse) error {
	logger.Debugf(mylog, "Write offset %d size %d", req.Offset, len(req.Data))

	if fh.h == nil {
		return EBADF
	}

	if err := fh.h.PWrite(req.Data, req.Offset); err != nil {
		return err
	}
	resp.Size = len(req.Data)

	return nil
}

func (fh FileHandle) Flush(ctx context.Context, req *bfuse.FlushRequest) error {
	if fh.h == nil {
		return EBADF
	}

	if err := fh.h.Sync(); err != nil {
		return err
	}
	return nil
}

func (fh FileHandle) Release(ctx context.Context, req *bfuse.ReleaseRequest) error {
	fh.Forget()
	return nil
}

func (fh FileHandle) Forget() {
	if fh.h == nil {
		return
	}
	fh.h.Close()
	fh.h = nil
}

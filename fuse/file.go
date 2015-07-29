package fuse

import (
	"fmt"
	"log"
	"math"
	"os"
	"syscall"

	"github.com/nyaxt/otaru"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"

	bfuse "bazil.org/fuse"
	bfs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

const (
	EBADF = syscall.Errno(syscall.EBADF)
)

type FileNode struct {
	fs *otaru.FileSystem
	id inodedb.ID
}

func (n FileNode) Attr(ctx context.Context, a *bfuse.Attr) error {
	attr, err := n.fs.Attr(n.id)
	if err != nil {
		panic("fs.Attr failed for FileNode")
	}

	a.Inode = uint64(n.id)
	a.Mode = os.FileMode(attr.PermMode) & os.ModePerm
	a.Atime = attr.ModifiedT
	a.Mtime = attr.ModifiedT
	a.Ctime = attr.ModifiedT
	a.Crtime = attr.ModifiedT
	a.Size = uint64(attr.Size)
	a.Uid = attr.Uid
	a.Gid = attr.Gid
	return nil
}

func Bazil2OtaruFlags(bf bfuse.OpenFlags) int {
	ret := 0
	if bf.IsReadOnly() {
		ret = oflags.O_RDONLY
	} else if bf.IsWriteOnly() {
		ret = oflags.O_WRONLY
	} else if bf.IsReadWrite() {
		ret = oflags.O_RDWR
	}

	if bf&bfuse.OpenAppend != 0 {
		ret |= oflags.O_APPEND
	}
	if bf&bfuse.OpenCreate != 0 {
		ret |= oflags.O_CREATE
	}
	if bf&bfuse.OpenExclusive != 0 {
		ret |= oflags.O_EXCL
	}
	if bf&bfuse.OpenSync != 0 {
		log.Printf("FIXME: OpenSync not supported yet !!!!!!!!!!!")
	}
	if bf&bfuse.OpenTruncate != 0 {
		ret |= oflags.O_TRUNCATE
	}

	return ret
}

func (n FileNode) Open(ctx context.Context, req *bfuse.OpenRequest, resp *bfuse.OpenResponse) (bfs.Handle, error) {
	log.Printf("Open flags: %s", req.Flags.String())

	fh, err := n.fs.OpenFile(n.id, Bazil2OtaruFlags(req.Flags))
	if err != nil {
		return nil, err
	}

	return FileHandle{fh}, nil
}

type FileHandle struct {
	h *otaru.FileHandle
}

func (fh FileHandle) Read(ctx context.Context, req *bfuse.ReadRequest, resp *bfuse.ReadResponse) error {
	log.Printf("Read offset %d size %d", req.Offset, req.Size)

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
	log.Printf("Write offset %d size %d", req.Offset, len(req.Data))

	if fh.h == nil {
		return EBADF
	}

	if err := fh.h.PWrite(req.Data, req.Offset); err != nil {
		return err
	}
	resp.Size = len(req.Data)

	return nil
}

// FIXME: move this to FileNode
func (fh FileHandle) Setattr(ctx context.Context, req *bfuse.SetattrRequest, resp *bfuse.SetattrResponse) error {
	if fh.h == nil {
		return EBADF
	}

	if req.Valid.Size() {
		log.Printf("Setattr size %d", req.Size)
		if req.Size > math.MaxInt64 {
			return fmt.Errorf("too big")
		}
		if err := fh.h.Truncate(int64(req.Size)); err != nil {
			return err
		}
	}

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

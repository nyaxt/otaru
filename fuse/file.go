package fuse

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/nyaxt/otaru"

	bfuse "bazil.org/fuse"
	bfs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type FileNode struct {
	fs *otaru.FileSystem
	id otaru.INodeID
}

func (n FileNode) Attr(a *bfuse.Attr) {
	attr, err := n.fs.Attr(n.id)
	if err != nil {
		panic("fs.Attr failed for FileNode")
	}

	a.Inode = uint64(n.id)
	a.Mode = 0666
	a.Atime = time.Now()
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	a.Crtime = time.Now()
	a.Size = uint64(attr.Size)
}

func (n FileNode) Open(ctx context.Context, req *bfuse.OpenRequest, resp *bfuse.OpenResponse) (bfs.Handle, error) {
	fh, err := n.fs.OpenFile(n.id)
	if err != nil {
		return nil, err
	}

	return FileHandle{fh}, nil
}

type FileHandle struct {
	h *otaru.FileHandle
}

func (fh FileHandle) Read(ctx context.Context, req *bfuse.ReadRequest, resp *bfuse.ReadResponse) error {
	/*
	     Header:
	       Conn *Conn     `json:"-"` // connection this request was received on
	       ID   RequestID // unique ID for request
	       Node NodeID    // file or directory the request is about
	       Uid  uint32    // user ID of process making request
	       Gid  uint32    // group ID of process making request
	       Pid  uint32    // process ID of process making request

	   	Handle HandleID
	   	Offset int64
	   	Size   int
	*/
	log.Printf("Read offset %d size %d", req.Offset, req.Size)

	resp.Data = resp.Data[:req.Size]
	if err := fh.h.PRead(req.Offset, resp.Data); err != nil {
		return err
	}

	return nil
}

func (fh FileHandle) Write(ctx context.Context, req *bfuse.WriteRequest, resp *bfuse.WriteResponse) error {
	log.Printf("Write offset %d size %d", req.Offset, len(req.Data))

	if err := fh.h.PWrite(req.Offset, req.Data); err != nil {
		return err
	}
	resp.Size = len(req.Data)

	return nil
}

// FIXME: move this to FileNode
func (fh FileHandle) Setattr(ctx context.Context, req *bfuse.SetattrRequest, resp *bfuse.SetattrResponse) error {
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
	if err := fh.h.Flush(); err != nil {
		return err
	}
	return nil
}

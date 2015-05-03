package fuse

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nyaxt/otaru"

	bfuse "bazil.org/fuse"
	bfs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type DirNode struct {
	dh *otaru.DirHandle
}

func (d DirNode) Attr(a *bfuse.Attr) {
	id := d.dh.ID()
	log.Printf("DirNode Attr id: %d", id)

	attr, err := d.dh.FileSystem().Attr(id)
	if err != nil {
		panic("bfs.Attr failed for DirNode")
	}

	a.Inode = uint64(id)
	a.Mode = os.ModeDir | 0777
	a.Atime = time.Now()
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	a.Crtime = time.Now()
	a.Size = uint64(attr.Size)
}

func (d DirNode) Lookup(ctx context.Context, name string) (bfs.Node, error) {
	entries := d.dh.Entries()
	if id, ok := entries[name]; ok {
		isdir, err := d.dh.FileSystem().IsDir(id)
		if err != nil {
			log.Fatalf("Stale inode in dir? Failed IsDir: %v", err)
		}
		if isdir {
			childdh, err := d.dh.FileSystem().OpenDir(id)
			if err != nil {
				return nil, err
			}
			return DirNode{childdh}, nil
		} else {
			return FileNode{d.dh.FileSystem(), id}, nil
		}
	}

	return nil, bfuse.ENOENT
}

func (d DirNode) Create(ctx context.Context, req *bfuse.CreateRequest, resp *bfuse.CreateResponse) (bfs.Node, bfs.Handle, error) {
	id, err := d.dh.CreateFile(req.Name) // req.Flags req.Mode
	if err != nil {
		return nil, nil, err
	}

	h, err := d.dh.FileSystem().OpenFile(id)
	if err != nil {
		return nil, nil, err
	}

	return FileNode{d.dh.FileSystem(), id}, FileHandle{h}, nil
}

func (d DirNode) ReadDirAll(ctx context.Context) ([]bfuse.Dirent, error) {
	entries := d.dh.Entries()

	fentries := make([]bfuse.Dirent, 0, len(entries))
	for name, id := range entries {
		t := bfuse.DT_File // FIXME!!!

		fentries = append(fentries, bfuse.Dirent{
			Inode: uint64(id),
			Name:  name,
			Type:  t,
		})
	}
	return fentries, nil
}

func (d DirNode) Rename(ctx context.Context, req *bfuse.RenameRequest, newDir bfs.Node) error {
	newdn, ok := newDir.(DirNode)
	if !ok {
		return fmt.Errorf("Node for provided target dir is not DirNode!")
	}

	if err := d.dh.Rename(req.OldName, newdn.dh, req.NewName); err != nil {
		return err
	}

	return nil
}

func (d DirNode) Remove(ctx context.Context, req *bfuse.RemoveRequest) error {
	if err := d.dh.Remove(req.Name); err != nil {
		return err
	}

	return nil
}

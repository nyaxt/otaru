package fuse

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/inodedb"

	bfuse "bazil.org/fuse"
	bfs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type DirNode struct {
	fs *otaru.FileSystem
	id inodedb.ID
}

func (d DirNode) Attr(ctx context.Context, a *bfuse.Attr) error {
	attr, err := d.fs.Attr(d.id)
	if err != nil {
		panic("bfs.Attr failed for DirNode")
	}

	a.Valid = 1 * time.Minute
	a.Nlink = 1
	a.Inode = uint64(d.id)
	a.Mode = os.ModeDir | (os.FileMode(attr.PermMode) & os.ModePerm)
	a.Atime = attr.ModifiedT
	a.Mtime = attr.ModifiedT
	a.Ctime = attr.ModifiedT
	a.Crtime = attr.ModifiedT
	a.Size = uint64(attr.Size)
	a.Uid = attr.Uid
	a.Gid = attr.Gid

	return nil
}

func (d DirNode) Getattr(ctx context.Context, req *bfuse.GetattrRequest, resp *bfuse.GetattrResponse) error {
	return d.Attr(ctx, &resp.Attr)
}

func (d DirNode) Setattr(ctx context.Context, req *bfuse.SetattrRequest, resp *bfuse.SetattrResponse) error {
	log.Printf("Setattr mode %o", req.Mode)

	var valid otaru.ValidAttrFields
	var a otaru.Attr

	if req.Valid.Mode() {
		valid |= otaru.PermModeValid
		a.PermMode = uint16(req.Mode & os.ModePerm)
	}

	if valid != 0 {
		if err := d.fs.SetAttr(d.id, a, valid); err != nil {
			return err
		}
	}

	if err := d.Attr(ctx, &resp.Attr); err != nil {
		return err
	}

	return nil
}

func (d DirNode) Lookup(ctx context.Context, name string) (bfs.Node, error) {
	entries, err := d.fs.DirEntries(d.id)
	if err != nil {
		return nil, err
	}

	if id, ok := entries[name]; ok {
		isdir, err := d.fs.IsDir(id)
		if err != nil {
			log.Fatalf("Stale inode in dir? Failed IsDir: %v", err)
		}
		if isdir {
			return DirNode{d.fs, id}, nil
		} else {
			return FileNode{d.fs, id}, nil
		}
	}

	return nil, bfuse.ENOENT
}

func (d DirNode) Create(ctx context.Context, req *bfuse.CreateRequest, resp *bfuse.CreateResponse) (bfs.Node, bfs.Handle, error) {
	if req.Mode&os.ModeType != 0 {
		// Disallow creating dir/symlink/pipe/socket/device
		return nil, nil, bfuse.EPERM
	}

	permmode := uint16(req.Mode &^ req.Umask & os.ModePerm)
	id, err := d.fs.CreateFile(d.id, req.Name, permmode, req.Uid, req.Gid, time.Now())
	if err != nil {
		return nil, nil, err
	}

	h, err := d.fs.OpenFile(id, Bazil2OtaruFlags(req.Flags))
	if err != nil {
		return nil, nil, err
	}

	return FileNode{d.fs, id}, FileHandle{h}, nil
}

func (d DirNode) ReadDirAll(ctx context.Context) ([]bfuse.Dirent, error) {
	parentID, err := d.fs.ParentID(d.id)
	if err != nil {
		return nil, err
	}

	entries, err := d.fs.DirEntries(d.id)
	if err != nil {
		return nil, err
	}

	fentries := make([]bfuse.Dirent, 0, len(entries)+2)
	fentries = append(fentries, bfuse.Dirent{Inode: uint64(d.id), Name: ".", Type: bfuse.DT_Dir})
	fentries = append(fentries, bfuse.Dirent{Inode: uint64(parentID), Name: "..", Type: bfuse.DT_Dir})
	for name, id := range entries {
		isdir, err := d.fs.IsDir(id)
		if err != nil {
			log.Printf("Error while querying IsDir for id %d: %v", id, err)
		}

		var t bfuse.DirentType
		if isdir {
			t = bfuse.DT_Dir
		} else {
			t = bfuse.DT_File
		}

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

	if err := d.fs.Rename(d.id, req.OldName, newdn.id, req.NewName); err != nil {
		return err
	}

	return nil
}

func (d DirNode) Remove(ctx context.Context, req *bfuse.RemoveRequest) error {
	if err := d.fs.Remove(d.id, req.Name); err != nil {
		return err
	}

	return nil
}

func (d DirNode) Mkdir(ctx context.Context, req *bfuse.MkdirRequest) (bfs.Node, error) {
	permmode := uint16(req.Mode &^ req.Umask & os.ModePerm)
	id, err := d.fs.CreateDir(d.id, req.Name, permmode, req.Uid, req.Gid, time.Now())
	if err != nil {
		return nil, err
	}

	return DirNode{fs: d.fs, id: id}, nil
}

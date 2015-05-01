package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"

	"github.com/nyaxt/otaru"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	Key = []byte("0123456789abcdef")
)
var Cipher otaru.Cipher

type FileNode struct {
	fs *otaru.FileSystem
	id otaru.INodeID
}

func (n FileNode) Attr(a *fuse.Attr) {
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

func (n FileNode) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fh, err := n.fs.OpenFile(n.id)
	if err != nil {
		return nil, err
	}

	return FileHandle{fh}, nil
}

type FileHandle struct {
	h *otaru.FileHandle
}

func (fh FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
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

func (fh FileHandle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	log.Printf("Write offset %d size %d", req.Offset, len(req.Data))

	if err := fh.h.PWrite(req.Offset, req.Data); err != nil {
		return err
	}
	resp.Size = len(req.Data)

	return nil
}

// FIXME: move this to FileNode
func (fh FileHandle) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
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

func (fh FileHandle) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	if err := fh.h.Flush(); err != nil {
		return err
	}
	return nil
}

type DirNode struct {
	dh *otaru.DirHandle
}

func (d DirNode) Attr(a *fuse.Attr) {
	id := d.dh.ID()
	log.Printf("DirNode Attr id: %d", id)

	attr, err := d.dh.FileSystem().Attr(id)
	if err != nil {
		panic("fs.Attr failed for DirNode")
	}

	a.Inode = uint64(id)
	a.Mode = os.ModeDir | 0777
	a.Atime = time.Now()
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	a.Crtime = time.Now()
	a.Size = uint64(attr.Size)
}

func (d DirNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
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

	return nil, fuse.ENOENT
}

func (d DirNode) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	h, err := d.dh.CreateFile(req.Name) // req.Flags req.Mode
	if err != nil {
		return nil, nil, err
	}

	fn := FileNode{d.dh.FileSystem(), h.ID()}
	fh := FileHandle{h}

	return fn, fh, nil
}

func (d DirNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries := d.dh.Entries()

	fentries := make([]fuse.Dirent, 0, len(entries))
	for name, id := range entries {
		t := fuse.DT_File // FIXME!!!

		fentries = append(fentries, fuse.Dirent{
			Inode: uint64(id),
			Name:  name,
			Type:  t,
		})
	}
	return fentries, nil
}

type FileSystem struct {
	ofs *otaru.FileSystem
}

func (fs FileSystem) Root() (fs.Node, error) {
	rootdir, err := fs.ofs.OpenDir(1)
	if err != nil {
		log.Fatalf("Failed to open rootdir: %v", err)
	}
	return DirNode{rootdir}, nil
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	fuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}

	var err error
	Cipher, err = otaru.NewCipher(Key)
	if err != nil {
		log.Fatalf("Failed to init Cipher: %v", err)
	}

	bs, err := otaru.NewFileBlobStore("/tmp/otaru")
	if err != nil {
		log.Fatalf("NewFileBlobStore failed: %v", err)
		return
	}
	ofs := otaru.NewFileSystem(bs, Cipher)

	log.Printf("arg: %v", flag.Args())
	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("otaru"),
		fuse.Subtype("otarufs"),
		fuse.VolumeName("Otaru"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, FileSystem{ofs})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

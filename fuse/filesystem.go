package fuse

import (
	"log"

	"github.com/nyaxt/otaru"

	bfuse "bazil.org/fuse"
	bfs "bazil.org/fuse/fs"
)

type FileSystem struct {
	ofs *otaru.FileSystem
}

func (fs FileSystem) Root() (bfs.Node, error) {
	rootdir, err := fs.ofs.OpenDir(1)
	if err != nil {
		log.Fatalf("Failed to open rootdir: %v", err)
	}
	return DirNode{rootdir}, nil
}

func ServeFUSE(mountpoint string, ofs *otaru.FileSystem) error {
	c, err := bfuse.Mount(
		mountpoint,
		bfuse.FSName("otaru"),
		bfuse.Subtype("otarufs"),
		bfuse.VolumeName("Otaru"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = bfs.Serve(c, FileSystem{ofs})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}

	if err := ofs.Sync(); err != nil {
		log.Fatalf("Failed to Sync fs: %v", err)
	}
	return nil
}

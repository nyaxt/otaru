package fuse

import (
	"fmt"
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

func ServeFUSE(mountpoint string, ofs *otaru.FileSystem, ready chan<- bool) error {
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

	log.Printf("Mountpoint \"%s\" should be ready now!", mountpoint)
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

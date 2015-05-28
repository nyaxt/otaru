package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/fuse"
	"github.com/nyaxt/otaru/util"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	flagMkfs         = flag.Bool("mkfs", false, "Reset metadata if no existing metadata exists")
	flagPasswordFile = flag.String("passwordfile", path.Join(os.Getenv("HOME"), ".otaru", "password"), "Path of a text file storing password")
	flagCacheDir     = flag.String("cachedir", "/var/cache/otaru", "Path to blob cache dir")
)

func main() {
	flag.Usage = Usage
	flag.Parse()

	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}

	password, err := util.StringFromFile(*flagPasswordFile)
	if err != nil {
		log.Fatalf("Failed to load encryption password: %v", err)
	}
	key := otaru.KeyFromPassword(password)
	cipher, err := otaru.NewCipher(key)
	if err != nil {
		log.Fatalf("Failed to init Cipher: %v", err)
	}
	bs, err := blobstore.NewFileBlobStore(*flagCacheDir, otaru.O_RDWRCREATE)
	if err != nil {
		log.Fatalf("NewFileBlobStore failed: %v", err)
		return
	}
	ofs, err := otaru.NewFileSystemFromSnapshot(bs, cipher)
	if err != nil {
		if err == otaru.ENOENT && *flagMkfs {
			ofs, err := otaru.NewFileSystemEmpty(bs, cipher)
			if err != nil {
				log.Fatalf("NewFileSystemEmpty failed: %v", err)
			}
		} else {
			log.Fatalf("NewFileSystemFromSnapshot failed: %v", err)
		}
	}

	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	if err := fuse.ServeFUSE(mountpoint, ofs, nil); err != nil {
		log.Fatalf("ServeFUSE failed: %v", err)
	}
}

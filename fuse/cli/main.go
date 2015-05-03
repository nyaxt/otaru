package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/fuse"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	Key      = []byte("0123456789abcdef")
	flagMkfs = flag.Bool("mkfs", false, "Reset metadata if no existing metadata exists")
)

func main() {
	flag.Usage = Usage
	flag.Parse()

	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}

	cipher, err := otaru.NewCipher(Key)
	if err != nil {
		log.Fatalf("Failed to init Cipher: %v", err)
	}

	bs, err := otaru.NewFileBlobStore("/tmp/otaru", otaru.O_RDWR)
	if err != nil {
		log.Fatalf("NewFileBlobStore failed: %v", err)
		return
	}
	ofs, err := otaru.NewFileSystemFromSnapshot(bs, cipher)
	if err != nil {
		if err == otaru.ENOENT && *flagMkfs {
			ofs = otaru.NewFileSystemEmpty(bs, cipher)
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

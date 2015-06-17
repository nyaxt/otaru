package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru/facade"
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
	flagPasswordFile = flag.String("passwordFile", path.Join(os.Getenv("HOME"), ".otaru", "password.txt"), "Path of a text file storing password")
	flagProjectName  = flag.String("projectName", "", "google cloud project name")
	flagBucketName   = flag.String("bucketName", "", "google cloud storage bucket name")
	flagCacheDir     = flag.String("cachedir", "/var/cache/otaru", "Path to blob cache dir")
	flagLocalDebug   = flag.Bool("localDebug", false, "Use local filesystem instead of GCP (for offline debug purposes)")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	flag.Usage = Usage
	flag.Parse()

	password := util.StringFromFileOrDie(*flagPasswordFile, "password")
	if *flagProjectName == "" {
		log.Printf("Please specify a valid project name")
		Usage()
		os.Exit(2)
	}
	if *flagBucketName == "" {
		log.Printf("Please specify a valid bucket name")
		Usage()
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	o, err := facade.NewOtaru(*flagMkfs, password, *flagProjectName, *flagBucketName, *flagCacheDir, *flagLocalDebug)
	if err != nil {
		log.Printf("NewOtaru failed: %v", err)
		os.Exit(1)
	}
	closeOtaruAndExit := func() {
		if err := bfuse.Unmount(mountpoint); err != nil {
			log.Printf("umount err: %v", err)
		}
		if o != nil {
			if err := o.Close(); err != nil {
				log.Printf("Otaru.Close() returned errs: %v", err)
			}
		}
		os.Exit(0)
	}
	defer closeOtaruAndExit()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)
	signal.Notify(sigC, syscall.SIGTERM)
	go func() {
		for s := range sigC {
			log.Printf("Received signal: %v", s)
			closeOtaruAndExit()
		}
	}()

	bfuse.Debug = func(msg interface{}) {
		log.Printf("fusedbg: %v", msg)
	}
	if err := fuse.ServeFUSE(mountpoint, o.FS, nil); err != nil {
		log.Fatalf("ServeFUSE failed: %v", err)
	}
	log.Printf("ServeFUSE end!")
}

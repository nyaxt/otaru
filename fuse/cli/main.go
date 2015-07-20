package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	bfuse "bazil.org/fuse"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/fuse"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	flagMkfs      = flag.Bool("mkfs", false, "Reset metadata if no existing metadata exists")
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	flag.Usage = Usage
	flag.Parse()

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		log.Printf("%v", err)
		Usage()
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	o, err := facade.NewOtaru(cfg, &facade.OneshotConfig{Mkfs: *flagMkfs})
	if err != nil {
		log.Printf("NewOtaru failed: %v", err)
		os.Exit(1)
	}
	var muClose sync.Mutex
	closeOtaruAndExit := func() {
		muClose.Lock()
		defer muClose.Unlock()

		if err := bfuse.Unmount(mountpoint); err != nil {
			log.Printf("umount err: %v", err)
		}
		if o != nil {
			if err := o.Close(); err != nil {
				log.Printf("Otaru.Close() returned errs: %v", err)
			}
			o = nil
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
		o.Close()
		log.Fatalf("ServeFUSE failed: %v", err)
	}
	log.Printf("ServeFUSE end!")
}

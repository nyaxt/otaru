package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	bfuse "github.com/nyaxt/fuse"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/fuse"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/version"
)

var mylog = logger.Registry().Category("otaru-mount")

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	flagVersion   = flag.Bool("version", false, "Show version info")
	flagMkfs      = flag.Bool("mkfs", false, "Reset metadata if no existing metadata exists")
	flagReadOnly  = flag.Bool("readonly", false, "Mount as read-only mode. No changes to the filesystem is allowed.")
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

var bfuseLogger = logger.Registry().Category("bfuse")

func main() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	flag.Usage = Usage
	flag.Parse()

	if *flagVersion {
		fmt.Print(version.DumpBuildInfo())
		os.Exit(1)
	}

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		logger.Criticalf(mylog, "%v", err)
		Usage()
		os.Exit(2)
	}
	if *flagReadOnly {
		cfg.ReadOnly = true
	}
	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	if err := facade.SetupFluentLogger(cfg); err != nil {
		logger.Criticalf(mylog, "Failed to setup fluentd logger: %v", err)
		os.Exit(1)
	}

	o, err := facade.NewOtaru(cfg, &facade.OneshotConfig{Mkfs: *flagMkfs})
	if err != nil {
		logger.Criticalf(mylog, "NewOtaru failed: %v", err)
		os.Exit(1)
	}
	var muClose sync.Mutex
	closeOtaruAndExit := func(exitCode int) {
		muClose.Lock()
		defer muClose.Unlock()

		if err := bfuse.Unmount(mountpoint); err != nil {
			logger.Warningf(mylog, "umount err: %v", err)
		}
		if o != nil {
			if err := o.Close(); err != nil {
				logger.Warningf(mylog, "Otaru.Close() returned errs: %v", err)
			}
			o = nil
		}
		os.Exit(exitCode)
	}
	defer closeOtaruAndExit(0)

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)
	signal.Notify(sigC, syscall.SIGTERM)
	go func() {
		for s := range sigC {
			logger.Warningf(mylog, "Received signal: %v", s)
			closeOtaruAndExit(1)
		}
	}()
	logger.Registry().AddOutput(logger.HandleCritical(func() {
		logger.Warningf(mylog, "Starting shutdown due to critical event.")
		go closeOtaruAndExit(1)
	}))

	bfuse.Debug = func(msg interface{}) { logger.Debugf(bfuseLogger, "%v", msg) }
	if err := fuse.ServeFUSE(cfg.BucketName, mountpoint, o.FS, nil); err != nil {
		logger.Warningf(mylog, "ServeFUSE failed: %v", err)
		closeOtaruAndExit(1)
	}
	logger.Infof(mylog, "ServeFUSE end!")
}

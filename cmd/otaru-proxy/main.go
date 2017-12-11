package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/version"
	"github.com/nyaxt/otaru/webdav"
)

var mylog = logger.Registry().Category("otaru-proxy")

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	flagVersion   = flag.Bool("version", false, "Show version info")
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

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
	cfg.ReadOnly = true
	if flag.NArg() != 0 {
		Usage()
		os.Exit(2)
	}

	o, err := facade.NewOtaru(cfg, &facade.OneshotConfig{Mkfs: false})
	if err != nil {
		logger.Criticalf(mylog, "NewOtaru failed: %v", err)
		os.Exit(1)
	}
	var muClose sync.Mutex
	closeOtaruAndExit := func(exitCode int) {
		muClose.Lock()
		defer muClose.Unlock()

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

	if err := webdav.Serve(o.FS); err != nil {
		logger.Warningf(mylog, "Serve failed: %v", err)
		closeOtaruAndExit(1)
	}
	logger.Infof(mylog, "Serve end!")
}

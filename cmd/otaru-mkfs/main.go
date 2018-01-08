package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/version"
)

var mylog = logger.Registry().Category("otaru-mkfs")

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	flagVersion   = flag.Bool("version", false, "Show version info")
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

func main() {
	flag.Usage = Usage
	flag.Parse()

	if *flagVersion {
		fmt.Print(version.DumpBuildInfo())
		os.Exit(1)
	}

	facade.BootstrapLogger()

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		logger.Criticalf(mylog, "%v", err)
		Usage()
		os.Exit(2)
	}
	if flag.NArg() != 0 {
		Usage()
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := facade.Mkfs(cfg); err != nil {
		logger.Warningf(mylog, "facade.Mkfs end: %v", err)
		return
	}
	logger.Infof(mylog, "facade.Mkfs end successfully!")
}

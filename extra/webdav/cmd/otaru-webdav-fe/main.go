package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/extra/webdav"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/version"
)

var (
	flagVersion   = flag.Bool("version", false, "Show version info")
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	flag.Parse()
	if *flagVersion {
		fmt.Print(version.DumpBuildInfo())
		os.Exit(1)
	}

	facade.BootstrapLogger()

	cfg, err := cli.NewConfig(*flagConfigDir)
	if err != nil {
		logger.Infof(cli.Log, "%v", err)
		Usage()
		os.Exit(2)
	}

	if err = webdav.Serve(cfg, nil); err != nil {
		logger.Criticalf(cli.Log, "%v", err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/extra/fuzzymv"
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
	fmt.Fprintf(os.Stderr, "  %s [options] update otaru://vhost\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] search otaru://vhost keyword\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	flag.Parse()
	if *flagVersion {
		fmt.Print(version.DumpBuildInfo())
		os.Exit(1)
	}
	if flag.NArg() < 1 {
		Usage()
		os.Exit(2)
	}

	facade.BootstrapLogger()

	cfg, err := cli.NewConfig(*flagConfigDir)
	if err != nil {
		logger.Infof(cli.Log, "%v", err)
		Usage()
		os.Exit(2)
	}

	ctx := context.Background() // FIXME

	switch flag.Arg(0) {
	case "update":
		err = fuzzymv.Update(ctx, cfg, flag.Args())

	case "search":
		err = fuzzymv.Search(ctx, cfg, flag.Args())

	default:
		logger.Infof(cli.Log, "Unknown cmd: %v", flag.Arg(0))
		Usage()
		os.Exit(2)
	}
	if err != nil {
		logger.Criticalf(cli.Log, "%v", err)
		os.Exit(1)
	}
}
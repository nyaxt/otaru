package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	fe_apiserver "github.com/nyaxt/otaru/extra/fe/apiserver"
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

func serve(cfg *cli.CliConfig) error {
	opts := fe_apiserver.BuildApiServerOptions(cfg)
	return apiserver.Serve(opts...)
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

	if err = serve(cfg); err != nil {
		logger.Criticalf(cli.Log, "%v", err)
		os.Exit(1)
	}
}

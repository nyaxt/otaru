package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("otaru-globallock")

var (
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
	flagForce     = flag.Bool("f", false, "Force Unlock currently active global lock when specified with unlock cmd")
	flagInfoStr   = flag.String("info", "", "Custom info string")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] {lock,unlock,query}\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	logger.Registry().AddOutput(logger.HandleCritical(func() { os.Exit(1) }))

	flag.Usage = Usage
	flag.Parse()

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		logger.Infof(mylog, "%v", err)
		Usage()
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	switch flag.Arg(0) {
	case "lock", "unlock", "query":
		break
	default:
		logger.Infof(mylog, "Unknown cmd: %v", flag.Arg(0))
		os.Exit(1)
	}

	tsrc, err := auth.GetGCloudTokenSource(context.Background(), cfg.CredentialsFilePath, cfg.TokenCacheFilePath, false)
	if err != nil {
		logger.Criticalf(mylog, "Failed to init GCloudClientSource: %v", err)
	}
	c := &btncrypt.Cipher{} // Null cipher is fine, as we GlobalLocker doesn't make use of it.
	dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, c, tsrc)
	info := *flagInfoStr
	if info == "" {
		info = "otaru-globallock-cli cmdline debug tool"
	}
	l := datastore.NewGlobalLocker(dscfg, "otaru-globallock-cli", info)

	switch flag.Arg(0) {
	case "lock":
		if err := l.Lock(); err != nil {
			logger.Infof(mylog, "Lock failed: %v", err)
		}
	case "unlock":
		if *flagForce {
			if err := l.ForceUnlock(); err != nil {
				logger.Infof(mylog, "ForceUnlock failed: %v", err)
				os.Exit(1)
			}
		} else {
			if err := l.UnlockIgnoreCreatedAt(); err != nil {
				logger.Infof(mylog, "Unlock failed: %v", err)
				os.Exit(1)
			}
		}
	case "query":
		entry, err := l.Query()
		if err != nil {
			logger.Infof(mylog, "Query failed: %v", err)
			os.Exit(1)
		}
		fmt.Printf("%+v\n", entry)

	default:
		panic("should not be reached")
	}
}

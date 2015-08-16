package main

import (
	"bufio"
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
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] {list,purge}\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	logger.Registry().AddOutput(logger.HandleCritical(func() { os.Exit(1) }))

	flag.Usage = Usage
	flag.Parse()
	if flag.NArg() != 1 {
		Usage()
		os.Exit(1)
	}
	switch flag.Arg(0) {
	case "list", "purge":
		break
	default:
		logger.Infof(mylog, "Unknown cmd: %v", flag.Arg(0))
		Usage()
		os.Exit(2)
	}

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		logger.Infof(mylog, "%v", err)
		Usage()
		os.Exit(1)
	}

	tsrc, err := auth.GetGCloudTokenSource(context.Background(), cfg.CredentialsFilePath, cfg.TokenCacheFilePath, false)
	if err != nil {
		logger.Criticalf(mylog, "Failed to init GCloudTokenSource: %v", err)
	}
	key := btncrypt.KeyFromPassword(cfg.Password)
	c, err := btncrypt.NewCipher(key)
	if err != nil {
		logger.Criticalf(mylog, "Failed to init btncrypt.Cipher: %v", err)
	}

	dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, c, tsrc)
	ssloc := datastore.NewINodeDBSSLocator(dscfg)
	switch flag.Arg(0) {
	case "purge":
		fmt.Printf("Do you really want to proceed with deleting all inodedbsslocator entry for %s?\n", cfg.BucketName)
		fmt.Printf("Type \"deleteall\" to proceed: ")
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			return
		}
		if sc.Text() != "deleteall" {
			logger.Infof(mylog, "Cancelled.\n")
			os.Exit(1)
		}

		es, err := ssloc.DeleteAll()
		if err != nil {
			logger.Infof(mylog, "DeleteAll failed: %v", err)
		}
		logger.Infof(mylog, "DeleteAll deleted entries for blobpath: %v", es)
		// FIXME: delete the entries from blobpath too

	case "list":
		history := 0
	histloop:
		for {
			bp, err := ssloc.Locate(history)
			if err != nil {
				if err == datastore.EEMPTY {
					logger.Infof(mylog, "Locate(%d): no entry", history)
				} else {
					logger.Infof(mylog, "Locate(%d) err: %v", history, err)
				}
				break histloop
			}
			logger.Infof(mylog, "Locate(%d): %v", history, bp)

			history++
		}
	default:
		panic("NOT REACHED")
	}
}

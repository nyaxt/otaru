package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"context"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"go.uber.org/zap"
)

var mylog = logger.Registry().Category("otaru-globallock")

var (
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
	flagDryRun    = flag.Bool("dryRun", false, "Don't actually make a change if set true")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] {list,purge}\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	panic("migrate to urfave/cli")

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
		zap.S().Infof("Unknown cmd: %v", flag.Arg(0))
		Usage()
		os.Exit(2)
	}

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		zap.S().Infof("%v", err)
		Usage()
		os.Exit(1)
	}

	tsrc, err := auth.GetGCloudTokenSource(cfg.CredentialsFilePath)
	if err != nil {
		zap.S().Errorf("Failed to init GCloudTokenSource: %v", err)
	}
	key := btncrypt.KeyFromPassword(cfg.Password)
	c, err := btncrypt.NewCipher(key)
	if err != nil {
		zap.S().Errorf("Failed to init *btncrypt.Cipher: %v", err)
	}

	dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, c, tsrc)
	ssloc := datastore.NewINodeDBSSLocator(dscfg, flags.O_RDWRCREATE)
	switch flag.Arg(0) {
	case "purge":
		fmt.Printf("Do you really want to proceed with deleting all inodedbsslocator entry for %s?\n", cfg.BucketName)
		fmt.Printf("Type \"deleteall\" to proceed: ")
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			return
		}
		if sc.Text() != "deleteall" {
			zap.S().Infof("Cancelled.\n")
			os.Exit(1)
		}

		es, err := ssloc.DeleteAll(context.Background(), *flagDryRun)
		if err != nil {
			zap.S().Infof("DeleteAll failed: %v", err)
		}
		zap.S().Infof("DeleteAll deleted entries for blobpath: %v", es)
		// FIXME: delete the entries from blobpath too

	case "list":
		history := 0
	histloop:
		for {
			bp, txid, err := ssloc.Locate(history)
			if err != nil {
				if err == datastore.EEMPTY {
					zap.S().Infof("Locate(%d): no entry", history)
				} else {
					zap.S().Infof("Locate(%d) err: %v", history, err)
				}
				break histloop
			}
			zap.S().Infof("Locate(%d) txid %v blobpath %v", history, inodedb.TxID(txid), bp)

			history++
		}
	default:
		panic("NOT REACHED")
	}
}

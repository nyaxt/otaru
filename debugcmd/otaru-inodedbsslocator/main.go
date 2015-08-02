package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
)

var (
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] {list,purge}\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

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
		log.Printf("Unknown cmd: %v", flag.Arg(0))
		Usage()
		os.Exit(2)
	}

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		log.Printf("%v", err)
		Usage()
		os.Exit(1)
	}

	clisrc, err := auth.GetGCloudClientSource(cfg.CredentialsFilePath, cfg.TokenCacheFilePath, false)
	if err != nil {
		log.Fatalf("Failed to init GCloudClientSource: %v", err)
	}
	key := btncrypt.KeyFromPassword(cfg.Password)
	c, err := btncrypt.NewCipher(key)
	if err != nil {
		log.Fatalf("Failed to init btncrypt.Cipher: %v", err)
	}

	dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, c, clisrc)
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
			log.Printf("Cancelled.\n")
			os.Exit(1)
		}

		es, err := ssloc.DeleteAll()
		if err != nil {
			log.Printf("DeleteAll failed: %v", err)
		}
		log.Printf("DeleteAll deleted entries for blobpath: %v", es)
		// FIXME: delete the entries from blobpath too

	case "list":
		history := 0
	histloop:
		for {
			bp, err := ssloc.Locate(history)
			if err != nil {
				if err == datastore.EEMPTY {
					log.Printf("Locate(%d): no entry", history)
				} else {
					log.Printf("Locate(%d) err: %v", history, err)
				}
				break histloop
			}
			log.Printf("Locate(%d): %v", history, bp)

			history++
		}
	default:
		panic("NOT REACHED")
	}
}
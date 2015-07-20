package main

import (
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
	flagForce     = flag.Bool("f", false, "Force Unlock currently active global lock when specified with unlock cmd")
	flagInfoStr   = flag.String("info", "", "Custom info string")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] {lock,unlock,query}\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	flag.Usage = Usage
	flag.Parse()

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		log.Printf("%v", err)
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
		log.Printf("Unknown cmd: %v", flag.Arg(0))
		os.Exit(1)
	}

	clisrc, err := auth.GetGCloudClientSource(cfg.CredentialsFilePath, cfg.TokenCacheFilePath, false)
	if err != nil {
		log.Fatalf("Failed to init GCloudClientSource: %v", err)
	}
	c := btncrypt.Cipher{} // Null cipher is fine, as we GlobalLocker doesn't make use of it.
	dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, c, clisrc)
	info := *flagInfoStr
	if info == "" {
		info = "otaru-globallock-cli cmdline debug tool"
	}
	l := datastore.NewGlobalLocker(dscfg, "otaru-globallock-cli", info)

	switch flag.Arg(0) {
	case "lock":
		if err := l.Lock(); err != nil {
			log.Printf("Lock failed: %v", err)
		}
	case "unlock":
		if *flagForce {
			if err := l.ForceUnlock(); err != nil {
				log.Printf("ForceUnlock failed: %v", err)
				os.Exit(1)
			}
		} else {
			if err := l.UnlockIgnoreCreatedAt(); err != nil {
				log.Printf("Unlock failed: %v", err)
				os.Exit(1)
			}
		}
	case "query":
		entry, err := l.Query()
		if err != nil {
			log.Printf("Query failed: %v", err)
			os.Exit(1)
		}
		fmt.Printf("%+v\n", entry)

	default:
		panic("should not be reached")
	}
}

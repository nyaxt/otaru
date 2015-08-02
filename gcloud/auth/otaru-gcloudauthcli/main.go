package main

import (
	"flag"
	"log"
	"os"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
)

var (
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

func main() {
	flag.Parse()

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		log.Printf("%v", err)
		os.Exit(2)
	}

	_, err = auth.GetGCloudTokenSource(context.Background(), cfg.CredentialsFilePath, cfg.TokenCacheFilePath, true)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	log.Printf("credentials valid!")
}

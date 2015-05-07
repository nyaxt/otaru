package main

import (
	"flag"
	"log"
	"os"
	"path"

	"github.com/nyaxt/otaru/gcs"
)

var (
	homedir = os.Getenv("HOME")

	projectName         = flag.String("project-name", "", "Gcloud project name")
	credentialsFilePath = flag.String("credentials", path.Join(homedir, ".otaru", "credentials.json"), "Credentials json path")
	tokenCacheFilePath  = flag.String("token-cache", path.Join(homedir, ".otaru", "tokencache.json"), "Token cache json path")
)

func main() {
	flag.Parse()

	_, err := gcs.GetGCloudClientSource(*credentialsFilePath, *tokenCacheFilePath, true)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	log.Printf("credentials valid!")
}

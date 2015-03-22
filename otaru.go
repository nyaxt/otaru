package otaru

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
)

var (
	projectName     = flag.String("project-name", "", "Gcloud project name")
	credentialsFile = flag.String("credentials", "credentials.json", "Credentials json path")
	tokenCacheFile  = flag.String("token-cache", "tokencache.json", "Token cache json path")
)

const (
	defaultBucketName = "otaru"
	storageScope      = storage.ScopeFullControl
)

func GetGCloudTokenViaWebUI(conf *oauth2.Config) (*oauth2.Token, error) {
	authurl := conf.AuthCodeURL("fixmeee", oauth2.AccessTypeOffline)
	fmt.Printf("visit %v\n", authurl)
	fmt.Printf("paste code:")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("Failed to scan auth code: %v", err)
	}
	initialToken, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Failed to use auth code: %v", err)
	}

	return initialToken, nil
}

func GetGCloudTokenCached() (*oauth2.Token, error) {
	tokenJson, err := ioutil.ReadFile(*tokenCacheFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read token cache file: %v", tokenCacheFile)
	}

	var token oauth2.Token
	if err = json.Unmarshal(tokenJson, &token); err != nil {
		return nil, fmt.Errorf("Failed to parse token cache file: %v", tokenCacheFile)
	}

	if token.Expiry.Before(time.Now()) {
		return nil, fmt.Errorf("Cached token already expired: %v", token.Expiry)
	}

	return &token, nil
}

func UpdateGCloudTokenCache(token *oauth2.Token) {
	tjson, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Serializing token failed: %v", err)
	}

	if err = ioutil.WriteFile(*tokenCacheFile, tjson, 0600); err != nil {
		log.Printf("Writing token cache failed: %v", err)
	}
}

func GetGCloudClient() *http.Client {
	credentialsJson, err := ioutil.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("failed to read google cloud client-secret file: %v", err)
	}

	conf, err := google.ConfigFromJSON(credentialsJson, storageScope)
	if err != nil {
		log.Fatalf("invalid google cloud key json: %v", err)
	}

	var initialToken *oauth2.Token
	if initialToken, err = GetGCloudTokenCached(); err != nil {
		if initialToken, err = GetGCloudTokenViaWebUI(conf); err != nil {
			log.Fatalf("Failed to get valid gcloud token: %v", err)
		}
		UpdateGCloudTokenCache(initialToken)
	}

	return conf.Client(oauth2.NoContext, initialToken)
}

func NewGCloudContext() context.Context {
	return cloud.NewContext(*projectName, GetGCloudClient())
}

type GCSBlobStore struct {
	bucketName string
	ctx        context.Context
}

func NewGCSBlobStore() GCSBlobStore {
	return GCSBlobStore{ctx: NewGCloudContext(), bucketName: defaultBucketName}
}

func (bs *GCSBlobStore) Put() {
	wc := storage.NewWriter(bs.ctx, bs.bucketName, "hoge2.txt")
	wc.ContentType = "application/octet-stream"
	wc.ACL = []storage.ACLRule{{storage.AllUsers, storage.RoleReader}}
	if _, err := wc.Write([]byte("hogefugapiyo")); err != nil {
		log.Fatalf("failed to write to storage Writer: %v", err)
	}
	if err := wc.Close(); err != nil {
		log.Fatalf("failed to close storage Writer: %v", err)
	}
	log.Println("ok")
}

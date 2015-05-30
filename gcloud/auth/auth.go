package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud/datastore"
	"google.golang.org/cloud/storage"
)

func GetGCloudTokenViaWebUI(conf *oauth2.Config) (*oauth2.Token, error) {
	authurl := conf.AuthCodeURL("fixmeee", oauth2.AccessTypeOffline)
	fmt.Printf("visit %v\n", authurl)
	fmt.Printf("paste code:")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("Failed to scan auth code: %v", err)
	}
	token, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("Failed to use auth code: %v", err)
	}

	return token, nil
}

func getGCloudTokenCached(tokenCacheFilePath string) (*oauth2.Token, error) {
	tokenJson, err := ioutil.ReadFile(tokenCacheFilePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read token cache file: %v", tokenCacheFilePath)
	}

	var token oauth2.Token
	if err = json.Unmarshal(tokenJson, &token); err != nil {
		return nil, fmt.Errorf("Failed to parse token cache file: %v", tokenCacheFilePath)
	}

	return &token, nil
}

func updateGCloudTokenCache(token *oauth2.Token, tokenCacheFilePath string) error {
	tjson, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("Serializing token failed: %v", err)
	}

	if err := ioutil.WriteFile(tokenCacheFilePath, tjson, 0600); err != nil {
		return fmt.Errorf("Writing token cache failed: %v", err)
	}

	return nil
}

type ClientSource func(ctx context.Context) *http.Client

func GetGCloudClientSource(credentialsFilePath string, tokenCacheFilePath string, tryWebUI bool) (ClientSource, error) {
	credentialsJson, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read google cloud client-secret file: %v", err)
	}

	conf, err := google.ConfigFromJSON(credentialsJson,
		storage.ScopeFullControl,
		datastore.ScopeDatastore,
		datastore.ScopeUserEmail,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid google cloud key json: %v", err)
	}

	revertToWebUI := func() (ClientSource, error) {
		if !tryWebUI {
			return nil, fmt.Errorf("OAuth2 token cache invalid.")
		}

		token, err := GetGCloudTokenViaWebUI(conf)
		if err != nil {
			return nil, fmt.Errorf("Failed to get valid gcloud token: %v", err)
		}
		if err := updateGCloudTokenCache(token, tokenCacheFilePath); err != nil {
			return nil, fmt.Errorf("Failed to update token cache: %v", err)
		}

		// FIXME: Token cache is not updated if token was refreshed by tokenRefresher from conf.Client
		return func(ctx context.Context) *http.Client { return conf.Client(ctx, token) }, nil
	}

	token, err := getGCloudTokenCached(tokenCacheFilePath)
	if err != nil {
		return revertToWebUI()
	}
	if !token.Valid() {
		// try refresh
		token, err = conf.TokenSource(oauth2.NoContext, token).Token()
		if err != nil {
			return revertToWebUI()
		}
		if updateGCloudTokenCache(token, tokenCacheFilePath); err != nil {
			return nil, fmt.Errorf("Failed to update token cache: %v", err)
		}
	}

	// FIXME: Token cache is not updated if token was refreshed by tokenRefresher from conf.Client
	return func(ctx context.Context) *http.Client { return conf.Client(ctx, token) }, nil
}

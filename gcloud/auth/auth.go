package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func GetGCloudTokenViaWebUI(ctx context.Context, conf *oauth2.Config) (*oauth2.Token, error) {
	authurl := conf.AuthCodeURL("otaru", oauth2.AccessTypeOffline)
	fmt.Printf("visit %v\n", authurl)
	fmt.Printf("paste code:")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("Failed to scan auth code: %v", err)
	}
	token, err := conf.Exchange(ctx, code)
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

func GetGCloudTokenSource(ctx context.Context, credentialsFilePath string, tokenCacheFilePath string, tryWebUI bool) (oauth2.TokenSource, error) {
	credentialsJson, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read google cloud client-secret file: %v", err)
	}

	conf, err := google.ConfigFromJSON(
		credentialsJson,
		storage.ScopeFullControl,
		datastore.ScopeDatastore,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid google cloud key json: %v", err)
	}

	revertToWebUI := func() (oauth2.TokenSource, error) {
		if !tryWebUI {
			return nil, fmt.Errorf("OAuth2 token cache invalid.")
		}

		token, err := GetGCloudTokenViaWebUI(ctx, conf)
		if err != nil {
			return nil, fmt.Errorf("Failed to get valid gcloud token: %v", err)
		}
		if err := updateGCloudTokenCache(token, tokenCacheFilePath); err != nil {
			return nil, fmt.Errorf("Failed to update token cache: %v", err)
		}

		// FIXME: Token cache is not updated if token was refreshed by tokenRefresher from conf.Client
		return conf.TokenSource(ctx, token), nil
	}

	token, err := getGCloudTokenCached(tokenCacheFilePath)
	if err != nil {
		return revertToWebUI()
	}
	if !token.Valid() {
		// try refresh
		token, err = conf.TokenSource(ctx, token).Token()
		if err != nil {
			return revertToWebUI()
		}
		if updateGCloudTokenCache(token, tokenCacheFilePath); err != nil {
			return nil, fmt.Errorf("Failed to update token cache: %v", err)
		}
	}

	// FIXME: Token cache is not updated if token was refreshed by tokenRefresher from conf.Client
	return conf.TokenSource(ctx, token), nil
}

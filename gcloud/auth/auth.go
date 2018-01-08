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

	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("gcloud-auth")

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

type updateCacheTokenSource struct {
	be            oauth2.TokenSource
	cacheFilePath string

	cachedToken *oauth2.Token
}

func (s *updateCacheTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.be.Token()
	if err != nil {
		return nil, err
	}

	if token != s.cachedToken {
		tjson, err := json.Marshal(token)
		if err != nil {
			logger.Panicf(mylog, "Marshalling token failed: %v", err)
			return nil, err
		}

		if err := ioutil.WriteFile(s.cacheFilePath, tjson, 0600); err != nil {
			logger.Warningf(mylog, "Marshalling token failed: %v", err)
			return token, nil
		}

		s.cachedToken = token
	}

	return token, err
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
		return nil, fmt.Errorf("invalid google cloud key json \"%v\" err: %v", credentialsFilePath, err)
	}

	var tsrc oauth2.TokenSource
	cachedToken, err := getGCloudTokenCached(tokenCacheFilePath)
	if err == nil {
		tsrc = &updateCacheTokenSource{
			be:            conf.TokenSource(ctx, cachedToken),
			cacheFilePath: tokenCacheFilePath,
			cachedToken:   cachedToken,
		}
		// Set |err| and fallback to webui if refresh failed.
		_, err = tsrc.Token()
		if err != nil {
			logger.Infof(mylog, "Cached token invalid or refresh failed. Falling back to webui auth: %v", err)
		}
	}
	if err != nil {
		if !tryWebUI {
			return nil, fmt.Errorf("OAuth2 token cache \"%v\" invalid, and WebUI session disabled.", tokenCacheFilePath)
		}

		token, err := GetGCloudTokenViaWebUI(ctx, conf)
		if err != nil {
			return nil, fmt.Errorf("Failed to get valid gcloud token via WebUI: %v", err)
		}
		if !token.Valid() {
			logger.Panicf(mylog, "gcloud auth token acquired via WebUI must be valid.")
		}

		tsrc = &updateCacheTokenSource{
			be:            conf.TokenSource(ctx, token),
			cacheFilePath: tokenCacheFilePath,
			cachedToken:   nil,
		}
	}

	if tsrc == nil {
		logger.Panicf(mylog, "tsrc must be non-nil by here.")
	}
	return tsrc, nil
}

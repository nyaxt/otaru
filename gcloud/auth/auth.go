package auth

import (
	"context"
	"fmt"
	"io/ioutil"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func GetGCloudTokenSource(credentialsFilePath string) (oauth2.TokenSource, error) {
	credentialsJson, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read google cloud client-secret file: %v", err)
	}

	conf, err := google.JWTConfigFromJSON(
		credentialsJson,
		storage.ScopeFullControl,
		datastore.ScopeDatastore,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid google cloud key json \"%v\" err: %v", credentialsFilePath, err)
	}

	return conf.TokenSource(context.Background()), nil
}

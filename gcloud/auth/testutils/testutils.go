package testutils

import (
	"log"

	"golang.org/x/oauth2"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/testutils"
)

var CredentialsFilePath string

func init() {
	CredentialsFilePath = facade.FindGCPServiceAccountJSON("")
	if CredentialsFilePath == "" {
		panic("Failed to find Google Cloud service account json file.")
	}
}

const TestBucketName = "otaru-dev-unittest"

func TestConfig() *facade.Config {
	return &facade.Config{
		ProjectName:         "otaru-dev",
		CredentialsFilePath: CredentialsFilePath,
	}
}

func TestTokenSource() oauth2.TokenSource {
	clisrc, err := auth.GetGCloudTokenSource(CredentialsFilePath)
	if err != nil {
		log.Fatalf("Failed to create TestTokenSource: %v", err)
	}
	return clisrc
}

func TestDSConfig(rootKeyStr string) *datastore.Config {
	projectName := TestConfig().ProjectName
	return datastore.NewConfig(projectName, rootKeyStr, testutils.TestCipher(), TestTokenSource())
}

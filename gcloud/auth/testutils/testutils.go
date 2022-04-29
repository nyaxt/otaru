package testutils

import (
	"log"
	"os"

	"golang.org/x/oauth2"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	tu "github.com/nyaxt/otaru/testutils"
)

var CredentialsFilePath = os.Getenv("OTARU_TEST_CREDENTIALS_FILE")

func init() {
	if CredentialsFilePath == "" {
		panic("OTARU_TEST_CREDENTIALS_FILE env missing")
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
	return datastore.NewConfig(projectName, rootKeyStr, tu.TestCipher(), TestTokenSource())
}

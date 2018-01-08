package testutils

import (
	"log"

	"golang.org/x/oauth2"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	tu "github.com/nyaxt/otaru/testutils"
)

var testConfigCached *facade.Config

func TestConfig() *facade.Config {
	if testConfigCached != nil {
		return testConfigCached
	}

	cfg, err := facade.NewConfig(facade.DefaultConfigDir())
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	testConfigCached = cfg
	return testConfigCached
}

func TestTokenSource() oauth2.TokenSource {
	cfg := TestConfig()

	clisrc, err := auth.GetGCloudTokenSource(cfg.CredentialsFilePath)
	if err != nil {
		log.Fatalf("Failed to create TestTokenSource: %v", err)
	}
	return clisrc
}

func TestBucketName() string {
	name := TestConfig().TestBucketName
	if len(name) == 0 {
		log.Fatalf("Please specify \"test_bucket_name\" in config.toml to run gcloud tests.")
	}
	return name
}

func TestDSConfig(rootKeyStr string) *datastore.Config {
	projectName := TestConfig().ProjectName
	return datastore.NewConfig(projectName, rootKeyStr, tu.TestCipher(), TestTokenSource())
}

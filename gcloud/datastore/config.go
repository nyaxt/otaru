package datastore

import (
	"github.com/nyaxt/otaru/btncrypt"

	"context"

	"cloud.google.com/go/datastore"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

var ctxNoNamespace = context.Background()

type Config struct {
	projectName string
	rootKeyStr  string
	c           *btncrypt.Cipher
	tsrc        oauth2.TokenSource
}

func NewConfig(projectName, rootKeyStr string, c *btncrypt.Cipher, tsrc oauth2.TokenSource) *Config {
	if len(projectName) == 0 {
		panic("empty projectName")
	}
	if len(rootKeyStr) == 0 {
		panic("empty rootKeyStr")
	}
	if tsrc == nil {
		panic("nil tokensource")
	}

	return &Config{
		projectName: projectName,
		rootKeyStr:  rootKeyStr,
		c:           c,
		tsrc:        tsrc,
	}
}

func (cfg *Config) getClient(ctx context.Context) (*datastore.Client, error) {
	return datastore.NewClient(ctx, cfg.projectName, option.WithTokenSource(cfg.tsrc))
}

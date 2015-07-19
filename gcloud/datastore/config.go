package datastore

import (
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/gcloud/auth"

	"golang.org/x/net/context"
	"google.golang.org/cloud"
)

type Config struct {
	projectName string
	rootKeyStr  string
	c           btncrypt.Cipher
	clisrc      auth.ClientSource
}

func NewConfig(projectName, rootKeyStr string, c btncrypt.Cipher, clisrc auth.ClientSource) *Config {
	if len(projectName) == 0 {
		panic("empty projectName")
	}
	if len(rootKeyStr) == 0 {
		panic("empty rootKeyStr")
	}
	if clisrc == nil {
		panic("nil clisrc")
	}

	return &Config{
		projectName: projectName,
		rootKeyStr:  rootKeyStr,
		c:           c,
		clisrc:      clisrc,
	}
}

func (cfg *Config) getContext() context.Context {
	return cloud.NewContext(cfg.projectName, cfg.clisrc(context.TODO()))
}

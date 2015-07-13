package facade

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/naoina/toml"

	"github.com/nyaxt/otaru/util"
)

type Config struct {
	PasswordFile                 string
	ProjectName                  string
	BucketName                   string
	UseSeparateBucketForMetadata bool
	BlobCacheDir                 string
	MetadataCacheDir             string
	LocalDebug                   bool

	Password string
}

func NewConfigFromTomlFile(configpath string) (*Config, error) {
	buf, err := ioutil.ReadFile(configpath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %v", err)
	}

	cfg := &Config{
		PasswordFile:                 path.Join(os.Getenv("HOME"), ".otaru", "password.txt"),
		UseSeparateBucketForMetadata: false,
		BlobCacheDir:                 "/var/cache/otaru/blob",
		MetadataCacheDir:             "/var/cache/otaru/metadata",
	}
	if err := toml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}

	if cfg.Password != "" {
		log.Printf("Storing password directly on config file is not recommended.")
	} else {
		fi, err := os.Stat(cfg.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to stat password file \"%s\": %v", cfg.PasswordFile, err)
		}
		if fi.Mode()&os.ModePerm != 0400 {
			log.Printf("Password file \"%s\" permission is not 0400", cfg.PasswordFile)
		}

		cfg.Password, err = util.StringFromFile(cfg.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to read password file \"%s\": %v", cfg.PasswordFile, err)
		}
	}

	if cfg.ProjectName == "" {
		return nil, fmt.Errorf("Config Error: ProjectName must be given.")
	}
	if cfg.BucketName == "" {
		return nil, fmt.Errorf("Config Error: BucketName must be given.")
	}

	os.MkdirAll(cfg.BlobCacheDir, 0700)
	os.MkdirAll(cfg.MetadataCacheDir, 0700)

	return cfg, nil
}

type OneshotConfig struct {
	Mkfs bool
}

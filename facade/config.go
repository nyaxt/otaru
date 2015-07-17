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
	TestBucketName               string
	ProjectName                  string
	BucketName                   string
	UseSeparateBucketForMetadata bool
	CacheDir                     string
	LocalDebug                   bool

	Password string

	CredentialsFilePath string
	TokenCacheFilePath  string
}

func DefaultConfigDir() string {
	if otarudir := os.Getenv("OTARUDIR"); len(otarudir) > 0 {
		return otarudir
	}

	return path.Join(os.Getenv("HOME"), ".otaru")
}

func NewConfig(configdir string) (*Config, error) {
	tomlpath := path.Join(configdir, "config.toml")

	buf, err := ioutil.ReadFile(tomlpath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %v", err)
	}

	cfg := &Config{
		PasswordFile:                 path.Join(configdir, "password.txt"),
		UseSeparateBucketForMetadata: false,
		CacheDir:                     "/var/cache/otaru",
		CredentialsFilePath:          path.Join(configdir, "credentials.json"),
		TokenCacheFilePath:           path.Join(configdir, "tokencache.json"),
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
			log.Printf("Warning: Password file \"%s\" permission is not 0400", cfg.PasswordFile)
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

	if _, err := os.Stat(cfg.CredentialsFilePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Credentials not found at %s", cfg.CredentialsFilePath)
		} else {
			return nil, fmt.Errorf("Failed to stat credentials file \"%s\" from unknown err: %v", cfg.CredentialsFilePath, err)
		}
	}

	if _, err := os.Stat(cfg.TokenCacheFilePath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Warning: Token cache file found not at %s", cfg.TokenCacheFilePath)
		} else {
			return nil, fmt.Errorf("Failed to stat token cache file \"%s\" from unknown err: %v", cfg.TokenCacheFilePath, err)
		}
	}

	return cfg, nil
}

type OneshotConfig struct {
	Mkfs bool
}

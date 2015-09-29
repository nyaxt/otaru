package facade

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"

	"github.com/dustin/go-humanize"
	gfluent "github.com/fluent/fluent-logger-golang/fluent"
	"github.com/naoina/toml"

	"github.com/nyaxt/otaru/logger"
	loggerconfig "github.com/nyaxt/otaru/logger/config"
	"github.com/nyaxt/otaru/util"
)

type Config struct {
	PasswordFile                 string
	TestBucketName               string
	ProjectName                  string
	BucketName                   string
	UseSeparateBucketForMetadata bool

	CacheDir string
	// Cache size high watermark: discard cache when cache dir usage reach here.
	CacheHighWatermarkInBytes int64
	CacheHighWatermark        string
	// Cache size low watermark: when discarding cache, try to reduce cache dir usage under here.
	CacheLowWatermarkInBytes int64
	CacheLowWatermark        string

	LocalDebug bool

	Password string

	CredentialsFilePath string
	TokenCacheFilePath  string

	// Run GC every "GCPeriod" seconds.
	GCPeriod int64

	// Install /api/debug handlers.
	InstallDebugApi bool

	Fluent gfluent.Config

	Logger loggerconfig.Config
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
		CacheHighWatermarkInBytes:    math.MaxInt64,
		CacheLowWatermarkInBytes:     math.MaxInt64,
		CredentialsFilePath:          path.Join(configdir, "credentials.json"),
		TokenCacheFilePath:           path.Join(configdir, "tokencache.json"),
		GCPeriod:                     15 * 60,
		InstallDebugApi:              false,
	}

	if err := toml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}

	if cfg.CacheHighWatermark != "" {
		bytes, err := humanize.ParseBytes(cfg.CacheHighWatermark)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse cache_high_watermark \"%s\"", cfg.CacheHighWatermark)
		}
		cfg.CacheHighWatermarkInBytes = int64(bytes)
	}
	if cfg.CacheLowWatermark != "" {
		bytes, err := humanize.ParseBytes(cfg.CacheLowWatermark)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse cache_low_watermark \"%s\"", cfg.CacheLowWatermark)
		}
		cfg.CacheLowWatermarkInBytes = int64(bytes)
	}
	if cfg.CacheLowWatermarkInBytes > cfg.CacheHighWatermarkInBytes {
		return nil, fmt.Errorf("cache_low_watermark %s higher than cache_high_watermark %s",
			humanize.Bytes(uint64(cfg.CacheLowWatermarkInBytes)), humanize.Bytes(uint64(cfg.CacheHighWatermarkInBytes)))
	}

	if cfg.Password != "" {
		logger.Warningf(mylog, "Storing password directly on config file is not recommended.")
	} else {
		fi, err := os.Stat(cfg.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to stat password file \"%s\": %v", cfg.PasswordFile, err)
		}
		if fi.Mode()&os.ModePerm != 0400 {
			logger.Warningf(mylog, "Warning: Password file \"%s\" permission is not 0400", cfg.PasswordFile)
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

	if !cfg.LocalDebug {
		if _, err := os.Stat(cfg.CredentialsFilePath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("Credentials not found at %s", cfg.CredentialsFilePath)
			} else {
				return nil, fmt.Errorf("Failed to stat credentials file \"%s\" from unknown err: %v", cfg.CredentialsFilePath, err)
			}
		}

		if _, err := os.Stat(cfg.TokenCacheFilePath); err != nil {
			if os.IsNotExist(err) {
				logger.Warningf(mylog, "Warning: Token cache file found not at %s", cfg.TokenCacheFilePath)
			} else {
				return nil, fmt.Errorf("Failed to stat token cache file \"%s\" from unknown err: %v", cfg.TokenCacheFilePath, err)
			}
		}
	}

	if cfg.Fluent.TagPrefix == "" {
		cfg.Fluent.TagPrefix = "otaru"
	}
	cfg.Fluent.MaxRetry = math.MaxInt32

	if err := loggerconfig.Apply(mylog, cfg.Logger); err != nil {
		return nil, err
	}

	return cfg, nil
}

type OneshotConfig struct {
	Mkfs bool
}

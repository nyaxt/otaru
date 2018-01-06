package facade

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"

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

	ReadOnly   bool
	LocalDebug bool

	Password string

	CredentialsFilePath string
	TokenCacheFilePath  string

	// If non-empty, perform fuse mount.
	FuseMountPoint string

	// If non-empty, serve content over webdav.
	WebdavAddr string

	// Run GC every "GCPeriod" seconds.
	GCPeriod int64

	Fluent       gfluent.Config
	Logger       loggerconfig.Config
	ApiServer    ApiServerConfig
	WebdavServer WebdavServerConfig
}

type ApiServerConfig struct {
	// API listen addr. Defaults to ":10246".
	ListenAddr string

	// Install debug handlers if enabled.
	EnableDebug bool

	WebUIRootPath      string `toml:"webui_root_path"`
	CertFile           string
	KeyFile            string
	CORSAllowedOrigins []string `toml:"cors_allowed_origins"`
}

type WebdavServerConfig struct {
	ListenAddr string

	// Serve WebDav over TLS
	EnableTLS bool `toml:"enable_tls"`

	CertFile string
	KeyFile  string

	// Require digest auth if specified
	HtdigestFilePath string

	DigestAuthRealm string
}

func DefaultConfigDir() string {
	if otarudir := os.Getenv("OTARUDIR"); len(otarudir) > 0 {
		return otarudir
	}

	return path.Join(os.Getenv("HOME"), ".otaru")
}

func NewConfig(configdir string) (*Config, error) {
	if err := util.IsDir(configdir); err != nil {
		return nil, fmt.Errorf("configdir \"%s\" is not a dir: %v", configdir, err)
	}
	os.Setenv("OTARUDIR", configdir)

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
		ApiServer: ApiServerConfig{
			ListenAddr:  ":10246",
			EnableDebug: false,
			CertFile:    path.Join(configdir, "cert.pem"),
			KeyFile:     path.Join(configdir, "cert-key.pem"),
		},
	}

	if err := toml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}

	cfg.CacheDir = os.ExpandEnv(cfg.CacheDir)
	cfg.CacheDir, err = filepath.Abs(cfg.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve cache dir to absolute path \"%s\": %v", cfg.CacheDir, err)
	}
	if err := util.IsDir(cfg.CacheDir); err != nil {
		return nil, fmt.Errorf("Failed to resolve cache dir \"%s\": %v", cfg.CacheDir, err)
	} else {
		logger.Infof(mylog, "Cache dir resolved to: %s", cfg.CacheDir)
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

	if err := verifyWebdavServerConfig(configdir, &cfg.WebdavServer); err != nil {
		return nil, err
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

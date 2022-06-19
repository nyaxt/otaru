package facade

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/naoina/toml"
	"go.uber.org/zap"

	"github.com/nyaxt/otaru/util"
	"github.com/nyaxt/otaru/util/readpem"
)

type Config struct {
	PasswordFile                 string
	ProjectName                  string
	BucketName                   string
	UseSeparateBucketForMetadata bool
	CredentialsFilePath          string

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

	// If non-empty, perform fuse mount.
	FuseMountPoint string

	// Run GC every "GCPeriod" seconds.
	GCPeriod int64 `toml:"gc_period"`

	Logger    *zap.Logger
	ApiServer ApiServerConfig
}

type ApiServerConfig struct {
	// API listen addr. Defaults to ":10246".
	ListenAddr string

	// Install debug handlers if enabled.
	EnableDebug bool

	Certs            []*x509.Certificate
	CertsFile        string
	Key              crypto.PrivateKey
	KeyFile          string
	ClientCACert     *x509.Certificate
	ClientCACertFile string `toml:"client_ca_cert_file"`

	WebUIRootPath      string   `toml:"webui_root_path"`
	CORSAllowedOrigins []string `toml:"cors_allowed_origins"`
}

func DefaultConfigDir() string {
	return path.Join(os.Getenv("HOME"), ".otaru")
}

func FindGCPServiceAccountJSON(configdir string) string {
	candidates := []string{
		path.Join(configdir, "credentials.json"),
		os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		fi, err := os.Stat(c)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			continue
		}

		return c
	}
	return ""
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
		GCPeriod:                     15 * 60,
		ApiServer: ApiServerConfig{
			ListenAddr:       ":10246",
			EnableDebug:      false,
			CertsFile:        path.Join(configdir, "cert.pem"),
			KeyFile:          path.Join(configdir, "cert-key.pem"),
			ClientCACertFile: path.Join(configdir, "clientauth-ca.pem"),
		},
	}

	if err := toml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.L()
	}

	s := cfg.Logger.Named("NewConfig").Sugar()

	if cfg.CredentialsFilePath == "" {
		cfg.CredentialsFilePath = FindGCPServiceAccountJSON(configdir)
		if cfg.CredentialsFilePath == "" {
			return nil, fmt.Errorf("Failed to find Google Cloud service account json.")
		}
	}

	cfg.CacheDir = os.ExpandEnv(cfg.CacheDir)
	cfg.CacheDir, err = filepath.Abs(cfg.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve cache dir to absolute path \"%s\": %v", cfg.CacheDir, err)
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
		s.Warnf("Storing password directly on config file is not recommended.")
	} else {
		fi, err := os.Stat(cfg.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to stat password file \"%s\": %v", cfg.PasswordFile, err)
		}
		if fi.Mode()&os.ModePerm != 0400 {
			s.Warnf("Warning: Password file \"%s\" permission is not 0400", cfg.PasswordFile)
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
		cfg.CredentialsFilePath = os.ExpandEnv(cfg.CredentialsFilePath)
		if _, err := os.Stat(cfg.CredentialsFilePath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("Credentials not found at %s", cfg.CredentialsFilePath)
			} else {
				return nil, fmt.Errorf("Failed to stat credentials file \"%s\" from unknown err: %v", cfg.CredentialsFilePath, err)
			}
		}
	}

	if err := readpem.ReadCertificatesFile(
		"ApiServer.Cert",
		cfg.ApiServer.CertsFile,
		&cfg.ApiServer.Certs,
	); err != nil {
		return nil, err
	}
	if err := readpem.ReadKeyFile(
		"ApiServer.Key",
		cfg.ApiServer.KeyFile,
		&cfg.ApiServer.Key,
	); err != nil {
		return nil, err
	}
	if err := readpem.ReadCertificateFile(
		"ApiServer.ClientCACert",
		cfg.ApiServer.ClientCACertFile,
		&cfg.ApiServer.ClientCACert,
	); err != nil {
		return nil, err
	}

	return cfg, nil
}

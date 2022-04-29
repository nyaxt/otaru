package cli

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/naoina/toml"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/util"
)

var ErrNoLocalPathDefined = errors.New("No local root path is defined.")

type CliConfig struct {
	Host map[string]*Host

	Fe     FeConfig
	Webdav WebdavConfig

	LocalRootPath string
	TrashDirPath  string
}

type Host struct {
	ApiEndpoint        string
	ExpectedCertFile   string
	OverrideServerName string

	AuthToken     string
	AuthTokenFile string

	CACert     *x509.Certificate
	CACertFile string
}

type FeConfig struct {
	ListenAddr    string
	WebUIRootPath string `toml:"webui_root_path"`

	CertFile string
	KeyFile  string

	JwtPubkeyFile string
}

type WebdavConfig struct {
	ListenAddr string

	CertFile string
	KeyFile  string

	BasicAuthUser     string
	BasicAuthPassword string
}

func NewConfig(configdir string) (*CliConfig, error) {
	if err := util.IsDir(configdir); err != nil {
		return nil, fmt.Errorf("configdir %q is not a dir: %v", configdir, err)
	}
	os.Setenv("OTARUDIR", configdir)

	tomlpath := path.Join(configdir, "cliconfig.toml")

	buf, err := ioutil.ReadFile(tomlpath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %v", err)
	}

	possibleCertFile := path.Join(configdir, "cert.pem")
	if util.IsRegular(possibleCertFile) != nil {
		possibleCertFile = ""
	}
	cfg := CliConfig{
		Fe: FeConfig{
			ListenAddr: ":10247",
			CertFile:   possibleCertFile,
			KeyFile:    path.Join(configdir, "cert-key.pem"),
		},
		Webdav: WebdavConfig{
			CertFile:      possibleCertFile,
			KeyFile:       path.Join(configdir, "cert-key.pem"),
			BasicAuthUser: "readonly",
		},
	}
	if err := toml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}
	for vhost, h := range cfg.Host {
		h.ApiEndpoint = os.ExpandEnv(h.ApiEndpoint)
		h.ExpectedCertFile = os.ExpandEnv(h.ExpectedCertFile)
		h.OverrideServerName = os.ExpandEnv(h.OverrideServerName)
		h.AuthToken = os.ExpandEnv(h.AuthToken)
		h.AuthTokenFile = os.ExpandEnv(h.AuthTokenFile)

		if h.AuthToken == "" {
			if h.AuthTokenFile != "" {
				bs, err := ioutil.ReadFile(h.AuthTokenFile)
				if err != nil {
					return nil, fmt.Errorf("Failed to read token file %q: %v", h.AuthTokenFile, vhost)
				}
				h.AuthToken = strings.TrimSpace(string(bs))
			}
		} else {
			if h.AuthTokenFile != "" {
				return nil, fmt.Errorf("AuthToken and AuthTokenFile are both specified for host %q.", vhost)
			}
		}
	}
	cfg.Fe.CertFile = os.ExpandEnv(cfg.Fe.CertFile)
	cfg.Fe.KeyFile = os.ExpandEnv(cfg.Fe.KeyFile)
	cfg.Fe.JwtPubkeyFile = os.ExpandEnv(cfg.Fe.JwtPubkeyFile)
	cfg.Webdav.CertFile = os.ExpandEnv(cfg.Webdav.CertFile)
	cfg.Webdav.KeyFile = os.ExpandEnv(cfg.Webdav.KeyFile)

	if cfg.LocalRootPath != "" {
		if err := util.IsDir(cfg.LocalRootPath); err != nil {
			return nil, fmt.Errorf("local_root_path %q is not a dir: %v", cfg.LocalRootPath, err)
		}
	}
	if cfg.TrashDirPath != "" {
		if err := util.IsDir(cfg.TrashDirPath); err != nil {
			return nil, fmt.Errorf("trash_dir_path %q is not a dir: %v", cfg.TrashDirPath, err)
		}
	}

	return &cfg, nil
}

func (cfg *CliConfig) ResolveLocalPath(relpath string) (string, error) {
	if cfg.LocalRootPath == "" {
		return "", ErrNoLocalPathDefined
	}
	return opath.ResolveLocalPath(cfg.LocalRootPath, relpath), nil
}

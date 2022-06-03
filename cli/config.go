package cli

import (
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/naoina/toml"

	opath "github.com/nyaxt/otaru/cli/path"
	"github.com/nyaxt/otaru/util"
	"github.com/nyaxt/otaru/util/readpem"
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
	ApiEndpoint string

	// FIXME: Remove ExpectedCertFile
	ExpectedCertFile   string
	OverrideServerName string

	CACert     *x509.Certificate
	CACertFile string `toml:"ca_cert_file"`
	Cert       *x509.Certificate
	CertFile   string
	Key        crypto.PrivateKey
	KeyFile    string
}

type FeConfig struct {
	ListenAddr    string
	WebUIRootPath string `toml:"webui_root_path"`

	Certs     []*x509.Certificate
	CertsFile string
	Key       crypto.PrivateKey
	KeyFile   string

	BasicAuthUser     string
	BasicAuthPassword string
}

type WebdavConfig struct {
	ListenAddr string

	Certs     []*x509.Certificate
	CertsFile string
	Key       crypto.PrivateKey
	KeyFile   string

	BasicAuthUser     string
	BasicAuthPassword string
}

func NewConfig(configdir string) (*CliConfig, error) {
	if err := util.IsDir(configdir); err != nil {
		return nil, fmt.Errorf("configdir %q is not a dir: %v", configdir, err)
	}
	os.Setenv("OTARUDIR", configdir)

	tomlpath := path.Join(configdir, "cliconfig.toml")

	bs, err := ioutil.ReadFile(tomlpath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %v", err)
	}

	possibleCertsFile := path.Join(configdir, "cert.pem")
	if util.IsRegular(possibleCertsFile) != nil {
		possibleCertsFile = ""
	}
	cfg := CliConfig{
		Fe: FeConfig{
			ListenAddr: ":10247",
			CertsFile:  possibleCertsFile,
			KeyFile:    path.Join(configdir, "cert-key.pem"),
		},
		Webdav: WebdavConfig{
			CertsFile:     possibleCertsFile,
			KeyFile:       path.Join(configdir, "cert-key.pem"),
			BasicAuthUser: "readonly",
		},
	}
	if err := toml.Unmarshal(bs, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}
	for vhost, h := range cfg.Host {
		h.ApiEndpoint = os.ExpandEnv(h.ApiEndpoint)
		h.ExpectedCertFile = os.ExpandEnv(h.ExpectedCertFile)
		h.OverrideServerName = os.ExpandEnv(h.OverrideServerName)
		if err := readpem.ReadCertificateFile("CACert", h.CACertFile, &h.CACert); err != nil {
			return nil, fmt.Errorf("vhost %q: %w", vhost, err)
		}
		if err := readpem.ReadCertificateFile("Cert", h.CertFile, &h.Cert); err != nil {
			return nil, fmt.Errorf("vhost %q: %w", vhost, err)
		}
		if err := readpem.ReadKeyFile("Key", h.KeyFile, &h.Key); err != nil {
			return nil, fmt.Errorf("vhost %q: %w", vhost, err)
		}
	}
	if err := readpem.ReadCertificatesFile("Fe.Certs", cfg.Fe.CertsFile, &cfg.Fe.Certs); err != nil {
		return nil, err
	}
	if err := readpem.ReadKeyFile("Fe.Key", cfg.Fe.KeyFile, &cfg.Fe.Key); err != nil {
		return nil, err
	}
	if err := readpem.ReadCertificatesFile("Webdav.Cert", cfg.Webdav.CertsFile, &cfg.Webdav.Certs); err != nil {
		return nil, err
	}
	if err := readpem.ReadKeyFile("Webdav.Key", cfg.Webdav.KeyFile, &cfg.Webdav.Key); err != nil {
		return nil, err
	}

	cfg.LocalRootPath = os.ExpandEnv(cfg.LocalRootPath)
	if cfg.LocalRootPath != "" {
		if err := util.IsDir(cfg.LocalRootPath); err != nil {
			return nil, fmt.Errorf("local_root_path %q is not a dir: %v", cfg.LocalRootPath, err)
		}
	}
	cfg.TrashDirPath = os.ExpandEnv(cfg.TrashDirPath)
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

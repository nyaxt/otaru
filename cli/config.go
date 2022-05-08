package cli

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

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
	ApiEndpoint string

	// FIXME: Remove ExpectedCertFile
	ExpectedCertFile   string
	OverrideServerName string

	CACert     *x509.Certificate
	CACertFile string
	Cert       *x509.Certificate
	CertFile   string
	Key        crypto.PrivateKey
	KeyFile    string
}

type FeConfig struct {
	ListenAddr    string
	WebUIRootPath string `toml:"webui_root_path"`

	Cert     *x509.Certificate
	CertFile string
	Key      crypto.PrivateKey
	KeyFile  string

	BasicAuthUser     string
	BasicAuthPassword string
}

type WebdavConfig struct {
	ListenAddr string

	Cert     *x509.Certificate
	CertFile string
	Key      crypto.PrivateKey
	KeyFile  string

	BasicAuthUser     string
	BasicAuthPassword string
}

func readCertificateFile(configkey, path string, pcert **x509.Certificate) error {
	if *pcert != nil {
		if path != "" {
			return fmt.Errorf("%[1]s and %[1]sFile are both specified", configkey)
		}
		return nil
	}

	if path == "" {
		return nil
	}
	path = os.ExpandEnv(path)

	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read %sFile %q: %w", configkey, path, err)
	}

	block, _ := pem.Decode(bs)
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to parse %sFile %q: %w", configkey, path, err)
	}

	*pcert = c
	return nil
}

func readKeyFile(configkey, path string, pkey *crypto.PrivateKey) error {
	if *pkey != nil {
		if path != "" {
			return fmt.Errorf("%[1]s and %[1]sFile are both specified", configkey)
		}
		return nil
	}

	if path == "" {
		return nil
	}
	path = os.ExpandEnv(path)

	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read %sFile %q: %w", configkey, path, err)
	}

	block, _ := pem.Decode(bs)

	var k crypto.PrivateKey
	if eck, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		k = eck
	} else if p1k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		k = p1k
	} else {
		return fmt.Errorf("Failed to parse %sFile %q as EC nor PKCS1 private key", configkey, path)
	}

	*pkey = k
	return nil

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
	if err := toml.Unmarshal(bs, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}
	for vhost, h := range cfg.Host {
		h.ApiEndpoint = os.ExpandEnv(h.ApiEndpoint)
		h.ExpectedCertFile = os.ExpandEnv(h.ExpectedCertFile)
		h.OverrideServerName = os.ExpandEnv(h.OverrideServerName)
		if err := readCertificateFile("CACert", h.CACertFile, &h.CACert); err != nil {
			return nil, fmt.Errorf("vhost %q: %w", vhost, err)
		}
		if err := readCertificateFile("Cert", h.CertFile, &h.Cert); err != nil {
			return nil, fmt.Errorf("vhost %q: %w", vhost, err)
		}
		if err := readKeyFile("Key", h.KeyFile, &h.Key); err != nil {
			return nil, fmt.Errorf("vhost %q: %w", vhost, err)
		}
	}
	if err := readCertificateFile("Fe.Cert", cfg.Fe.CertFile, &cfg.Fe.Cert); err != nil {
		return nil, err
	}
	if err := readKeyFile("Fe.Key", cfg.Fe.KeyFile, &cfg.Fe.Key); err != nil {
		return nil, err
	}
	if err := readCertificateFile("Webdav.Cert", cfg.Webdav.CertFile, &cfg.Webdav.Cert); err != nil {
		return nil, err
	}
	if err := readKeyFile("Webdav.Key", cfg.Webdav.KeyFile, &cfg.Webdav.Key); err != nil {
		return nil, err
	}

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

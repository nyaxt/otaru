package facade

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/webdav"
)

func verifyWebdavServerConfig(cfg *WebdavServerConfig) error {
	if cfg.EnableTLS {
		if cfg.CertFile == "" || cfg.KeyFile == "" {
			return errors.New("Webdav TLS is enabled, but {cert,key} file not specified.")
		}
	} else {
		if cfg.CertFile != "" || cfg.KeyFile != "" {
			logger.Warningf(mylog, "Webdav {cert,key} file specified, but TLS is not enabled.")
		}
	}
	if _, err := ioutil.ReadFile(cfg.HtdigestFilePath); err != nil {
		return fmt.Errorf("Failed to read htdigest file: %v", err)
	}

	return nil
}

func (o *Otaru) buildWebdavServerOptions(ofs *filesystem.FileSystem, cfg *WebdavServerConfig) []webdav.Option {
	opts := []webdav.Option{
		webdav.FileSystem(ofs),
		webdav.ListenAddr(cfg.ListenAddr),
	}

	if cfg.EnableTLS {
		opts = append(opts, webdav.X509KeyPair(cfg.CertFile, cfg.KeyFile))
	}
	if cfg.HtdigestFilePath != "" {
		realm := cfg.DigestAuthRealm
		if realm == "" {
			realm = "otaru webdav"
		}
		opts = append(opts, webdav.DigestAuth(realm, cfg.HtdigestFilePath))
	}
	return opts
}

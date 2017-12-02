package facade

import (
	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/logger"
)

func (o *Otaru) buildApiServerOptions(cfg *Config) []apiserver.Option {
	options := []apiserver.Option{
		apiserver.ListenAddr(cfg.HttpApiAddr),
		apiserver.InstallSystemService(),
		apiserver.InstallBlobstoreService(o.S, o.DefaultBS, o.CBS),
		apiserver.InstallFileSystemService(o.FS),
		apiserver.InstallFileHandler(o.FS),
	}
	if cfg.WebUIRootPath != "" {
		logger.Infof(mylog, "Overriding embedded WebUI and serving WebUI at %s", cfg.WebUIRootPath)
		options = append(options, apiserver.OverrideWebUI(cfg.WebUIRootPath))
	}
	/*
		mlogger.Install(o.MGMT)
		if cfg.InstallDebugApi {
			logger.Infof(mylog, "Installing Debug APIs.")
			mdebug.Install(o.MGMT)
		}
		msystem.Install(o.MGMT)
		if gcsbs, ok := o.DefaultBS.(*gcs.GCSBlobStore); ok {
			mgcsblobstore.Install(o.MGMT, gcsbs)
		}
		minodedb.Install(o.MGMT, o.IDBS)
		mscheduler.Install(o.MGMT, o.S, o.R)
		mgc.Install(o.MGMT, o.S, o)
	*/

	return options
}

package facade

import (
	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/otaruapiserver"
)

func (o *Otaru) buildApiServerOptions(cfg *ApiServerConfig) []apiserver.Option {
	options := []apiserver.Option{
		apiserver.ListenAddr(cfg.ListenAddr),
		apiserver.X509KeyPair(cfg.CertFile, cfg.KeyFile),
		apiserver.CORSAllowedOrigins(cfg.CORSAllowedOrigins),
		otaruapiserver.InstallBlobstoreService(o.S, o.DefaultBS, o.CBS),
		otaruapiserver.InstallFileHandler(o.FS),
		otaruapiserver.InstallFileSystemService(o.FS),
		otaruapiserver.InstallINodeDBService(o.IDBS),
		otaruapiserver.InstallLoggerService(),
		otaruapiserver.InstallSystemService(),
	}
	if cfg.WebUIRootPath != "" {
		logger.Infof(mylog, "Overriding embedded WebUI and serving WebUI at %s", cfg.WebUIRootPath)
		options = append(options, apiserver.OverrideWebUI(cfg.WebUIRootPath))
	}
	/*
		if cfg.InstallDebugApi {
			logger.Infof(mylog, "Installing Debug APIs.")
			mdebug.Install(o.MGMT)
		}
		msystem.Install(o.MGMT)
		if gcsbs, ok := o.DefaultBS.(*gcs.GCSBlobStore); ok {
			mgcsblobstore.Install(o.MGMT, gcsbs)
		}
		mscheduler.Install(o.MGMT, o.S, o.R)
		mgc.Install(o.MGMT, o.S, o)
	*/

	return options
}

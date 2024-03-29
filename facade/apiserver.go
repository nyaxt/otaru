package facade

import (
	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/assets/webui"
	"github.com/nyaxt/otaru/otaruapiserver"
	"github.com/nyaxt/otaru/pb/json"
	"go.uber.org/zap"
)

func (o *Otaru) buildApiServerOptions(cfg *ApiServerConfig) ([]apiserver.Option, error) {
	override := cfg.WebUIRootPath
	if override != "" {
		zap.S().Infof("Overriding embedded WebUI and serving WebUI at %s", override)
	}

	options := []apiserver.Option{
		apiserver.ListenAddr(cfg.ListenAddr),
		apiserver.TLSCertKey(cfg.Certs, cfg.Key),
		apiserver.ClientCACert(cfg.ClientCACert),
		apiserver.CORSAllowedOrigins(cfg.CORSAllowedOrigins),
		apiserver.SetDefaultHandler(webui.WebUIHandler(override, "/index.otaru-server.html")),
		apiserver.SetSwaggerJson(json.Assets, "/otaru.swagger.json"),
		otaruapiserver.InstallBlobstoreService(o.S, o.DefaultBS, o.CBS),
		otaruapiserver.InstallFileHandler(o.FS),
		otaruapiserver.InstallFileSystemService(o.FS),
		otaruapiserver.InstallINodeDBService(o.IDBS),
		otaruapiserver.InstallSystemService(),
	}

	return options, nil
}

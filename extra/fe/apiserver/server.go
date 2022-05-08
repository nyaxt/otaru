package apiserver

import (
	"net/http"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/assets/webui"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/extra/fe/pb/json"
	"github.com/nyaxt/otaru/extra/fe/preview"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("fe-apiserver")
var accesslog = logger.Registry().Category("http-fe")

func BuildApiServerOptions(cfg *cli.CliConfig) ([]apiserver.Option, error) {
	var webuifs http.FileSystem
	override := cfg.Fe.WebUIRootPath
	if override == "" {
		webuifs = webui.Assets
	} else {
		logger.Infof(mylog, "Overriding embedded WebUI and serving WebUI at %s", override)
		webuifs = http.Dir(override)
	}

	opts := []apiserver.Option{
		apiserver.ListenAddr(cfg.Fe.ListenAddr),
		apiserver.TLSCertKey(cfg.Fe.Cert, cfg.Fe.Key),
		apiserver.SetSwaggerJson(json.Assets, "/otaru-fe.swagger.json"),
		apiserver.SetWebUI(webuifs, "/index.otaru-fe.html"),
		preview.Install(cfg),
		InstallFeService(cfg),
		InstallProxyHandler(cfg, cfg.Fe.BasicAuthUser, cfg.Fe.BasicAuthPassword),
	}

	return opts, nil
}

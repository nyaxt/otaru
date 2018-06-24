package apiserver

import (
	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/extra/fe/pb/json"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("fe-apiserver")
var accesslog = logger.Registry().Category("http-fe")

func BuildApiServerOptions(cfg *cli.CliConfig) []apiserver.Option {
	return []apiserver.Option{
		apiserver.ListenAddr(cfg.Fe.ListenAddr),
		apiserver.X509KeyPair(cfg.Fe.CertFile, cfg.Fe.KeyFile),
		apiserver.SetSwaggerJson(json.Assets, "/otaru-fe.swagger.json"),
		InstallFeService(cfg),
		InstallProxyHandler(cfg),
	}
}

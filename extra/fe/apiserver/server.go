package apiserver

import (
	"errors"
	"net/http"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/apiserver/jwt"
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

	if cfg.Fe.JwtPubkeyFile == "" {
		return nil, errors.New("JwtPubkeyFile must be specified to start otaru-fe server.")
	}

	jwtauth, err := jwt.NewJWTAuthProviderFromFile(cfg.Fe.JwtPubkeyFile)
	if err != nil {
		return nil, err
	}

	opts := []apiserver.Option{
		apiserver.ListenAddr(cfg.Fe.ListenAddr),
		apiserver.X509KeyPairPath(cfg.Fe.CertFile, cfg.Fe.KeyFile),
		apiserver.SetSwaggerJson(json.Assets, "/otaru-fe.swagger.json"),
		apiserver.SetWebUI(webuifs, "/index.otaru-fe.html"),
		apiserver.JWTAuthProvider(jwtauth),
		preview.Install(cfg),
		InstallFeService(cfg),
		InstallProxyHandler(cfg, jwtauth),
	}

	return opts, nil
}

package facade

import (
	"net/http"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/apiserver/jwt"
	"github.com/nyaxt/otaru/assets/webui"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/otaruapiserver"
	"github.com/nyaxt/otaru/pb/json"
)

func (o *Otaru) buildApiServerOptions(cfg *ApiServerConfig) ([]apiserver.Option, error) {
	var webuifs http.FileSystem
	override := cfg.WebUIRootPath
	if override == "" {
		webuifs = webui.Assets
	} else {
		logger.Infof(mylog, "Overriding embedded WebUI and serving WebUI at %s", override)
		webuifs = http.Dir(override)
	}

	jwtauth := jwt.NoJWTAuth
	if cfg.JwtPubkeyFile == "" {
		logger.Infof(mylog, "The public key file for JWT auth is not specified. Disabling JWT auth.")
	} else {
		var err error
		jwtauth, err = jwt.NewJWTAuthProviderFromFile(cfg.JwtPubkeyFile)
		if err != nil {
			return nil, err
		}
	}

	options := []apiserver.Option{
		apiserver.ListenAddr(cfg.ListenAddr),
		apiserver.X509KeyPairPath(cfg.CertFile, cfg.KeyFile),
		apiserver.CORSAllowedOrigins(cfg.CORSAllowedOrigins),
		apiserver.SetWebUI(webuifs, "/index.otaru-server.html"),
		apiserver.SetSwaggerJson(json.Assets, "/otaru.swagger.json"),
		apiserver.JWTAuthProvider(jwtauth),
		otaruapiserver.InstallBlobstoreService(o.S, o.DefaultBS, o.CBS),
		otaruapiserver.InstallFileHandler(o.FS, jwtauth),
		otaruapiserver.InstallFileSystemService(o.FS),
		otaruapiserver.InstallINodeDBService(o.IDBS),
		otaruapiserver.InstallLoggerService(),
		otaruapiserver.InstallSystemService(),
	}

	/*
		if cfg.InstallDebugApi {
			logger.Infof(mylog, "Installing Debug APIs.")
			mdebug.Install(o.MGMT)
		}
		mscheduler.Install(o.MGMT, o.S, o.R)
		mgc.Install(o.MGMT, o.S, o)
	*/

	return options, nil
}

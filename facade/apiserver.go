package facade

import (
	"fmt"
	"io/ioutil"
	"net/http"

	jwt "gopkg.in/dgrijalva/jwt-go.v3"

	"github.com/nyaxt/otaru/apiserver"
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

	options := []apiserver.Option{
		apiserver.ListenAddr(cfg.ListenAddr),
		apiserver.X509KeyPair(cfg.CertFile, cfg.KeyFile),
		apiserver.CORSAllowedOrigins(cfg.CORSAllowedOrigins),
		apiserver.SetWebUI(webuifs, "/index.otaru-server.html"),
		apiserver.SetSwaggerJson(json.Assets, "/otaru.swagger.json"),
		otaruapiserver.InstallBlobstoreService(o.S, o.DefaultBS, o.CBS),
		otaruapiserver.InstallFileHandler(o.FS),
		otaruapiserver.InstallFileSystemService(o.FS),
		otaruapiserver.InstallINodeDBService(o.IDBS),
		otaruapiserver.InstallLoggerService(),
		otaruapiserver.InstallSystemService(),
	}

	if cfg.JwtPubkey != "" {
		keytext, err := ioutil.ReadFile(cfg.JwtPubkey)
		if err != nil {
			return nil, fmt.Errorf("Failed to load ECDSA public key file %q: %v", cfg.JwtPubkey, err)
		}

		pk, err := jwt.ParseECPublicKeyFromPEM(keytext)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse ECDSA public key %q: %v", cfg.JwtPubkey, err)
		}

		options = append(options, apiserver.JWTPublicKey(pk))
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

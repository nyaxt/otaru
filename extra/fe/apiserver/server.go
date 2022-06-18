package apiserver

import (
	"context"
	"fmt"
	"net/http"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/assets/webui"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/extra/fe/preview"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

var mylog = logger.Registry().Category("fe-apiserver")
var accesslog = logger.Registry().Category("http-fe")

type registerServiceHandlerFunc func(ctx context.Context, mux *gwruntime.ServeMux, conn *grpc.ClientConn) error

var serviceHandlers = []registerServiceHandlerFunc{
	pb.RegisterBlobstoreServiceHandler,
	pb.RegisterFileSystemServiceHandler,
	pb.RegisterINodeDBServiceHandler,
	pb.RegisterLoggerServiceHandler,
	pb.RegisterSystemInfoServiceHandler,
}

func InstallApiGatewayProxy(hostmap map[string]*cli.Host) apiserver.Option {
	return apiserver.AddMuxHook(func(ctx context.Context, mux *http.ServeMux) error {
		for vhost, h := range hostmap {
			ci := cli.ConnectionInfoFromHost(h)
			conn, err := ci.DialGrpc(ctx)
			if err != nil {
				return err
			}

			gwmux := gwruntime.NewServeMux()

			for _, sh := range serviceHandlers {
				if err := sh(ctx, gwmux, conn); err != nil {
					return fmt.Errorf("Failed to register a grpc-gateway handler for host %q: %w", vhost, err)
				}
			}

			prefix := fmt.Sprintf("/apigw/%s", vhost)
			mux.Handle(prefix+"/", http.StripPrefix(prefix, gwmux))
		}

		return nil
	})
}

func BuildApiServerOptions(cfg *cli.CliConfig) ([]apiserver.Option, error) {
	override := cfg.Fe.WebUIRootPath
	if override != "" {
		logger.Infof(mylog, "Overriding embedded WebUI and serving WebUI at %s", override)
	}

	opts := []apiserver.Option{
		apiserver.ListenAddr(cfg.Fe.ListenAddr),
		apiserver.TLSCertKey(cfg.Fe.Certs, cfg.Fe.Key),
		// apiserver.SetSwaggerJson(json.Assets, "/otaru-fe.swagger.json"),
		apiserver.ServeApiGateway(true),
		apiserver.SetDefaultHandler(webui.WebUIHandler(override, "/index.otaru-fe.html")),
		preview.Install(cfg),
		InstallFeService(cfg),
		InstallProxyHandler(cfg, cfg.Fe.BasicAuthUser, cfg.Fe.BasicAuthPassword),
		InstallApiGatewayProxy(cfg.Host),
	}

	return opts, nil
}

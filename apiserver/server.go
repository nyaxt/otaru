package apiserver

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyaxt/otaru/apiserver/clientauth"
	apiserver_logger "github.com/nyaxt/otaru/apiserver/logger"
	"github.com/nyaxt/otaru/assets/swaggerui"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
)

var LogCategory = logger.Registry().Category("apiserver")
var mylog = LogCategory

type serviceRegistryEntry struct {
	registerServiceServer func(*grpc.Server)
	registerProxy         func(ctx context.Context, mux *gwruntime.ServeMux, endpoint string, opts []grpc.DialOption) error
}

type MuxHook func(mux *http.ServeMux)

type options struct {
	listenAddr string

	cert *x509.Certificate
	key  crypto.PrivateKey

	clientCACert *x509.Certificate

	allowedOrigins []string

	serviceRegistry []serviceRegistryEntry
	accesslogger    logger.Logger
	muxhooks        []MuxHook
	closeC          <-chan struct{}
}

var defaultOptions = options{
	serviceRegistry: []serviceRegistryEntry{},
	accesslogger:    logger.Registry().Category("http-apiserver"),
}

type Option func(*options)

func ListenAddr(listenAddr string) Option {
	return func(o *options) { o.listenAddr = listenAddr }
}

func TLSCertKey(cert *x509.Certificate, key crypto.PrivateKey) Option {
	return func(o *options) {
		o.cert = cert
		o.key = key
	}
}

func ClientCACert(cert *x509.Certificate) Option {
	return func(o *options) {
		o.clientCACert = cert
	}
}

func AddMuxHook(h MuxHook) Option {
	return func(o *options) { o.muxhooks = append(o.muxhooks, h) }
}

func SetDefaultHandler(h http.Handler) Option {
	return AddMuxHook(func(mux *http.ServeMux) {
		mux.Handle("/", h)
	})
}

func SetSwaggerJson(fs http.FileSystem, path string) Option {
	return AddMuxHook(func(mux *http.ServeMux) {
		mux.HandleFunc("/otaru.swagger.json", func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = path
			http.FileServer(fs).ServeHTTP(w, r)
		})
	})
}

func CloseChannel(c <-chan struct{}) Option {
	return func(o *options) { o.closeC = c }
}

func CORSAllowedOrigins(allowedOrigins []string) Option {
	return func(o *options) { o.allowedOrigins = allowedOrigins }
}

func RegisterService(
	registerServiceServer func(*grpc.Server),
	registerProxy func(ctx context.Context, mux *gwruntime.ServeMux, endpoint string, opts []grpc.DialOption) error,
) Option {
	return func(o *options) {
		o.serviceRegistry = append(o.serviceRegistry,
			serviceRegistryEntry{registerServiceServer, registerProxy})
	}
}

func AccessLogger(l logger.Logger) Option {
	return func(o *options) { o.accesslogger = l }
}

// grpcHttpMux, serveSwagger, and Serve functions based on code from:
// https://github.com/philips/grpc-gateway-example

func grpcHttpMux(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
}

func serveSwagger(mux *http.ServeMux) {
	uisrv := http.FileServer(swaggerui.Assets)
	prefix := "/swagger/"
	mux.Handle(prefix, http.StripPrefix(prefix, uisrv))
}

func serveApiGateway(mux *http.ServeMux, opts *options) error {
	// FIXME: impersonate

	tc, err := cli.TLSConfigFromCert(opts.cert)
	if err != nil {
		return err
	}
	tc.Certificates = []tls.Certificate{{
		Certificate: [][]byte{opts.cert.Raw},
		PrivateKey:  opts.key,
	}}

	cred := credentials.NewTLS(tc)
	gwdialopts := []grpc.DialOption{grpc.WithTransportCredentials(cred)}

	ctx := context.Background()
	gwmux := gwruntime.NewServeMux()
	loopbackaddr := opts.listenAddr
	if strings.HasPrefix(loopbackaddr, ":") {
		loopbackaddr = "localhost" + loopbackaddr
	}
	for _, e := range opts.serviceRegistry {
		if err := e.registerProxy(ctx, gwmux, loopbackaddr, gwdialopts); err != nil {
			return fmt.Errorf("Failed to register gw handler: %v", err)
		}
	}
	mux.Handle("/api/", gwmux)

	return nil
}

func Serve(opt ...Option) error {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{opts.cert.Raw},
		PrivateKey:  opts.key,
	}
	grpcCredentials := credentials.NewServerTLSFromCert(cert)

	var clientAuthEnabled bool
	if opts.clientCACert != nil {
		logger.Infof(mylog, "Client certificate authentication is enabled.")
		clientAuthEnabled = true
	} else {
		logger.Infof(mylog, "Client certificate authentication is disabled. Any request to the server will treated as if it were from role \"admin\".")
		clientAuthEnabled = false
	}

	uics := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,
		clientauth.AuthProvider{Disabled: !clientAuthEnabled}.UnaryServerInterceptor(),
		grpc_ctxtags.UnaryServerInterceptor(
			grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.TagBasedRequestFieldExtractor("log_fields")),
		),
		apiserver_logger.UnaryServerInterceptor(),
	}
	grpcServer := grpc.NewServer(
		grpc.Creds(grpcCredentials),
		grpc_middleware.WithUnaryServerChain(uics...),
	)
	for _, e := range opts.serviceRegistry {
		e.registerServiceServer(grpcServer)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("/metrics", promhttp.Handler())
	for _, hook := range opts.muxhooks {
		hook(mux)
	}

	c := cors.New(cors.Options{AllowedOrigins: opts.allowedOrigins})

	lis, err := net.Listen("tcp", opts.listenAddr)
	if err != nil {
		return fmt.Errorf("Failed to listen %q: %v", opts.listenAddr, err)
	}
	closed := false
	if opts.closeC != nil {
		go func() {
			<-opts.closeC
			closed = true
			lis.Close()
		}()
	}

	httpHandler := logger.HttpHandler(opts.accesslogger, logger.Info, c.Handler(mux))
	cliauthtype := tls.NoClientCert
	var clicp *x509.CertPool
	if clientAuthEnabled {
		cliauthtype = tls.VerifyClientCertIfGiven

		clicp = x509.NewCertPool()
		clicp.AddCert(opts.clientCACert)
		// clicp.AddCert(opts.cert)
	}

	httpServer := &http.Server{
		Addr:    opts.listenAddr,
		Handler: grpcHttpMux(grpcServer, httpHandler),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{*cert},
			NextProtos:   []string{"h2"},
			ClientAuth:   cliauthtype,
			ClientCAs:    clicp,
		},
	}

	if err := httpServer.Serve(tls.NewListener(lis, httpServer.TLSConfig)); err != nil {
		if closed {
			// Suppress "use of closed network connection" error if we intentionally closed the listener.
			return nil
		}
		return err
	}
	return nil
}

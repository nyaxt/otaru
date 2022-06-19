package apiserver

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyaxt/otaru/apiserver/clientauth"
	"github.com/nyaxt/otaru/assets/swaggerui"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util/readpem"
)

type serviceRegistryEntry struct {
	registerServiceServer func(*grpc.Server)
	registerProxy         func(ctx context.Context, mux *gwruntime.ServeMux, endpoint string, opts []grpc.DialOption) error
}

type MuxHook func(ctx context.Context, mux *http.ServeMux) error

type options struct {
	listenAddr string

	certs []*x509.Certificate
	key   crypto.PrivateKey

	clientCACert *x509.Certificate

	allowedOrigins []string

	serveApiGateway bool

	serviceRegistry []serviceRegistryEntry
	muxhooks        []MuxHook

	logger *zap.Logger
}

var defaultOptions = options{
	serveApiGateway: false,
	serviceRegistry: []serviceRegistryEntry{},
}

type Option func(*options)

func ListenAddr(listenAddr string) Option {
	return func(o *options) { o.listenAddr = listenAddr }
}

func TLSCertKey(certs []*x509.Certificate, key crypto.PrivateKey) Option {
	return func(o *options) {
		o.certs = certs
		o.key = key
	}
}

func ClientCACert(cert *x509.Certificate) Option {
	return func(o *options) {
		o.clientCACert = cert
	}
}

func ServeApiGateway(b bool) Option {
	return func(o *options) {
		o.serveApiGateway = b
	}
}

func AddMuxHook(h MuxHook) Option {
	return func(o *options) { o.muxhooks = append(o.muxhooks, h) }
}

func SetDefaultHandler(h http.Handler) Option {
	return AddMuxHook(func(_ context.Context, mux *http.ServeMux) error {
		mux.Handle("/", h)
		return nil
	})
}

func SetSwaggerJson(fs http.FileSystem, path string) Option {
	return AddMuxHook(func(_ context.Context, mux *http.ServeMux) error {
		mux.HandleFunc("/otaru.swagger.json", func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = path
			http.FileServer(fs).ServeHTTP(w, r)
		})
		return nil
	})
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

func serveApiGateway(ctx context.Context, mux *http.ServeMux, opts *options) error {
	c := opts.certs[0]

	tc, err := cli.TLSConfigFromCert(c)
	if err != nil {
		return err
	}
	tc.Certificates = []tls.Certificate{{
		Certificate: [][]byte{c.Raw},
		PrivateKey:  opts.key,
	}}

	cred := credentials.NewTLS(tc)
	gwdialopts := []grpc.DialOption{grpc.WithTransportCredentials(cred)}

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

func Serve(ctx context.Context, opt ...Option) error {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}

	l := opts.logger
	if l == nil {
		l = zap.L()
	}
	l = l.Named("apiserver.Serve")
	s := l.Sugar()

	tc := readpem.TLSCertificate(opts.certs, opts.key)
	grpcCredentials := credentials.NewServerTLSFromCert(&tc)

	var clientAuthEnabled bool
	if opts.clientCACert != nil {
		s.Infof("Client certificate authentication is enabled.")
		clientAuthEnabled = true
	} else {
		s.Infof("Client certificate authentication is disabled. Any request to the server will treated as if it were from role \"admin\".")
		clientAuthEnabled = false
	}

	uics := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,
		clientauth.AuthProvider{Disabled: !clientAuthEnabled}.UnaryServerInterceptor(),
		grpc_ctxtags.UnaryServerInterceptor(
			grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.TagBasedRequestFieldExtractor("log_fields")),
		),
		grpc_zap.UnaryServerInterceptor(l.Named("grpc")),
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
		err := hook(ctx, mux)
		if err != nil {
			return err
		}
	}
	if opts.serveApiGateway {
		if err := serveApiGateway(ctx, mux, &opts); err != nil {
			return err
		}
	}

	c := cors.New(cors.Options{AllowedOrigins: opts.allowedOrigins})

	lis, err := net.Listen("tcp", opts.listenAddr)
	if err != nil {
		return fmt.Errorf("Failed to listen %q: %v", opts.listenAddr, err)
	}

	httpHandler := logger.HttpHandler(l, c.Handler(mux))
	cliauthtype := tls.NoClientCert
	var clicp *x509.CertPool
	if clientAuthEnabled {
		cliauthtype = tls.VerifyClientCertIfGiven

		clicp = x509.NewCertPool()
		clicp.AddCert(opts.clientCACert)
		if opts.serveApiGateway {
			clicp.AddCert(opts.certs[0])
		}
	}

	httpServer := &http.Server{
		Addr:    opts.listenAddr,
		Handler: grpcHttpMux(grpcServer, httpHandler),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tc},
			NextProtos:   []string{"h2"},
			ClientAuth:   cliauthtype,
			ClientCAs:    clicp,
		},
	}

	go func() {
		<-ctx.Done()
		httpServer.Close()
		lis.Close()
	}()
	if err = httpServer.Serve(tls.NewListener(lis, httpServer.TLSConfig)); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

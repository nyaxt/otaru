package apiserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/elazarl/go-bindata-assetfs"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyaxt/otaru/logger"
	sjson "github.com/nyaxt/otaru/pb/json"
	"github.com/nyaxt/otaru/webui/swaggerui"
)

var mylog = logger.Registry().Category("apiserver")

type serviceRegistryEntry struct {
	registerServiceServer func(*grpc.Server)
	registerProxy         func(ctx context.Context, mux *gwruntime.ServeMux, endpoint string, opts []grpc.DialOption) error
}

type options struct {
	listenAddr      string
	defaultHandler  http.Handler
	fileHandler     http.Handler
	serviceRegistry []serviceRegistryEntry
}

var defaultOptions = options{
	defaultHandler:  http.NotFoundHandler(),
	fileHandler:     nil,
	serviceRegistry: []serviceRegistryEntry{},
}

type Option func(*options)

func ListenAddr(listenAddr string) Option {
	return func(o *options) { o.listenAddr = listenAddr }
}

func OverrideWebUI(rootPath string) Option {
	return func(o *options) { o.defaultHandler = http.FileServer(http.Dir(rootPath)) }
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
	uisrv := http.FileServer(
		&assetfs.AssetFS{
			Asset:     swaggerui.Asset,
			AssetDir:  swaggerui.AssetDir,
			AssetInfo: swaggerui.AssetInfo,
		})
	prefix := "/swagger/"
	mux.Handle(prefix, http.StripPrefix(prefix, uisrv))

	mux.Handle("/otaru.swagger.json", http.FileServer(
		&assetfs.AssetFS{
			Asset:     sjson.Asset,
			AssetDir:  sjson.AssetDir,
			AssetInfo: sjson.AssetInfo,
		}))
}

func Serve(opt ...Option) (io.Closer, error) {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}

	certtext, err := ioutil.ReadFile("/home/kouhei/otaru-testconf/tls.crt")
	if err != nil {
		return nil, fmt.Errorf("Failed to load TLS cert file: %v", err)
	}
	keytext, err := ioutil.ReadFile("/home/kouhei/otaru-testconf/tls.key")
	if err != nil {
		return nil, fmt.Errorf("Failed to load TLS key file: %v", err)
	}

	cert, err := tls.X509KeyPair(certtext, keytext)
	if err != nil {
		return nil, fmt.Errorf("Failed to create X509KeyPair: %v", err)
	}
	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM(certtext) {
		return nil, fmt.Errorf("certpool creation failure")
	}

	grpcCredentials := credentials.NewServerTLSFromCert(&cert)
	grpcServer := grpc.NewServer(grpc.Creds(grpcCredentials))
	for _, e := range opts.serviceRegistry {
		e.registerServiceServer(grpcServer)
	}

	mux := http.NewServeMux()

	loopbackaddr := "localhost" + opts.listenAddr // FIXME
	ctx := context.Background()
	gwmux := gwruntime.NewServeMux()
	gwdialopts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certpool, "")),
	}
	for _, e := range opts.serviceRegistry {
		if err := e.registerProxy(ctx, gwmux, loopbackaddr, gwdialopts); err != nil {
			return nil, fmt.Errorf("Failed to register gw handler: %v", err)
		}
	}
	mux.Handle("/", opts.defaultHandler)
	mux.Handle("/api/", gwmux)
	if opts.fileHandler != nil {
		filePrefix := "/file/"
		mux.Handle(filePrefix, http.StripPrefix(filePrefix, opts.fileHandler))
	}
	serveSwagger(mux)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:9000"}, // gulp devsrv
	})

	lis, err := net.Listen("tcp", opts.listenAddr)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen \"%s\": %v", opts.listenAddr, err)
	}

	httpHandler := c.Handler(mux)
	httpServer := &http.Server{
		Addr:    opts.listenAddr,
		Handler: grpcHttpMux(grpcServer, httpHandler),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2"},
		},
	}

	go func() {
		if err := httpServer.Serve(tls.NewListener(lis, httpServer.TLSConfig)); err != nil {
			logger.Debugf(mylog, "http.Server.Serve exit: %v", err)
		}
	}()
	return lis, nil
}

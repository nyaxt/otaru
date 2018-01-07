package apiserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
	x509util "github.com/nyaxt/otaru/util/x509"
	"github.com/nyaxt/otaru/webui"
	"github.com/nyaxt/otaru/webui/swaggerui"
)

var mylog = logger.Registry().Category("apiserver")
var accesslog = logger.Registry().Category("http-apiserver")

type serviceRegistryEntry struct {
	registerServiceServer func(*grpc.Server)
	registerProxy         func(ctx context.Context, mux *gwruntime.ServeMux, endpoint string, opts []grpc.DialOption) error
}

type options struct {
	listenAddr string
	certFile   string
	keyFile    string

	allowedOrigins  []string
	defaultHandler  http.Handler
	fileHandler     http.Handler
	serviceRegistry []serviceRegistryEntry
	closeC          <-chan struct{}
}

var defaultOptions = options{
	defaultHandler: http.FileServer(&assetfs.AssetFS{
		Asset:     webui.Asset,
		AssetDir:  webui.AssetDir,
		AssetInfo: webui.AssetInfo,
	}),
	fileHandler:     nil,
	serviceRegistry: []serviceRegistryEntry{},
}

type Option func(*options)

func ListenAddr(listenAddr string) Option {
	return func(o *options) { o.listenAddr = listenAddr }
}

func X509KeyPair(certFile, keyFile string) Option {
	return func(o *options) {
		o.certFile = certFile
		o.keyFile = keyFile
	}
}

func OverrideWebUI(rootPath string) Option {
	return func(o *options) { o.defaultHandler = http.FileServer(http.Dir(rootPath)) }
}

func CloseChannel(c <-chan struct{}) Option {
	return func(o *options) { o.closeC = c }
}

func CORSAllowedOrigins(allowedOrigins []string) Option {
	return func(o *options) { o.allowedOrigins = allowedOrigins }
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

func serveApiGateway(mux *http.ServeMux, opts *options, certtext []byte) error {
	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM(certtext) {
		return fmt.Errorf("certpool creation failure")
	}
	serverName, err := x509util.FindServerName(certpool.Subjects())
	if err != nil {
		return fmt.Errorf("failed to find server name")
	}
	logger.Infof(mylog, "Using server name \"%s\" for grpc loopback connection.", serverName)
	gwdialopts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certpool, serverName)),
	}

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

	certtext, err := ioutil.ReadFile(opts.certFile)
	if err != nil {
		return fmt.Errorf("Failed to load TLS cert file: %v", err)
	}
	keytext, err := ioutil.ReadFile(opts.keyFile)
	if err != nil {
		return fmt.Errorf("Failed to load TLS key file: %v", err)
	}

	cert, err := tls.X509KeyPair(certtext, keytext)
	if err != nil {
		return fmt.Errorf("Failed to create X509KeyPair: %v", err)
	}

	grpcCredentials := credentials.NewServerTLSFromCert(&cert)
	grpcServer := grpc.NewServer(grpc.Creds(grpcCredentials))
	for _, e := range opts.serviceRegistry {
		e.registerServiceServer(grpcServer)
	}

	mux := http.NewServeMux()
	mux.Handle("/", opts.defaultHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	if err := serveApiGateway(mux, &opts, certtext); err != nil {
		return err
	}
	if opts.fileHandler != nil {
		filePrefix := "/file/"
		mux.Handle(filePrefix, http.StripPrefix(filePrefix, opts.fileHandler))
	}
	serveSwagger(mux)

	c := cors.New(cors.Options{AllowedOrigins: opts.allowedOrigins})

	lis, err := net.Listen("tcp", opts.listenAddr)
	if err != nil {
		return fmt.Errorf("Failed to listen \"%s\": %v", opts.listenAddr, err)
	}
	if opts.closeC != nil {
		go func() {
			<-opts.closeC
			lis.Close()
		}()
	}

	httpHandler := logger.HttpHandler(accesslog, logger.Info, c.Handler(mux))
	httpServer := &http.Server{
		Addr:    opts.listenAddr,
		Handler: grpcHttpMux(grpcServer, httpHandler),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2"},
		},
	}

	return httpServer.Serve(tls.NewListener(lis, httpServer.TLSConfig))
}

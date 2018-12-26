package apiserver

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	jwt "github.com/nyaxt/otaru/apiserver/jwt"
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

	certFile string
	keyFile  string

	allowedOrigins []string

	jwtauth *jwt.JWTAuthProvider

	serviceRegistry []serviceRegistryEntry
	accesslogger    logger.Logger
	muxhooks        []MuxHook
	closeC          <-chan struct{}
}

var defaultOptions = options{
	serviceRegistry: []serviceRegistryEntry{},
	accesslogger:    logger.Registry().Category("http-apiserver"),
	jwtauth:         jwt.NoJWTAuth,
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

func AddMuxHook(h MuxHook) Option {
	return func(o *options) { o.muxhooks = append(o.muxhooks, h) }
}

func SetWebUI(fs http.FileSystem, indexpath string) Option {
	return AddMuxHook(func(mux *http.ServeMux) {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				r.URL.Path = indexpath
			}
			http.FileServer(fs).ServeHTTP(w, r)
		})
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

func JWTAuthProvider(jwtauth *jwt.JWTAuthProvider) Option {
	return func(o *options) {
		o.jwtauth = jwtauth
	}
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

func serveApiGateway(mux *http.ServeMux, opts *options, certtext []byte) error {
	tc, err := cli.TLSConfigFromCertText(certtext)
	if err != nil {
		return err
	}
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

	certtext, err := ioutil.ReadFile(opts.certFile)
	if err != nil {
		return fmt.Errorf("Failed to load TLS cert file %q: %v", opts.certFile, err)
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

	if opts.jwtauth == jwt.NoJWTAuth {
		logger.Infof(mylog, "Authentication is disabled. Any request to the server will treated as if it were from role \"admin\".")
	} else {
		logger.Infof(mylog, "Authentication is enabled.")
	}

	uics := []grpc.UnaryServerInterceptor{
		opts.jwtauth.UnaryServerInterceptor(),
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
		w.Write([]byte("ok\n"))
	})
	if err := serveApiGateway(mux, &opts, certtext); err != nil {
		return err
	}
	for _, hook := range opts.muxhooks {
		hook(mux)
	}
	serveSwagger(mux)

	c := cors.New(cors.Options{AllowedOrigins: opts.allowedOrigins})

	lis, err := net.Listen("tcp", opts.listenAddr)
	if err != nil {
		return fmt.Errorf("Failed to listen \"%s\": %v", opts.listenAddr, err)
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
	httpServer := &http.Server{
		Addr:    opts.listenAddr,
		Handler: grpcHttpMux(grpcServer, httpHandler),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2"},
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

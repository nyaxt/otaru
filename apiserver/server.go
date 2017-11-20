package apiserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/pb"
)

var clog = logger.Registry().Category("apiserver")

func grpcHttpMux(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
}

func Serve(addr string) error {
	certtext, err := ioutil.ReadFile("/home/kouhei/otaru-testconf/tls.crt")
	if err != nil {
		return fmt.Errorf("Failed to load TLS cert file: %v", err)
	}
	keytext, err := ioutil.ReadFile("/home/kouhei/otaru-testconf/tls.key")
	if err != nil {
		return fmt.Errorf("Failed to load TLS key file: %v", err)
	}

	cert, err := tls.X509KeyPair(certtext, keytext)
	if err != nil {
		return fmt.Errorf("Failed to create X509KeyPair: %v", err)
	}
	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM(certtext) {
		return fmt.Errorf("certpool creation failure")
	}

	grpcCredentials := credentials.NewClientTLSFromCert(certpool, "localhost:10249")
	opts := []grpc.ServerOption{grpc.Creds(grpcCredentials)}
	grpcServer := grpc.NewServer(opts...)

	mux := http.NewServeMux()
	// mux.HandleFunc("/swagger.json", ...

	loopbackaddr := "localhost" + addr // FIXME
	logger.Debugf(clog, "loopbackaddr: %v", loopbackaddr)
	ctx := context.Background()
	gwmux := gwruntime.NewServeMux()
	allowcertcred := credentials.NewTLS(&tls.Config{
		ServerName: "localhost",
		RootCAs:    certpool,
	})
	gwdialopts := []grpc.DialOption{grpc.WithTransportCredentials(allowcertcred)}
	if err := pb.RegisterSystemInfoServiceHandlerFromEndpoint(ctx, gwmux, loopbackaddr, gwdialopts); err != nil {
		return fmt.Errorf("Failed to register gw handler: %v", err)
	}
	logger.Debugf(clog, "RegisteredSystemInfoServiceHandler")
	mux.Handle("/", gwmux)
	// serveSwagger(mux)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:9000"}, // gulp devsrv
	})

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("Failed to listen \"%s\": %v", addr, err)
	}
	logger.Debugf(clog, "StartListen")

	httpHandler := c.Handler(mux)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: grpcHttpMux(grpcServer, httpHandler),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2"},
		},
	}

	if err := httpServer.Serve(tls.NewListener(lis, httpServer.TLSConfig)); err != nil {
		return fmt.Errorf("http.Server.Serve: %v", err)
	}
	return nil
}

package webdav

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/nyaxt/otaru/basicauth"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util/readpem"
	"go.uber.org/zap"
)

func Serve(ctx context.Context, cfg *cli.CliConfig) error {
	s := zap.S().Named("webdav.Serve")

	wcfg := cfg.Webdav

	var handler http.Handler
	handler = &Handler{cfg}

	if wcfg.ListenAddr == "" {
		return errors.New("Webdav server listen addr must be configured.")
	}

	lis, err := net.Listen("tcp", wcfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("Failed to listen \"%s\": %v", wcfg.ListenAddr, err)
	}

	if wcfg.BasicAuthPassword == "" {
		s.Warnf("Basic auth not enabled!")
	} else {
		s.Infof("Basic auth enabled.")
		handler = &basicauth.Handler{
			User:     wcfg.BasicAuthUser,
			Password: wcfg.BasicAuthPassword,
			Handler:  handler,
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("/", handler)

	loghandler := logger.HttpHandler(s.Desugar(), mux)

	// Note: This doesn't enable h2. Reconsider this if there is a webdav client w/ h2 support.
	tc := readpem.TLSCertificate(wcfg.Certs, wcfg.Key)
	httpsrv := http.Server{
		Addr:    wcfg.ListenAddr,
		Handler: loghandler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tc},
			NextProtos:   []string{"http/1.1"},
		},
	}
	tlis := tls.NewListener(lis, httpsrv.TLSConfig)

	go func() {
		<-ctx.Done()
		httpsrv.Close()
		lis.Close()
	}()

	if err := httpsrv.Serve(tlis); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

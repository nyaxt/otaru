package webdav

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/nyaxt/otaru/basicauth"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("fe-webdav")
var accesslog = logger.Registry().Category("http-webdav")

func Serve(cfg *cli.CliConfig, closeC <-chan struct{}) error {
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

	closed := false
	if closeC != nil {
		go func() {
			<-closeC
			closed = true
			lis.Close()
		}()
	}

	cert, err := tls.LoadX509KeyPair(wcfg.CertFile, wcfg.KeyFile)
	if err != nil {
		return fmt.Errorf("Failed to load webdav {cert,key} pair: %v", err)
	}

	if wcfg.BasicAuthPassword == "" {
		logger.Warningf(mylog, "Basic auth not enabled!")
	} else {
		logger.Infof(mylog, "Basic auth enabled.")
		handler = &basicauth.Handler{
			User:     wcfg.BasicAuthUser,
			Password: wcfg.BasicAuthPassword,
			Handler:  handler,
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	mux.Handle("/", handler)

	loghandler := logger.HttpHandler(accesslog, logger.Info, mux)

	// Note: This doesn't enable h2. Reconsider this if there is a webdav client w/ h2 support.
	httpsrv := http.Server{
		Addr:    wcfg.ListenAddr,
		Handler: loghandler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"http/1.1"},
		},
	}
	tlis := tls.NewListener(lis, httpsrv.TLSConfig)

	if err := httpsrv.Serve(tlis); err != nil {
		if closed {
			// Suppress "use of closed network connection" error if we intentionally closed the listener.
			return nil
		}
		return err
	}
	return nil
}

package cli

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyaxt/otaru/logger"
)

func ConnectionInfo(cfg *CliConfig, vhost string) (string, *tls.Config, error) {
	h, ok := cfg.Host[vhost]
	if !ok {
		return "", nil, errors.New("Unknown vhost.")
	}

	var tc *tls.Config
	if h.ExpectedCertFile != "" {
		certfile := h.ExpectedCertFile
		certtext, err := ioutil.ReadFile(certfile)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to read specified cert file: %s", certfile)
		}
		logger.Debugf(Log, "Expecting server cert to match: %v", certfile)

		tc, err = TLSConfigFromCertText(certtext)
		if err != nil {
			return "", nil, err
		}
	} else {
		logger.Debugf(Log, "No server cert expectation given.")
		tc = &tls.Config{}
	}
	return h.ApiEndpoint, tc, nil
}

func DialVhost(cfg *CliConfig, vhost string) (*grpc.ClientConn, error) {
	ep, tc, err := ConnectionInfo(cfg, vhost)
	if err != nil {
		return nil, fmt.Errorf("Failed to init conn info to vhost \"%s\". err: %v", vhost, err)
	}
	cred := credentials.NewTLS(tc)
	conn, err := grpc.Dial(ep, grpc.WithTransportCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("Failed to grpc.Dial(\"%s\"). err: %v", ep, err)
	}
	return conn, nil
}

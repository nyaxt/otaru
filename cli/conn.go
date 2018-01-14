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

func getConnInfo(cfg *CliConfig, vhost string) (string, []grpc.DialOption, error) {
	h, ok := cfg.Host[vhost]
	if !ok {
		return "", nil, errors.New("Unknown vhost.")
	}

	var transcred credentials.TransportCredentials
	if h.ExpectedCertFile != "" {
		certfile := h.ExpectedCertFile
		certtext, err := ioutil.ReadFile(certfile)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to read specified cert file: %s", certfile)
		}
		logger.Debugf(Log, "Expecting server cert to match: %v", certfile)

		transcred, err = ClientTransportCredentialFromCertText(certtext)
		if err != nil {
			return "", nil, err
		}
	} else {
		logger.Debugf(Log, "No server cert expectation given.")
		transcred = credentials.NewTLS(&tls.Config{})
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(transcred)}
	return h.ApiEndpoint, opts, nil
}

func DialVhost(cfg *CliConfig, vhost string) (*grpc.ClientConn, error) {
	ep, opts, err := getConnInfo(cfg, vhost)
	if err != nil {
		return nil, fmt.Errorf("Failed to init conn info to vhost \"%s\". err: %v", vhost, err)
	}
	conn, err := grpc.Dial(ep, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to grpc.Dial(\"%s\"). err: %v", ep, err)
	}
	return conn, nil
}

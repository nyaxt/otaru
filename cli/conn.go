package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyaxt/otaru/logger"
)

var ErrUnknownVhost = errors.New("Unknown vhost.")

type ConnectionInfo struct {
	ApiEndpoint string
	TLSConfig   *tls.Config
}

func QueryConnectionInfo(cfg *CliConfig, vhost string) (*ConnectionInfo, error) {
	h, ok := cfg.Host[vhost]
	if !ok {
		return nil, ErrUnknownVhost
	}

	ci, err := connectionInfoFromHost(h)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

func connectionInfoFromHost(h *Host) (*ConnectionInfo, error) {
	var tc *tls.Config

	if h.CACert != nil {
		cp := x509.NewCertPool()
		cp.AddCert(h.CACert)

		tc = &tls.Config{RootCAs: cp}
	} else if h.ExpectedCertFile != "" {
		// FIXME: This is obsolete non-sense. Remove
		logger.Infof(Log, "The use of ExpectedCertFile is deprecated!")

		certfile := h.ExpectedCertFile
		certtext, err := ioutil.ReadFile(certfile)
		if err != nil {
			return nil, fmt.Errorf("Failed to read specified cert file: %s", certfile)
		}
		logger.Debugf(Log, "Expecting server cert to match: %v", certfile)

		block, _ := pem.Decode(certtext)
		if block == nil {
			return nil, fmt.Errorf("Failed to parse specified cert file")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		tc, err = TLSConfigFromCert(cert)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Debugf(Log, "No server cert expectation given for %q.", h.ApiEndpoint)
		tc = &tls.Config{}
	}

	if h.Cert != nil {
		logger.Debugf(Log, "Configuring client cert: cn=%s", h.Cert.Subject.CommonName)
		tc.Certificates = []tls.Certificate{{
			Certificate: [][]byte{h.Cert.Raw},
			PrivateKey:  h.Key,
		}}
	}

	if h.OverrideServerName != "" {
		tc.ServerName = h.OverrideServerName
	}

	return &ConnectionInfo{
		ApiEndpoint: h.ApiEndpoint,
		TLSConfig:   tc,
	}, nil
}

func (ci *ConnectionInfo) DialGrpc(ctx context.Context) (*grpc.ClientConn, error) {
	logger.Debugf(Log, "about to dial %s with len(tlsc.Certificates)=%d", ci.ApiEndpoint, len(ci.TLSConfig.Certificates))

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(ci.TLSConfig)),
	}
	conn, err := grpc.DialContext(ctx, ci.ApiEndpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to grpc.Dial(%q). err: %v", ci.ApiEndpoint, err)
	}
	return conn, nil
}

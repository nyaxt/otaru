package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	"github.com/nyaxt/otaru/logger"
)

var ErrUnknownVhost = errors.New("Unknown vhost.")

type ConnectionInfo struct {
	ApiEndpoint string
	TLSConfig   *tls.Config
	AuthToken   string
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

	if h.CACert == nil && h.CACertFile != "" {
		certpem, err := ioutil.ReadFile(h.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to read specified CACert file: %s", h.CACertFile)
		}

		block, _ := pem.Decode(certpem)
		if block == nil {
			return nil, fmt.Errorf("Failed to parse specified cert file")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		h.CACert = cert
	}

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

	if h.OverrideServerName != "" {
		tc.ServerName = h.OverrideServerName
	}

	return &ConnectionInfo{
		ApiEndpoint: h.ApiEndpoint,
		TLSConfig:   tc,
		AuthToken:   h.AuthToken,
	}, nil
}

func (ci *ConnectionInfo) DialGrpc(ctx context.Context) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(ci.TLSConfig)),
	}
	if ci.AuthToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ci.AuthToken})
		opts = append(opts,
			grpc.WithPerRPCCredentials(oauth.TokenSource{ts}))
	}
	conn, err := grpc.DialContext(ctx, ci.ApiEndpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to grpc.Dial(%q). err: %v", ci.ApiEndpoint, err)
	}
	return conn, nil
}

package cli

import (
	"context"
	"crypto/tls"
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

	ci, err := ConnectionInfoFromHost(h)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

func ConnectionInfoFromHost(h *Host) (*ConnectionInfo, error) {
	var tc *tls.Config
	if h.ExpectedCertFile != "" {
		certfile := h.ExpectedCertFile
		certtext, err := ioutil.ReadFile(certfile)
		if err != nil {
			return nil, fmt.Errorf("Failed to read specified cert file: %s", certfile)
		}
		logger.Debugf(Log, "Expecting server cert to match: %v", certfile)

		tc, err = TLSConfigFromCertText(certtext)
		if err != nil {
			return nil, err
		}
	} else if h.OverrideServerName != "" {
		tc = &tls.Config{ServerName: h.OverrideServerName}
	} else {
		logger.Debugf(Log, "No server cert expectation given for %q.", h.ApiEndpoint)
		tc = &tls.Config{}
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

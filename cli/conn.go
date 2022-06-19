package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	ci := ConnectionInfoFromHost(h)
	return ci, nil
}

func ConnectionInfoFromHost(h *Host) *ConnectionInfo {
	var tc tls.Config

	if h.CACert != nil {
		cp := x509.NewCertPool()
		cp.AddCert(h.CACert)

		tc.RootCAs = cp
	}
	if h.Cert != nil {
		zap.S().Infof("Configuring client cert: cn=%s", h.Cert.Subject.CommonName)
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
		TLSConfig:   &tc,
	}
}

func (ci *ConnectionInfo) DialGrpc(ctx context.Context) (*grpc.ClientConn, error) {
	zap.S().Infof("about to dial %s with len(tlsc.Certificates)=%d", ci.ApiEndpoint, len(ci.TLSConfig.Certificates))

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(ci.TLSConfig)),
	}
	conn, err := grpc.DialContext(ctx, ci.ApiEndpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to grpc.Dial(%q). err: %v", ci.ApiEndpoint, err)
	}
	return conn, nil
}

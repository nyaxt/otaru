package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/nyaxt/otaru/logger"
	x509util "github.com/nyaxt/otaru/util/x509"
)

func TLSConfigFromCertText(certtext []byte) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM(certtext) {
		return nil, fmt.Errorf("certpool creation failure")
	}
	serverName, err := x509util.FindServerName(certpool.Subjects())
	if err != nil {
		return nil, fmt.Errorf("failed to find server name")
	}
	logger.Infof(Log, "Using server name \"%s\" for grpc loopback connection.", serverName)

	return &tls.Config{ServerName: serverName, RootCAs: certpool}, nil
}

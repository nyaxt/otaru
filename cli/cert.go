package cli

import (
	"crypto/x509"
	"fmt"

	"google.golang.org/grpc/credentials"

	"github.com/nyaxt/otaru/logger"
	x509util "github.com/nyaxt/otaru/util/x509"
)

func ClientTransportCredentialFromCertText(certtext []byte) (credentials.TransportCredentials, error) {
	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM(certtext) {
		return nil, fmt.Errorf("certpool creation failure")
	}
	serverName, err := x509util.FindServerName(certpool.Subjects())
	if err != nil {
		return nil, fmt.Errorf("failed to find server name")
	}
	logger.Infof(Log, "Using server name \"%s\" for grpc loopback connection.", serverName)

	return credentials.NewClientTLSFromCert(certpool, serverName), nil
}

package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"go.uber.org/zap"
)

func TLSConfigFromCert(cert *x509.Certificate) (*tls.Config, error) {
	cp := x509.NewCertPool()
	cp.AddCert(cert)

	// Subject Alt Name DNSNames are preferred over CommonName, since CommonName is ignored when SAN available.
	var serverNames []string
	serverNames = append(serverNames, cert.DNSNames...)
	serverNames = append(serverNames, cert.Subject.CommonName)

	if len(serverNames) == 0 {
		return nil, fmt.Errorf("Failed to find any valid server name from given certs.")
	}

	zap.S().Infof("Found server names %v. Using the first entry for grpc loopback connection.", serverNames)
	return &tls.Config{ServerName: serverNames[0], RootCAs: cp}, nil
}

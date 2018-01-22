package cli

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/nyaxt/otaru/logger"
)

func TLSConfigFromCertText(certtext []byte) (*tls.Config, error) {
	cp := x509.NewCertPool()

	serverNames := []string{}
	for len(certtext) > 0 {
		var block *pem.Block
		block, certtext = pem.Decode(certtext)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		cp.AddCert(c)

		// Subject Alt Name DNSNames are preferred over CommonName, since CommonName is ignored when SAN available.
		serverNames = append(serverNames, c.DNSNames...)
		serverNames = append(serverNames, c.Subject.CommonName)
	}

	if len(serverNames) == 0 {
		return nil, fmt.Errorf("Failed to find any valid server name from given certs.")
	}

	logger.Infof(Log, "Found server names %v. Using the first entry for grpc loopback connection.", serverNames)
	return &tls.Config{ServerName: serverNames[0], RootCAs: cp}, nil
}

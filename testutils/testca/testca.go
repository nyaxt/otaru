package testca

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"net/http"
)

var (
	//go:embed cert.pem
	CertPEM []byte

	//go:embed cert-key.pem
	KeyPEM []byte

	//go:embed cacert.pem
	CACertPEM []byte

	CertBytes   []byte
	KeyBytes    []byte
	CACertBytes []byte

	Cert   *x509.Certificate
	CACert *x509.Certificate

	CertPool *x509.CertPool
)

var TLSHTTPClient *http.Client

func init() {
	block, _ := pem.Decode(CertPEM)
	CertBytes = block.Bytes

	block, _ = pem.Decode(KeyPEM)
	KeyBytes = block.Bytes

	block, _ = pem.Decode(CACertPEM)
	CACertBytes = block.Bytes

	c, err := x509.ParseCertificate(CertBytes)
	if err != nil {
		panic(err)
	}
	Cert = c

	c, err = x509.ParseCertificate(CACertBytes)
	if err != nil {
		panic(err)
	}
	CACert = c

	CertPool := x509.NewCertPool()
	CertPool.AddCert(CACert)

	TLSHTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: CertPool,
			},
		},
	}
}

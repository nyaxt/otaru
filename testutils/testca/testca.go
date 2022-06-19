package testca

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"net/http"
)

type PrivateKey struct {
	DER    []byte
	Parsed crypto.PrivateKey
}

var (
	//go:embed cert.pem
	CertPEM []byte
	Cert    *x509.Certificate

	//go:embed cert-key.pem
	KeyPEM []byte
	Key    PrivateKey

	//go:embed cacert.pem
	CACertPEM []byte
	CACert    *x509.Certificate

	Certs    []*x509.Certificate
	CertPool *x509.CertPool

	//go:embed clientauth_admin.pem
	ClientAuthAdminCertPEM []byte
	ClientAuthAdminCert    *x509.Certificate
	ClientAuthAdminCerts   []*x509.Certificate

	//go:embed clientauth_admin-key.pem
	ClientAuthAdminKeyPEM []byte
	ClientAuthAdminKey    PrivateKey

	//go:embed clientauth_readonly.pem
	ClientAuthReadOnlyCertPEM []byte
	ClientAuthReadOnlyCert    *x509.Certificate
	ClientAuthReadOnlyCerts   []*x509.Certificate

	//go:embed clientauth_readonly-key.pem
	ClientAuthReadOnlyKeyPEM []byte
	ClientAuthReadOnlyKey    PrivateKey

	//go:embed clientauth_invalid.pem
	ClientAuthInvalidCertPEM []byte
	ClientAuthInvalidCert    *x509.Certificate

	//go:embed clientauth_invalid-key.pem
	ClientAuthInvalidKeyPEM []byte
	ClientAuthInvalidKey    PrivateKey

	//go:embed clientauth_cacert.pem
	ClientAuthCACertPEM []byte
	ClientAuthCACert    *x509.Certificate
)

var TLSHTTPClient *http.Client

func init() {
	block, _ := pem.Decode(CertPEM)
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	Cert = c

	block, _ = pem.Decode(KeyPEM)
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	Key = PrivateKey{
		DER:    block.Bytes,
		Parsed: key,
	}

	block, _ = pem.Decode(CACertPEM)
	c, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	CACert = c

	Certs = []*x509.Certificate{Cert, CACert}

	CertPool := x509.NewCertPool()
	CertPool.AddCert(CACert)

	block, _ = pem.Decode(ClientAuthCACertPEM)
	c, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	ClientAuthCACert = c

	block, _ = pem.Decode(ClientAuthAdminCertPEM)
	c, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	ClientAuthAdminCert = c
	ClientAuthAdminCerts = []*x509.Certificate{
		ClientAuthAdminCert, ClientAuthCACert}

	block, _ = pem.Decode(ClientAuthAdminKeyPEM)
	key, err = x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	ClientAuthAdminKey = PrivateKey{
		DER:    block.Bytes,
		Parsed: key,
	}

	TLSHTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: CertPool,
				Certificates: []tls.Certificate{{
					Certificate: [][]byte{ClientAuthAdminCert.Raw},
					PrivateKey:  ClientAuthAdminKey.Parsed,
				}},
			},
		},
	}

	block, _ = pem.Decode(ClientAuthReadOnlyCertPEM)
	c, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	ClientAuthReadOnlyCert = c
	ClientAuthReadOnlyCerts = []*x509.Certificate{
		ClientAuthReadOnlyCert, ClientAuthCACert}

	block, _ = pem.Decode(ClientAuthReadOnlyKeyPEM)
	key, err = x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	ClientAuthReadOnlyKey = PrivateKey{
		DER:    block.Bytes,
		Parsed: key,
	}

	block, _ = pem.Decode(ClientAuthInvalidCertPEM)
	c, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	ClientAuthInvalidCert = c

	block, _ = pem.Decode(ClientAuthInvalidKeyPEM)
	key, err = x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	ClientAuthInvalidKey = PrivateKey{
		DER:    block.Bytes,
		Parsed: key,
	}
}

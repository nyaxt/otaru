package readpem

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
)

func ReadCertificatesFile(configkey, path string, pcerts *[]*x509.Certificate) error {
	if len(*pcerts) != 0 {
		if path != "" {
			return fmt.Errorf("%[1]s and %[1]sFile are both specified", configkey)
		}
		return nil
	}

	if path == "" {
		return nil
	}
	path = os.ExpandEnv(path)

	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read %sFile %q: %w", configkey, path, err)
	}

	var cs []*x509.Certificate

	for len(bs) > 0 {
		var block *pem.Block
		block, bs = pem.Decode(bs)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			return fmt.Errorf("%d-th PEM block on %sFile %q is not CERTIFICATE", len(cs)+1, configkey, path)
		}

		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("Failed to parse %d-th PEM block on %sFile %q: %w", len(cs)+1, configkey, path, err)
		}
		cs = append(cs, c)
	}

	*pcerts = cs
	return nil
}

func ReadCertificateFile(configkey, path string, pcert **x509.Certificate) error {
	if *pcert != nil {
		if path != "" {
			return fmt.Errorf("%[1]s and %[1]sFile are both specified", configkey)
		}
		return nil
	}

	var cs []*x509.Certificate
	if err := ReadCertificatesFile(configkey, path, &cs); err != nil {
		return err
	}
	if len(cs) != 1 {
		return fmt.Errorf("Expected 1 certificate on %sFile: %q, but got %d certificates", configkey, path, len(cs))
	}
	*pcert = cs[0]
	return nil
}

func ReadKeyFile(configkey, path string, pkey *crypto.PrivateKey) error {
	if *pkey != nil {
		if path != "" {
			return fmt.Errorf("%[1]s and %[1]sFile are both specified", configkey)
		}
		return nil
	}

	if path == "" {
		return nil
	}
	path = os.ExpandEnv(path)

	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read %sFile %q: %w", configkey, path, err)
	}

	block, _ := pem.Decode(bs)

	var k crypto.PrivateKey
	if eck, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		k = eck
	} else if p1k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		k = p1k
	} else {
		return fmt.Errorf("Failed to parse %sFile %q as EC nor PKCS1 private key", configkey, path)
	}

	*pkey = k
	return nil
}

func TLSCertificate(certs []*x509.Certificate, key crypto.PrivateKey) tls.Certificate {
	certDers := make([][]byte, len(certs))
	for i, c := range certs {
		certDers[i] = c.Raw
	}
	return tls.Certificate{
		Certificate: certDers,
		PrivateKey:  key,
	}
}

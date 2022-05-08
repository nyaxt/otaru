package readpem

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
)

func ReadCertificateFile(configkey, path string, pcert **x509.Certificate) error {
	if *pcert != nil {
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
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to parse %sFile %q: %w", configkey, path, err)
	}

	*pcert = c
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

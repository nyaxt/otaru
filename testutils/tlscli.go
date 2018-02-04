package testutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
)

func TLSHTTPClient(certFile string) (*http.Client, error) {
	certtext, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("cert file read: %v", err)
	}
	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM(certtext) {
		return nil, fmt.Errorf("certpool creation failure")
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certpool,
			},
		},
	}, nil
}

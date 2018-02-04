package apiserver_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/nyaxt/otaru/apiserver"
	tu "github.com/nyaxt/otaru/testutils"
)

const testListenAddr = "localhost:20246"

func init() { tu.EnsureLogger() }

func TestServe_Healthz(t *testing.T) {
	otarudir := os.Getenv("OTARUDIR")
	certFile := path.Join(otarudir, "cert.pem")
	keyFile := path.Join(otarudir, "cert-key.pem")

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		if err := Serve(
			ListenAddr(testListenAddr),
			X509KeyPair(certFile, keyFile),
			CloseChannel(closeC),
		); err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		close(joinC)
	}()

	// FIXME: wait until Serve to actually start accepting conns
	time.Sleep(100 * time.Millisecond)

	c, err := tu.TLSHTTPClient(certFile)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	resp, err := c.Get("https://" + testListenAddr + "/healthz")
	if err != nil {
		t.Errorf("http.Get: %v", err)
		return
	}
	cont, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ReadAll(http.Get resp.Body): %v", err)
		return
	}
	if !bytes.Equal(cont, []byte("ok\n")) {
		t.Errorf("unexpected content: %v", cont)
	}
	resp.Body.Close()

	close(closeC)
	<-joinC
}

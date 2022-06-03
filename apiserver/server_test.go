package apiserver_test

import (
	"bytes"
	_ "embed"
	"io/ioutil"
	"testing"
	"time"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/testutils/testca"
)

const testListenAddr = "localhost:20246"

func init() { testutils.EnsureLogger() }

func TestServe_Healthz(t *testing.T) {
	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testListenAddr),
			apiserver.TLSCertKey(testca.Certs, testca.Key.Parsed),
			apiserver.ClientCACert(testca.ClientAuthCACert),
			apiserver.CloseChannel(closeC),
		); err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		close(joinC)
	}()

	// FIXME: wait until Serve to actually start accepting conns
	time.Sleep(100 * time.Millisecond)

	resp, err := testca.TLSHTTPClient.Get("https://" + testListenAddr + "/healthz")
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

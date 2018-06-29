package apiserver_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	feapiserver "github.com/nyaxt/otaru/extra/fe/apiserver"
	"github.com/nyaxt/otaru/testutils"
)

const (
	testBeListenAddr = "localhost:20246"
	testFeListenAddr = "localhost:20247"
)

func init() { testutils.EnsureLogger() }

type mockRequest struct {
	url.URL
	Method  string
	Payload []byte
}

type mockBackend struct {
	closeC chan struct{}
	joinC  chan struct{}

	Reqs []mockRequest
}

func runMockBackend(certFile, keyFile string) *mockBackend {
	m := &mockBackend{
		closeC: make(chan struct{}),
		joinC:  make(chan struct{}),
		Reqs:   []mockRequest{},
	}

	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testBeListenAddr),
			apiserver.X509KeyPair(certFile, keyFile),
			apiserver.CloseChannel(m.closeC),
			apiserver.AddMuxHook(func(mux *http.ServeMux) {
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					mr := mockRequest{
						URL:    *r.URL,
						Method: r.Method,
					}
					if r.Body != nil {
						payload, err := ioutil.ReadAll(r.Body)
						if err != nil {
							panic(err)
						}
						mr.Payload = payload
					}
					m.Reqs = append(m.Reqs, mr)
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte("fuga\n"))
				})
			}),
		); err != nil {
			panic(err)
		}
		close(m.joinC)
	}()

	return m
}

func (m *mockBackend) Terminate() {
	close(m.closeC)
	<-m.joinC
}

func TestProxyHandler(t *testing.T) {
	otarudir := os.Getenv("OTARUDIR")
	certFile := path.Join(otarudir, "cert.pem")
	keyFile := path.Join(otarudir, "cert-key.pem")
	cfg := &cli.CliConfig{
		Host: map[string]*cli.Host{
			"hostfoo": &cli.Host{
				ApiEndpoint:      testBeListenAddr,
				ExpectedCertFile: certFile,
			},
		},
	}

	m := runMockBackend(certFile, keyFile)

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testFeListenAddr),
			apiserver.X509KeyPair(certFile, keyFile),
			apiserver.CloseChannel(closeC),
			feapiserver.InstallProxyHandler(cfg),
		); err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		close(joinC)
	}()

	// FIXME: wait until Serve to actually start accepting conns
	time.Sleep(100 * time.Millisecond)

	c, err := testutils.TLSHTTPClient(certFile)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	resp, err := c.Get("https://" + testFeListenAddr + "/proxy/hostfoo/a/b/c?query=param&foo=bar")
	if err != nil {
		t.Errorf("http.Get: %v", err)
		return
	}
	cont, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ReadAll(http.Get resp.Body): %v", err)
		return
	}
	if !bytes.Equal(cont, []byte("fuga\n")) {
		t.Errorf("unexpected content: %v", string(cont))
	}
	resp.Body.Close()

	buf := bytes.NewBuffer([]byte("body"))
	resp, err = c.Post("https://"+testFeListenAddr+"/proxy/hostfoo/post", "text/plain", buf)
	if err != nil {
		t.Errorf("http.Post: %v", err)
		return
	}
	cont, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("ReadAll(http.Post resp.Body): %v", err)
		return
	}
	if !bytes.Equal(cont, []byte("fuga\n")) {
		t.Errorf("unexpected content: %v", string(cont))
	}
	resp.Body.Close()

	close(closeC)
	<-joinC

	m.Terminate()

	if len(m.Reqs) != 2 {
		t.Errorf("lem(m.Reqs) %d", len(m.Reqs))
	}
	if m.Reqs[0].URL.String() != "/a/b/c?query=param&foo=bar" {
		t.Errorf("unexpected req url: %v", m.Reqs[0].URL.String())
	}
	if m.Reqs[0].Method != "GET" {
		t.Errorf("unexpected method %q", m.Reqs[0].Method)
	}
	if len(m.Reqs[0].Payload) != 0 {
		t.Errorf("unexpected payload %v", m.Reqs[0].Payload)
	}
	if m.Reqs[1].URL.String() != "/post" {
		t.Errorf("unexpected req url: %v", m.Reqs[1].URL.String())
	}
	if m.Reqs[1].Method != "POST" {
		t.Errorf("unexpected method %q", m.Reqs[1].Method)
	}
	if !bytes.Equal(m.Reqs[1].Payload, []byte("body")) {
		t.Errorf("unexpected payload %v", m.Reqs[1].Payload)
	}
}

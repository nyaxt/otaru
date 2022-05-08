package apiserver_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/cli"
	feapiserver "github.com/nyaxt/otaru/extra/fe/apiserver"
	"github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/testutils/testca"
)

const (
	testBeListenAddr = "localhost:20246"
	testFeListenAddr = "localhost:20249"
)

func init() {
	testutils.EnsureLogger()
}

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

func runMockBackend() *mockBackend {
	m := &mockBackend{
		closeC: make(chan struct{}),
		joinC:  make(chan struct{}),
		Reqs:   []mockRequest{},
	}

	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testBeListenAddr),
			apiserver.TLSCertKey(testca.Cert, testca.Key.Parsed),
			apiserver.ClientCACert(testca.ClientAuthCACert),
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

func (m *mockBackend) PopReqs() (reqs []mockRequest) {
	reqs = m.Reqs
	m.Reqs = []mockRequest{}
	return
}

func (m *mockBackend) Terminate() {
	close(m.closeC)
	<-m.joinC
}

const (
	basicuser     = "proxyuser"
	basicpassword = "proxypassword"
)

func TestProxyHandler(t *testing.T) {
	cfg := &cli.CliConfig{
		Host: map[string]*cli.Host{
			"hostfoo": {
				ApiEndpoint: testBeListenAddr,
				CACert:      testca.CACert,
				Cert:        testca.ClientAuthAdminCert,
				Key:         testca.ClientAuthAdminKey.Parsed,
			},
		},
	}

	m := runMockBackend()

	closeC := make(chan struct{})
	joinC := make(chan struct{})
	go func() {
		if err := apiserver.Serve(
			apiserver.ListenAddr(testFeListenAddr),
			apiserver.TLSCertKey(testca.Cert, testca.Key.Parsed),
			apiserver.CloseChannel(closeC),
			feapiserver.InstallProxyHandler(cfg, basicuser, basicpassword),
		); err != nil {
			t.Errorf("Serve failed: %v", err)
		}
		close(joinC)
	}()
	defer func() {
		close(closeC)
		<-joinC
		defer m.Terminate()
	}()

	// FIXME: wait until Serve to actually start accepting conns
	time.Sleep(100 * time.Millisecond)

	cp := x509.NewCertPool()
	cp.AddCert(testca.CACert)
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: cp,
			},
		},
	}

	t.Run("Get", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://"+testFeListenAddr+"/proxy/hostfoo/a/b/c?query=param&foo=bar", nil)
		if err != nil {
			t.Fatalf("http.NewRequest: %v", err)
		}
		req.SetBasicAuth(basicuser, basicpassword)

		resp, err := c.Do(req)
		if err != nil {
			t.Errorf("(http.Client).Do(Get): %v", err)
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

		reqs := m.PopReqs()
		if len(reqs) != 1 {
			t.Fatalf("lem(reqs) %d", len(reqs))
		}
		if reqs[0].URL.String() != "/a/b/c?query=param&foo=bar" {
			t.Errorf("unexpected req url: %v", reqs[0].URL.String())
		}
		if reqs[0].Method != "GET" {
			t.Errorf("unexpected method %q", reqs[0].Method)
		}
		if len(reqs[0].Payload) != 0 {
			t.Errorf("unexpected payload %v", reqs[0].Payload)
		}
	})

	t.Run("Post", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte("body"))
		req, err := http.NewRequest(http.MethodPost, "https://"+testFeListenAddr+"/proxy/hostfoo/post", buf)
		if err != nil {
			t.Fatalf("http.NewRequest: %v", err)
		}
		req.SetBasicAuth(basicuser, basicpassword)
		req.Header.Set("Content-Type", "text/plain")

		resp, err := c.Do(req)
		if err != nil {
			t.Errorf("(http.Client).Do(Post): %v", err)
			return
		}
		cont, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("ReadAll(http.Post resp.Body): %v", err)
			return
		}
		if !bytes.Equal(cont, []byte("fuga\n")) {
			t.Errorf("unexpected content: %v", string(cont))
		}
		resp.Body.Close()

		reqs := m.PopReqs()
		if len(reqs) != 1 {
			t.Fatalf("lem(reqs) %d", len(reqs))
		}
		if reqs[0].URL.String() != "/post" {
			t.Errorf("unexpected req url: %v", reqs[0].URL.String())
		}
		if reqs[0].Method != "POST" {
			t.Errorf("unexpected method %q", reqs[0].Method)
		}
		if !bytes.Equal(reqs[0].Payload, []byte("body")) {
			t.Errorf("unexpected payload %v", reqs[0].Payload)
		}
	})
}

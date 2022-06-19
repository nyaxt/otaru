package apiserver

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/basicauth"
	"github.com/nyaxt/otaru/cli"
	"go.uber.org/zap"
)

type apiproxy struct {
	cfg *cli.CliConfig
}

var reMatch = regexp.MustCompile(`^/proxy/(\w+)(/.*)$`)

func copyHeader(d, s http.Header) {
	for k, v := range s {
		d[k] = v
	}
}

func (ap *apiproxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	ms := reMatch.FindStringSubmatch(path)
	if len(ms) != 3 {
		http.Error(w, "Invalid proxy URL.", http.StatusBadRequest)
		return
	}
	// fmt.Printf("path %v match %v", path, ms)

	hname, tgtpath := ms[1], ms[2]

	ci, err := cli.QueryConnectionInfo(ap.cfg, hname)
	if err != nil {
		http.Error(w, "Failed to construct connection info.", http.StatusInternalServerError)
		return
	}

	hcli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: ci.TLSConfig,
		},
	}
	zap.S().Debugf("tc.RootCAs: %+v", ci.TLSConfig.RootCAs)
	url := &url.URL{
		Scheme:   "https",
		Host:     ci.ApiEndpoint,
		Path:     tgtpath,
		RawQuery: r.URL.RawQuery,
	}
	zap.S().Debugf("URL: %v", url)

	// FIXME: filter tgtpath
	// FIXME: add forwarded-for

	preq := &http.Request{
		Method: r.Method,
		Header: make(http.Header),
		URL:    url,
		Body:   r.Body,
	}
	copyHeader(preq.Header, r.Header)

	presp, err := hcli.Do(preq)
	if err != nil {
		zap.S().Warnf("Failed to issue request: %v", err)
		http.Error(w, "Failed to issue request.", http.StatusInternalServerError)
		return
	}
	defer presp.Body.Close()

	// zap.S().Debugf("resp st: %d", presp.StatusCode)

	copyHeader(w.Header(), presp.Header)
	w.WriteHeader(presp.StatusCode)
	io.Copy(w, presp.Body)
}

func InstallProxyHandler(cfg *cli.CliConfig, basicuser, basicpassword string) apiserver.Option {
	return apiserver.AddMuxHook(func(_ context.Context, mux *http.ServeMux) error {
		mux.Handle("/proxy/", basicauth.Handler{
			User:     basicuser,
			Password: basicpassword,
			Handler: &apiproxy{
				cfg: cfg,
			},
		})
		return nil
	})
}

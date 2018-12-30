package apiserver

import (
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/nyaxt/otaru/apiserver"
	"github.com/nyaxt/otaru/apiserver/jwt"
	"github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/logger"
)

type apiproxy struct {
	cfg     *cli.CliConfig
	jwtauth *jwt.JWTAuthProvider
}

var reMatch = regexp.MustCompile(`^/proxy/(\w+)(/.*)$`)

func copyHeader(d, s http.Header) {
	for k, v := range s {
		d[k] = v
	}
}

func (ap *apiproxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	ui, err := ap.jwtauth.UserInfoFromAuthHeader(auth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if ui.Role < jwt.RoleReadOnly {
		http.Error(w, "", http.StatusForbidden)
		return
	}

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
	url := &url.URL{
		Scheme:   "https",
		Host:     ci.ApiEndpoint,
		Path:     tgtpath,
		RawQuery: r.URL.RawQuery,
	}
	logger.Debugf(mylog, "URL: %v", url)

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
		logger.Warningf(mylog, "Failed to issue request: %v", err)
		http.Error(w, "Failed to issue request.", http.StatusInternalServerError)
		return
	}
	defer presp.Body.Close()

	// logger.Debugf(mylog, "resp st: %d", presp.StatusCode)

	copyHeader(w.Header(), presp.Header)
	w.WriteHeader(presp.StatusCode)
	io.Copy(w, presp.Body)
}

func InstallProxyHandler(cfg *cli.CliConfig, jwtauth *jwt.JWTAuthProvider) apiserver.Option {
	return apiserver.AddMuxHook(func(mux *http.ServeMux) {
		mux.Handle("/proxy/", &apiproxy{cfg, jwtauth})
	})
}

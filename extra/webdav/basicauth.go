package webdav

import (
	"net/http"

	"github.com/nyaxt/otaru/logger"
)

type BasicAuthHandler struct {
	User     string
	Password string
	Handler  http.Handler
}

func (h BasicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, password, ok := r.BasicAuth()
	if !ok || user != h.User || password != h.Password {
		logger.Infof(mylog, "Auth failed. User: %q Password: %q Header: %+v", user, password, r.Header)

		w.Header().Set("WWW-Authenticate", "Basic realm=\"otaru-webdav-fe\"")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Authentication Required."))
		return
	}

	h.Handler.ServeHTTP(w, r)
}

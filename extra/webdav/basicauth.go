package webdav

import (
	"net/http"
)

type BasicAuthHandler struct {
	User     string
	Password string
	Handler  http.Handler
}

func (h BasicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, password, ok := r.BasicAuth()
	if !ok || user != h.User || password != h.Password {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"otaru-webdav-fe\"")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Authentication Required."))
		return
	}

	h.Handler.ServeHTTP(w, r)
}

package basicauth

import (
	"net/http"
)

type Handler struct {
	User     string
	Password string
	Handler  http.Handler
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, password, ok := r.BasicAuth()
	if !ok || user != h.User || password != h.Password {
		// zap.S().Infof("Authentication failed. User: %q Password: %q Header: %+v", user, password, r.Header)

		w.Header().Set("WWW-Authenticate", "Basic realm=\"otaru\"")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Authentication Required."))
		return
	}

	h.Handler.ServeHTTP(w, r)
}

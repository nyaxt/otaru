package logger

import (
	"net/http"

	"go.uber.org/zap"
)

func HttpHandler(l *zap.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.Info(
			"http request",
			zap.String("uri", r.URL.RequestURI()),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.String("referer", r.Referer()),
			zap.String("user_agent", r.UserAgent()),
		)

		h.ServeHTTP(w, r)
	})
}

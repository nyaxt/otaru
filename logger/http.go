package logger

import (
	"fmt"
	"net/http"
	"time"
)

func HttpHandler(logger Logger, level Level, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logger.WillAccept(level) {
			uri := r.URL.RequestURI()
			logstr := fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, uri)
			logger.Log(level, map[string]interface{}{
				"log":         logstr,
				"time":        time.Now(),
				"location":    "httplogger.go:1",
				"remote_addr": r.RemoteAddr,
				"method":      r.Method,
				"uri":         uri,
				"referer":     r.Referer(),
				"user_agent":  r.UserAgent(),
			})
		}

		h.ServeHTTP(w, r)
	})
}

package apiserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nyaxt/otaru/logger"
)

var httplog = logger.Registry().Category("http")

const httplogLevel = logger.Info

func httpLoggerHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httplog.WillAccept(httplogLevel) {
			uri := r.URL.RequestURI()
			logstr := fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, uri)
			httplog.Log(httplogLevel, map[string]interface{}{
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

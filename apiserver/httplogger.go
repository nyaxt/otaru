package apiserver

import (
	"net/http"

	"github.com/nyaxt/otaru/logger"
)

var httplog = logger.Registry().Category("http-apiserver")

func httpLoggerHandler(h http.Handler) http.Handler {
	return logger.HttpHandler(httplog, logger.Info, h)
}

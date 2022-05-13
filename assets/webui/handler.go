package webui

import (
	"embed"
	"net/http"
)

//go:embed dist/*
var fs embed.FS

var Handler http.Handler

func init() {
	fs := http.FileServer(http.FS(fs))
	Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/dist" + r.URL.Path
		fs.ServeHTTP(w, r)
	})
}

func WebUIHandler(override, indexpath string) http.Handler {
	handler := Handler
	if override != "" {
		handler = http.FileServer(http.Dir(override))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = indexpath
		}
		handler.ServeHTTP(w, r)
	})
}

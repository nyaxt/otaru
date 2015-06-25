package mgmt

import (
	"encoding/json"
	"net/http"
)

type GenericHandlerFunc func(*http.Request) interface{}

func JSONHandler(genericHandler GenericHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		res := genericHandler(req)
		if err, ok := res.(error); ok {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		b, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "json")
		w.Write(b)
	}
}

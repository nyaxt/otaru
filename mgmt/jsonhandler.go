package mgmt

import (
	"encoding/json"
	"net/http"
)

type GenericHandlerFunc func(*http.Request) interface{}

func JSONHandler(genericHandler GenericHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		res := genericHandler(req)
		b, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "json")
		w.Write(b)
	}
}

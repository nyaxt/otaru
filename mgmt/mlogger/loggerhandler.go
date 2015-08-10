package mlogger

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/mgmt"
)

func Install(srv *mgmt.Server) {
	rtr := srv.APIRouter().PathPrefix("/logger").Subrouter()

	rtr.HandleFunc("/categories", mgmt.JSONHandler(func(req *http.Request) interface{} {
		return logger.Registry().Categories()
	}))
	rtr.HandleFunc("/category/{cat:\\w+}", mgmt.JSONHandler(func(req *http.Request) interface{} {
		vars := mux.Vars(req)
		cat := vars["cat"]
		cl := logger.Registry().Category(cat)
		if cl == nil {
			return fmt.Errorf("Cateogory not found")
		}
		if req.Method == "GET" {
			return cl.View()
		} else if req.Method == "POST" {
			levelp := req.URL.Query().Get("level")
			nlevel, err := strconv.ParseUint(levelp, 10, 32)
			if err != nil {
				return fmt.Errorf("Failed to parse level")
			}
			cl.Level = logger.Level(nlevel)

			return cl.View()
		} else {
			return fmt.Errorf("Unknown method!")
		}
	}))
}

package main

import (
	"github.com/nyaxt/otaru/mgmt"
	"github.com/nyaxt/otaru/mgmt/mblobstore"
)

func main() {
	srv := mgmt.NewServer()
	mblobstore.Install(srv)
	srv.Run()
}

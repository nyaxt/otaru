package main

import (
	"github.com/nyaxt/otaru/mgmt"
)

func main() {
	srv := mgmt.NewServer()
	srv.Run()
}

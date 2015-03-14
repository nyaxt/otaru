package main

import (
	"flag"
	"fmt"
	"github.com/nyaxt/otaru"
)

func main() {
	flag.Parse()
	bs := otaru.NewGCSBlobStore()
	bs.Put()

	fmt.Printf("hi\n")
}

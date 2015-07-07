package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/util"
)

var (
	flagPasswordFile = flag.String("passwordFile", path.Join(os.Getenv("HOME"), ".otaru", "password.txt"), "Path of a text file storing password")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s OTARU_BLOBFILE\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	flag.Usage = Usage
	flag.Parse()

	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	filepath := flag.Arg(0)

	f, err := os.Open(filepath)
	if err != nil {
		log.Printf("Failed to read file: %s", filepath)
	}
	bh := blobstore.FileBlobHandle{f}
	defer bh.Close()

	password := util.StringFromFileOrDie(*flagPasswordFile, "password")
	key := btncrypt.KeyFromPassword(password)
	c, err := btncrypt.NewCipher(key)
	if err != nil {
		log.Printf("Failed to init Cipher: %v", err)
		return
	}

	cio := otaru.NewChunkIO(bh, c)
	defer cio.Close()
	log.Printf("Header: %v", cio.Header())
}

package main

import (
	"flag"
	"log"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/util"
)

var (
	flagPasswordFile = flag.String("passwordFile", path.Join(os.Getenv("HOME"), ".otaru", "password.txt"), "Path of a text file storing password")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	flag.Usage = Usage
	flag.Parse()

	password := util.StringFromFileOrDie(*flagPasswordFile, "password")
	c, err = btncrypt.NewCipher(key)
	if err != nil {
		log.Printf("Failed to init Cipher: %v", err)
		return
	}

	cio := otaru.NewChunkIO(bh, c)
}

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("otaru-dumpblob")

var (
	flagPasswordFile = flag.String("passwordFile", path.Join(facade.DefaultConfigDir(), "password.txt"), "Path of a text file storing password")
	flagHeader       = flag.Bool("header", false, "Show header")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s OTARU_BLOBFILE\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	logger.Registry().AddOutput(logger.HandleCritical(func() { os.Exit(1) }))

	flag.Usage = Usage
	flag.Parse()

	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	filepath := flag.Arg(0)

	f, err := os.Open(filepath)
	if err != nil {
		logger.Infof(mylog, "Failed to read file: %s", filepath)
	}
	defer f.Close()

	password := util.StringFromFileOrDie(*flagPasswordFile, "password")
	key := btncrypt.KeyFromPassword(password)
	c, err := btncrypt.NewCipher(key)
	if err != nil {
		logger.Infof(mylog, "Failed to init Cipher: %v", err)
		return
	}

	cr, err := chunkstore.NewChunkReader(f, c)
	if err != nil {
		logger.Infof(mylog, "Failed to init ChunkReader: %v", err)
		return
	}
	defer cr.Close()

	if *flagHeader {
		logger.Infof(mylog, "Header: %+v", cr.Header())
	}
	io.Copy(os.Stdout, cr)
}

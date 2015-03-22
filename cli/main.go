package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nyaxt/otaru"
)

func help() {
	flag.PrintDefaults()

	os.Exit(1)
}

func put(fromurl, tourl string) error {
	fromfile, err := os.Open(fromurl)
	if err != nil {
		return fmt.Errorf("File open failed: %v", err)
	}
	defer fromfile.Close()

	fromstat, err := fromfile.Stat()
	if err != nil {
		return fmt.Errorf("File stat failed: %v", err)
	}
	// FIXME: handle unsized file / non-file (e.g. pipe)
	fromsize := fromstat.Size()

	bs := &otaru.FileBlobStore{}
	w, err := bs.OpenWriter("testblob.dat")
	if err != nil {
		return err
	}

	cw := otaru.NewChunkWriter(w, []byte("0123456789abcdef"))
	defer cw.Close()

	// FIXME: split file into multiple chunks
	cw.WriteHeader(&otaru.ChunkHeader{
		PayloadLen:   fromsize,
		OrigFilename: fromurl,
		OrigOffset:   0,
	})

	buf := make([]byte, otaru.BtnFrameMaxPayload)
	nr, err := fromfile.Read(buf)
	if err != nil {
		return fmt.Errorf("Failed to read: %v", err)
	}

	nw, err := cw.Write(buf[:nr])
	if err != nil {
		return fmt.Errorf("Failed to write: %v", err)
	}
	if nw != nr {
		return fmt.Errorf("Incomplete write")
	}

	return nil
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) == 1 {
		// get cmd
		log.Printf("Get: %v", args[0])
	} else if len(args) == 2 {
		log.Printf("Put: %v -> %v", args[0], args[1])
		if err := put(args[0], args[1]); err != nil {
			log.Fatalf("Put failed: %v", err)
		}
	} else {
		log.Printf("Invalid number of args")
		help()
	}
}

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/flags"
)

var (
	Key = []byte("0123456789abcdef")
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

	bs, err := blobstore.NewFileBlobStore(".", flags.O_RDWR)
	if err != nil {
		return err
	}
	w, err := bs.OpenWriter("testblob.dat")
	if err != nil {
		return err
	}

	c, err := otaru.NewCipher(Key)
	if err != nil {
		return err
	}

	cw, err := otaru.NewChunkWriter(w, c,
		otaru.ChunkHeader{
			OrigFilename: fromurl,
			OrigOffset:   0,
			PayloadLen:   uint32(fromsize),
		},
	)
	if err != nil {
		return err
	}
	defer cw.Close()

	buf := make([]byte, otaru.BtnFrameMaxPayload)
	for {
		nr, err := fromfile.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("Failed to read: %v", err)
		}

		nw, err := cw.Write(buf[:nr])
		if err != nil {
			return fmt.Errorf("Failed to write: %v", err)
		}
		if nw != nr {
			return fmt.Errorf("Incomplete write")
		}
	}

	if err := cw.Close(); err != nil {
		return err
	}

	return nil
}

func get(fromurl string) error {
	bs, err := blobstore.NewFileBlobStore(".", flags.O_RDWR)
	if err != nil {
		return err
	}
	r, err := bs.OpenReader("testblob.dat")
	if err != nil {
		return fmt.Errorf("Blob open failed: %v", err)
	}

	c, err := otaru.NewCipher(Key)
	if err != nil {
		return err
	}

	cr, err := otaru.NewChunkReader(r, c)
	if err != nil {
		return err
	}
	buf := make([]byte, otaru.BtnFrameMaxPayload)
	unreadLen := cr.Length()
	for unreadLen > 0 {
		nr, err := cr.Read(buf)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("Unexpected EOF reached.")
			}
			return fmt.Errorf("Failed to read chunk content: %v", err)
		}
		unreadLen -= nr

		nw, err := os.Stdout.Write(buf[:nr])
		if err != nil {
			return err
		}
		if nw != nr {
			return fmt.Errorf("Unexpected partial write")
		}
	}

	return nil
}

func main() {
	flag.Parse()
	log.SetOutput(os.Stderr)

	args := flag.Args()
	if len(args) == 1 {
		// get cmd
		log.Printf("Get: %v", args[0])
		if err := get(args[0]); err != nil {
			log.Fatalf("Get err: %v", err)
		}
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

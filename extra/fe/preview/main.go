package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"log"
	"os"
)

var flagInput = flag.String("i", "", "input zip")

func run() error {
	if *flagInput == "" {
		return fmt.Errorf("input file not given.")
	}

	f, err := os.Open(*flagInput)
	if err != nil {
		return fmt.Errorf("Failed to open file %q. err: %v", *flagInput, err)
	}
	defer f.Close()

	zipsize, err := f.Seek(0, 2)
	if err != nil {
		return fmt.Errorf("Failed to seek eof: %v", err)
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("Failed to seek start: %v", err)
	}
	log.Printf("zip size: %v", zipsize)

	z, err := zip.NewReader(f, zipsize)
	if err != nil {
		return fmt.Errorf("Failed to open zip reader: %v", err)
	}
	for _, e := range z.File {
		log.Printf("FileName: %v", e.Name)
	}

	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Printf("err: %v", err)
	}
}

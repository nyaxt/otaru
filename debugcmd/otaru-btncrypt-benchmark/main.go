package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	tu "github.com/nyaxt/otaru/testutils"
	"os"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("otaru-btncrypt-benchmark")

var (
	flagSize = flag.String("size", "100MB", "Test target blob size")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	logger.Registry().AddOutput(logger.HandleCritical(func() { os.Exit(1) }))

	flag.Usage = Usage
	flag.Parse()

	size, err := humanize.ParseBytes(*flagSize)
	if err != nil {
		logger.Criticalf(mylog, "Failed to parse size: %s", *flagSize)
	}
	logger.Infof(mylog, "Target blob size: %d", size)
	tstart := time.Now()
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		logger.Criticalf(mylog, "Failed to generate random seq: %v", err)
	}
	logger.Infof(mylog, "Preparing target took: %v", time.Since(tstart))

	var envelope []byte
	{
		tstart = time.Now()
		envelope, err = btncrypt.Encrypt(tu.TestCipher(), buf)
		elapsed := time.Since(tstart)

		if err != nil {
			logger.Criticalf(mylog, "Failed to encrypt: %v", err)
		}
		logger.Infof(mylog, "Encrypt took %v", elapsed)
		sizeMB := (float64)(size) / (1000 * 1000)
		MBps := sizeMB / elapsed.Seconds()
		Mbps := MBps * 8
		logger.Infof(mylog, "%v MB/s %v Mbps", MBps, Mbps)
	}

	{
		tstart = time.Now()
		_, err = btncrypt.Decrypt(tu.TestCipher(), envelope, len(buf))
		elapsed := time.Since(tstart)

		if err != nil {
			logger.Criticalf(mylog, "Failed to decrypt: %v", err)
		}
		logger.Infof(mylog, "Decrypt took %v", elapsed)
		sizeMB := (float64)(size) / (1000 * 1000)
		MBps := sizeMB / elapsed.Seconds()
		Mbps := MBps * 8
		logger.Infof(mylog, "%v MB/s %v Mbps", MBps, Mbps)
	}
}

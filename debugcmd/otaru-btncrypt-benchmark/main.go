package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"time"

	tu "github.com/nyaxt/otaru/testutils"
	"go.uber.org/zap"

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
		zap.S().Errorf("Failed to parse size: %s", *flagSize)
	}
	zap.S().Infof("Target blob size: %d", size)
	tstart := time.Now()
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		zap.S().Errorf("Failed to generate random seq: %v", err)
	}
	zap.S().Infof("Preparing target took: %v", time.Since(tstart))

	var envelope []byte
	{
		tstart = time.Now()
		envelope, err = btncrypt.Encrypt(tu.TestCipher(), buf)
		elapsed := time.Since(tstart)

		if err != nil {
			zap.S().Errorf("Failed to encrypt: %v", err)
		}
		zap.S().Infof("Encrypt took %v", elapsed)
		sizeMB := (float64)(size) / (1000 * 1000)
		MBps := sizeMB / elapsed.Seconds()
		Mbps := MBps * 8
		zap.S().Infof("%v MB/s %v Mbps", MBps, Mbps)
	}

	{
		tstart = time.Now()
		_, err = btncrypt.Decrypt(tu.TestCipher(), envelope, len(buf))
		elapsed := time.Since(tstart)

		if err != nil {
			zap.S().Errorf("Failed to decrypt: %v", err)
		}
		zap.S().Infof("Decrypt took %v", elapsed)
		sizeMB := (float64)(size) / (1000 * 1000)
		MBps := sizeMB / elapsed.Seconds()
		Mbps := MBps * 8
		zap.S().Infof("%v MB/s %v Mbps", MBps, Mbps)
	}
}

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("otaru-txlogio")

var (
	flagConfigDir = flag.String("configDir", facade.DefaultConfigDir(), "Config dirpath")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] purge\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [options] query [minID]\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	logger.Registry().AddOutput(logger.WriterLogger{os.Stderr})
	logger.Registry().AddOutput(logger.HandleCritical(func() { os.Exit(1) }))

	flag.Usage = Usage
	flag.Parse()

	cfg, err := facade.NewConfig(*flagConfigDir)
	if err != nil {
		logger.Infof(mylog, "%v", err)
		Usage()
		os.Exit(2)
	}
	minID := inodedb.LatestVersion
	if flag.NArg() < 1 {
		Usage()
		os.Exit(2)
	}
	switch flag.Arg(0) {
	case "purge":
		if flag.NArg() != 1 {
			Usage()
			os.Exit(2)
		}
	case "query":
		switch flag.NArg() {
		case 1:
			break
		case 2:
			n, err := strconv.ParseInt(flag.Arg(1), 10, 64)
			if err != nil {
				Usage()
				os.Exit(2)
			}
			minID = inodedb.TxID(n)
			break
		}
		break
	default:
		logger.Infof(mylog, "Unknown cmd: %v", flag.Arg(0))
		Usage()
		os.Exit(2)
	}

	tsrc, err := auth.GetGCloudTokenSource(context.Background(), cfg.CredentialsFilePath, cfg.TokenCacheFilePath, false)
	if err != nil {
		logger.Criticalf(mylog, "Failed to init GCloudClientSource: %v", err)
	}

	key := btncrypt.KeyFromPassword(cfg.Password)
	c, err := btncrypt.NewCipher(key)
	if err != nil {
		logger.Criticalf(mylog, "Failed to init *btncrypt.Cipher: %v", err)
	}
	dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, c, tsrc)

	txlogio := datastore.NewDBTransactionLogIO(dscfg)

	switch flag.Arg(0) {
	case "purge":
		if err := txlogio.DeleteAllTransactions(); err != nil {
			logger.Infof(mylog, "DeleteAllTransactions() failed: %v", err)
		}

	case "query":
		logger.Infof(mylog, "Start QueryTransactions(%v)", minID)
		txs, err := txlogio.QueryTransactions(minID)
		if err != nil {
			logger.Infof(mylog, "QueryTransactions() failed: %v", err)
		}
		for _, tx := range txs {
			fmt.Printf("%s\n", tx)
		}

	default:
		logger.Infof(mylog, "Unknown cmd: %v", flag.Arg(0))
		os.Exit(1)
	}
}

package dumpblob

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/util"
)

var Command = &cli.Command{
	Name:      "dumpblob",
	Usage:     "dump a chunkstore blob file content",
	ArgsUsage: "OTARU_BLOBFILE",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "header",
			Usage: "Show header",
		},
		&cli.PathFlag{
			Name:  "passwordFile",
			Value: path.Join(facade.DefaultConfigDir(), "password.txt"),
			Usage: "Path to a text file storing password",
		},
	},
	Action: func(c *cli.Context) error {
		s := zap.S().Named("dumpblob")
		if !c.Args().Present() {
			return fmt.Errorf("No file path specified on the commandline argument")
		}

		filepath := c.Args().First()

		f, err := os.Open(filepath)
		if err != nil {
			return fmt.Errorf("Failed to read file %q: %w", filepath, err)
		}
		defer f.Close()

		password := util.StringFromFileOrDie(c.Path("passwordFile"), "password")
		key := btncrypt.KeyFromPassword(password)
		cipher, err := btncrypt.NewCipher(key)
		if err != nil {
			return fmt.Errorf("Failed to init Cipher: %w", err)
		}

		cr, err := chunkstore.NewChunkReader(f, cipher)
		if err != nil {
			return fmt.Errorf("Failed to init ChunkReader: %v", err)
		}
		defer cr.Close()

		if c.Bool("header") {
			s.Infof("Header: %+v", cr.Header())
		}

		_, err = io.Copy(os.Stdout, cr)
		return err
	},
}

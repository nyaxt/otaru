package globallock

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/gcloud/auth"
	"github.com/nyaxt/otaru/gcloud/datastore"
)

type Action int

const (
	QueryAction Action = iota
	LockAction
	UnlockAction
)

var Command = &cli.Command{
	Name:      "globallock",
	Usage:     "Inspect or modify otaru global lock",
	ArgsUsage: "{lock,unlock,query}",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Usage: "Force Unlock currently active global lock when specified with unlock cmd",
		},
		&cli.StringFlag{
			Name:  "info",
			Usage: "Custom info string",
		},
	},
	Action: func(c *cli.Context) error {
		cfg, err := facade.NewConfig(c.Path("configDir"))
		if err != nil {
			return err
		}

		a := QueryAction
		if c.Args().Present() {
			actionstr := c.Args().First()
			switch actionstr {
			case "query":
				a = QueryAction
			case "lock":
				a = LockAction
			case "unlock":
				a = UnlockAction
			default:
				return fmt.Errorf("Unknown action %q", actionstr)
			}
		}

		tsrc, err := auth.GetGCloudTokenSource(cfg.CredentialsFilePath)
		if err != nil {
			return fmt.Errorf("Failed to init GCloudClientSource: %v", err)
		}

		nullCipher := &btncrypt.Cipher{} // Null cipher is fine, as GlobalLocker doesn't make use of it.
		dscfg := datastore.NewConfig(cfg.ProjectName, cfg.BucketName, nullCipher, tsrc)
		info := c.String("info")
		if info == "" {
			info = "otaru-globallock-cli cmdline debug tool"
		}
		l := datastore.NewGlobalLocker(dscfg, "otaru-globallock-cli", info)

		ctx := c.Context

		switch a {
		case LockAction:
			readOnly := false
			if err := l.Lock(ctx, readOnly); err != nil {
				return fmt.Errorf("Lock failed: %w", err)
			}
		case UnlockAction:
			if c.Bool("force") {
				if err := l.ForceUnlock(ctx); err != nil {
					return fmt.Errorf("ForceUnlock failed: %w", err)
				}
			} else {
				if err := l.UnlockIgnoreCreatedAt(ctx); err != nil {
					return fmt.Errorf("Unlock failed: %w", err)
				}
			}
		case QueryAction:
			entry, err := l.Query(ctx)
			if err != nil {
				return fmt.Errorf("Query failed: %w", err)
			}
			fmt.Printf("%+v\n", entry)

		default:
			panic("should not be reached")
		}
		return nil
	},
}

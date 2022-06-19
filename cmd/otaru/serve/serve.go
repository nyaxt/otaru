package serve

import (
	"github.com/urfave/cli/v2"

	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/logger"
)

var mylog = logger.Registry().Category("otaru-server")

var Command = &cli.Command{
	Name:  "serve",
	Usage: "Run otaru gRPC server",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "readonly",
			Usage: "Mount as read-only mode. No changes to the filesystem is allowed.",
		},
	},
	Action: func(c *cli.Context) error {
		cfg, err := facade.NewConfig(c.Path("configDir"))
		if err != nil {
			return err
		}
		if c.Bool("readonly") {
			cfg.ReadOnly = true
		}

		if err := facade.Serve(c.Context, cfg); err != nil {
			return err
		}

		return nil
	},
}

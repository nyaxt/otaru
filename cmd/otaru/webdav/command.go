package webdav

import (
	"github.com/urfave/cli/v2"

	ocli "github.com/nyaxt/otaru/cli"
	"github.com/nyaxt/otaru/webdav"
)

var Command = &cli.Command{
	Name:  "webdav",
	Usage: "Run otaru webdav server",
	Action: func(c *cli.Context) error {
		cfg, err := ocli.NewConfig(c.String("configDir"))
		if err != nil {
			return err
		}

		if err = webdav.Serve(c.Context, cfg); err != nil {
			return err
		}
		return nil
	},
}

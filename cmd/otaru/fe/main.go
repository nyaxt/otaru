package fe

import (
	"github.com/urfave/cli/v2"

	"github.com/nyaxt/otaru/apiserver"
	ocli "github.com/nyaxt/otaru/cli"
	fe_apiserver "github.com/nyaxt/otaru/extra/fe/apiserver"
)

var Command = &cli.Command{
	Name:    "frontend",
	Aliases: []string{"fe"},
	Usage:   "Run otaru frontend web server",
	Action: func(c *cli.Context) error {
		cfg, err := ocli.NewConfig(c.Path("configDir"))
		if err != nil {
			return err
		}

		opts, err := fe_apiserver.BuildApiServerOptions(cfg)
		if err != nil {
			return err
		}
		return apiserver.Serve(c.Context, opts...)
	},
}

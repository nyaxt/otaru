package fscli

import (
	"os"

	"github.com/urfave/cli/v2"

	ocli "github.com/nyaxt/otaru/cli"
)

var Commands = []*cli.Command{
	{
		Name:      "ls",
		Aliases:   []string{"list"},
		ArgsUsage: "otaru://vhost/path",
		Action: func(c *cli.Context) error {
			cfg, err := ocli.NewConfig(c.String("configDir"))
			if err != nil {
				return err
			}

			if err := ocli.Ls(c.Context, os.Stdout, cfg, c.Args().Slice()); err != nil {
				return err
			}

			return nil
		},
	},
	{
		Name: "attr",
		Action: func(c *cli.Context) error {
			cfg, err := ocli.NewConfig(c.String("configDir"))
			if err != nil {
				return err
			}

			if err := ocli.Attr(c.Context, cfg, c.Args().Slice()); err != nil {
				return err
			}

			return nil
		},
	},
	{
		Name: "get",
		Action: func(c *cli.Context) error {
			cfg, err := ocli.NewConfig(c.String("configDir"))
			if err != nil {
				return err
			}

			if err := ocli.Get(c.Context, cfg, c.Args().Slice()); err != nil {
				return err
			}

			return nil
		},
	},
	{
		Name: "put",
		Action: func(c *cli.Context) error {
			cfg, err := ocli.NewConfig(c.String("configDir"))
			if err != nil {
				return err
			}

			if err := ocli.Put(c.Context, cfg, c.Args().Slice()); err != nil {
				return err
			}

			return nil
		},
	},
}

package mkfs

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/nyaxt/otaru/facade"
)

var Command = &cli.Command{
	Name:  "mkfs",
	Usage: "Prep new otaru filesystem instance.",
	Action: func(c *cli.Context) error {
		cfg, err := facade.NewConfig(c.Path("configDir"))
		if err != nil {
			return err
		}

		if err := facade.Mkfs(cfg); err != nil {
			return fmt.Errorf("facade.Mkfs: %w", err)
		}
		zap.S().Infof("mkfs finished successfully!")

		return nil
	},
}

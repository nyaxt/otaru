package serve

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

		ctx, cancel := context.WithCancel(c.Context)

		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, os.Interrupt)
		signal.Notify(sigC, syscall.SIGTERM)
		go func() {
			for s := range sigC {
				logger.Warningf(mylog, "Received signal: %v", s)
				cancel()
			}
		}()
		logger.Registry().AddOutput(logger.HandleCritical(func() {
			logger.Warningf(mylog, "Starting shutdown due to critical event.")
			cancel()
		}))

		if err := facade.Serve(ctx, cfg); err != nil {
			return err
		}

		return nil
	},
}

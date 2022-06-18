package main

import (
	"io"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nyaxt/otaru/cmd/otaru/fscli"
	"github.com/nyaxt/otaru/cmd/otaru/serve"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/version"
)

func simpleTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("15:04:05.999"))
}

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = "otaru"
	app.Usage = "cloud-backed filesystem for archiving your files"
	app.Authors = []*cli.Author{
		{Name: "nyaxt", Email: "ueno _at_ nyaxtstep.com"},
	}
	app.Version = version.BuildVersion
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose logging",
		},
		&cli.PathFlag{
			Name:    "configDir",
			Value:   facade.DefaultConfigDir(),
			Usage:   "Config dirpath",
			EnvVars: []string{"OTARUDIR"},
		},
	}
	app.Commands = []*cli.Command{
		serve.Command,
	}
	app.Commands = append(app.Commands, fscli.Commands...)
	BeforeImpl := func(c *cli.Context) error {
		var logger *zap.Logger
		if loggeri, ok := app.Metadata["Logger"]; ok {
			logger = loggeri.(*zap.Logger)
		} else {
			cfg := zap.NewProductionConfig()
			cfg.Encoding = "console"
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			cfg.EncoderConfig.EncodeTime = simpleTimeEncoder

			if c.Bool("verbose") {
				cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
			}

			var err error
			logger, err = cfg.Build(
				zap.AddStacktrace(zap.NewAtomicLevelAt(zap.DPanicLevel)))
			if err != nil {
				return err
			}
		}
		zap.ReplaceGlobals(logger)

		return nil
	}
	app.Before = func(c *cli.Context) error {
		if err := BeforeImpl(c); err != nil {
			// Print error message to stderr
			app.Writer = app.ErrWriter

			// Suppress help message on app.Before() failure.
			cli.HelpPrinter = func(_ io.Writer, _ string, _ interface{}) {}
			return err
		}

		return nil
	}
	app.After = func(c *cli.Context) error {
		zap.L().Sync()
		return nil
	}

	return app
}

package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nyaxt/otaru/cmd/otaru/deleteallblobs"
	"github.com/nyaxt/otaru/cmd/otaru/fscli"
	"github.com/nyaxt/otaru/cmd/otaru/serve"
	"github.com/nyaxt/otaru/cmd/otaru/webdav"
	"github.com/nyaxt/otaru/facade"
	"github.com/nyaxt/otaru/version"
)

func simpleTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("15:04:05.999"))
}

type cancelContextHook struct {
	cancel context.CancelFunc
}

var _ zapcore.CheckWriteHook = cancelContextHook{}

func (h cancelContextHook) OnWrite(*zapcore.CheckedEntry, []zap.Field) {
	h.cancel()
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
		webdav.Command,
		deleteallblobs.Command,
	}
	app.Commands = append(app.Commands, fscli.Commands...)

	BeforeImpl := func(c *cli.Context) error {
		newContext, cancel := context.WithCancel(c.Context)
		c.Context = newContext

		facade.BootstrapLogger()

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
				zap.AddStacktrace(zap.NewAtomicLevelAt(zap.DPanicLevel)),
				zap.WithFatalHook(cancelContextHook{cancel: cancel}))
			if err != nil {
				return err
			}
		}
		zap.ReplaceGlobals(logger)

		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, os.Interrupt)
		signal.Notify(sigC, syscall.SIGTERM)
		go func() {
			for s := range sigC {
				logger.Warn("Received signal", zap.String("signal", s.String()))
				cancel()
			}
		}()

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
		_ = zap.L().Sync()
		return nil
	}

	return app
}

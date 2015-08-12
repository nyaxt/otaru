package facade

import (
	gfluent "github.com/fluent/fluent-logger-golang/fluent"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/logger/fluent"
)

func SetupFluentLogger(cfg *Config) error {
	if cfg.Fluent.FluentHost == "" {
		logger.Infof(mylog, "The fluentd host is not specified. Skipping fluent logger instantiation.")
		return nil
	}
	logger.Infof(mylog, "Initializing fluent logger based on config: %+v", cfg.Fluent)

	fcli, err := gfluent.New(cfg.Fluent)
	if err != nil {
		return err
	}

	logger.Registry().AddOutput(fluent.FluentLogger{fcli})
	return nil
}

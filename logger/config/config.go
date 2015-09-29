package config

import (
	"fmt"
	"strings"

	"github.com/nyaxt/otaru/logger"
)

type Config struct {
	LogLevel map[string]string
}

func Str2Level(s string) (logger.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return logger.Debug, nil
	case "info":
		return logger.Info, nil
	case "warn", "warning":
		return logger.Warning, nil
	case "critical":
		return logger.Critical, nil
	case "panic":
		return logger.Panic, nil
	default:
		return logger.Debug, fmt.Errorf("Unknown log level \"%s\"", s)
	}
}

func Apply(l logger.Logger, c Config) error {
	r := logger.Registry()

	if v, ok := c.LogLevel["*"]; ok {
		lv, err := Str2Level(v)
		if err != nil {
			return err
		}
		for _, e := range r.Categories() {
			c := r.Category(e.Category)
			c.Level = lv
		}
	}
	for k, v := range c.LogLevel {
		if k == "*" {
			continue
		}
		c := r.CategoryIfExist(k)
		if c == nil {
			logger.Warningf(l, "Log category \"%s\" does not exist.", k)
			continue
		}
		lv, err := Str2Level(v)
		if err != nil {
			return err
		}
		c.Level = lv
	}
	return nil
}

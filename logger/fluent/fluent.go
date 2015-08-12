package fluent

import (
	"log"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"

	"github.com/nyaxt/otaru/logger"
)

type FluentLogger struct {
	Cli *fluent.Fluent
}

var _ = logger.Logger(FluentLogger{})

func (l FluentLogger) Log(lv logger.Level, data map[string]interface{}) {
	datacp := make(map[string]interface{})
	for k, v := range data {
		if k == "category" || k == "time" {
			continue
		}

		datacp[k] = v
	}
	datacp["level"] = lv.String()

	if err := l.Cli.PostWithTime(data["category"].(string), data["time"].(time.Time), datacp); err != nil {
		log.Printf("Failed to post to fluentd: %v", err)
	}
}
func (l FluentLogger) WillAccept(lv logger.Level) bool { return true }

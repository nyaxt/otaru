package main

import (
	"os"

	"go.uber.org/zap"
)

func main() {
	if err := NewApp().Run(os.Args); err != nil {
		// omit stacktrace
		zap.L().WithOptions(zap.AddStacktrace(zap.FatalLevel)).Error(err.Error())
		os.Exit(1)
	}
}

package facade

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func GenHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		zap.S().Panicf("Failed to query local hostname: %v", err)
	}
	pid := os.Getpid()
	return fmt.Sprintf("%s-%d", hostname, pid)
}

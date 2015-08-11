package facade

import (
	"fmt"
	"os"

	"github.com/nyaxt/otaru/logger"
)

func GenHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Panicf(mylog, "Failed to query local hostname: %v", err)
	}
	pid := os.Getpid()
	return fmt.Sprintf("%s-%d", hostname, pid)
}

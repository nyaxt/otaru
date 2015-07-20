package facade

import (
	"fmt"
	"log"
	"os"
)

func genHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to query local hostname: %v", err)
	}
	pid := os.Getpid()
	return fmt.Sprintf("%s-%d", hostname, pid)
}

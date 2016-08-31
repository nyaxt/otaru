package util

import (
	"regexp"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/nyaxt/otaru/logger"
)

var shouldRetryErrorRegexp = regexp.MustCompile(`status code: (429|5\d\d)`)

func IsShouldRetryError(e error) bool {
	if e == nil {
		return false
	}

	if e == datastore.ErrConcurrentTransaction {
		return true
	}

	estr := e.Error()
	return shouldRetryErrorRegexp.MatchString(estr)
}

func RetryIfNeeded(f func() error, mylog logger.Logger) (err error) {
	const numRetries = 3
	for i := 0; i < numRetries; i++ {
		start := time.Now()
		err = f()
		if err == nil {
			return
		}
		if !IsShouldRetryError(err) {
			return
		}
		if i < numRetries {
			logger.Infof(mylog, "A Google Cloud API operation has failed after %s. Retrying %d / %d...", time.Since(start), i+1, numRetries)
			time.Sleep(time.Duration(i) * time.Second)
		}
	}
	return
}

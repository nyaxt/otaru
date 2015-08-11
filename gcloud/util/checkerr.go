package util

import (
	"regexp"
	"time"

	"google.golang.org/cloud/datastore"

	"github.com/nyaxt/otaru/logger"
)

var shouldRetryErrorRegexp = regexp.MustCompile("http status code: 5\\d\\d")

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
			logger.Debugf(mylog, "A Google Cloud Datastore operation has failed after %s. Retrying %d / %d...", time.Since(start), i+1, numRetries)
		}
	}
	return
}

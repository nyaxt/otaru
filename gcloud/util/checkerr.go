package util

import (
	"regexp"

	"google.golang.org/cloud/datastore"
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

func RetryIfNeeded(f func() error) (err error) {
	const numRetries = 3
	for i := 0; i < numRetries; i++ {
		err = f()
		if err == nil {
			return
		}
		if !IsShouldRetryError(err) {
			return
		}
	}
	return
}

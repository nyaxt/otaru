package util

import (
	"regexp"
)

var shouldRetryErrorRegexp = regexp.MustCompile("http status code: 5\\d\\d")

func IsShouldRetryError(e error) bool {
	if e == nil {
		return false
	}

	estr := e.Error()
	return shouldRetryErrorRegexp.MatchString(estr)
}

package util_test

import (
	"errors"
	"testing"

	"github.com/nyaxt/otaru/gcloud/util"
)

func TestIsShouldRetryError(t *testing.T) {
	if util.IsShouldRetryError(nil) {
		t.Errorf("IsShouldRetryError should have returned false on nil")
	}

	eShouldRetry := errors.New("error during call, http status code: 502")
	if !util.IsShouldRetryError(eShouldRetry) {
		t.Errorf("IsShouldRetryError should have returned true on: %v", eShouldRetry)
	}

	eShouldNotRetry := errors.New("error during call, http status code: 400")
	if util.IsShouldRetryError(eShouldNotRetry) {
		t.Errorf("IsShouldRetryError should have returned false on: %v", eShouldNotRetry)
	}
}

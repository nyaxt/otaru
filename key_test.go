package otaru_test

import (
	"testing"

	"github.com/nyaxt/otaru"
)

func TestKeyFromPassword(t *testing.T) {
	key := otaru.KeyFromPassword("hogefuga")
	if len(key) != 32 {
		t.Errorf("invalid key length: %d", len(key))
	}
	// t.Errorf("gen key: %v", key)
}

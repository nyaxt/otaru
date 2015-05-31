package btncrypt_test

import (
	"testing"

	"github.com/nyaxt/otaru/btncrypt"
)

func TestKeyFromPassword(t *testing.T) {
	key := btncrypt.KeyFromPassword("hogefuga")
	if len(key) != 32 {
		t.Errorf("invalid key length: %d", len(key))
	}
	// t.Errorf("gen key: %v", key)
}

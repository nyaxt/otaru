package testutils

import (
	"github.com/nyaxt/otaru/btncrypt"
)

var (
	Key = []byte("0123456789abcdef0123456789abcdef")
)

func TestCipher() *btncrypt.Cipher {
	c, err := btncrypt.NewCipher(Key)
	if err != nil {
		panic("Failed to init Cipher for testing")
	}
	return c
}

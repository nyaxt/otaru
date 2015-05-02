package otaru_test

import (
	. "github.com/nyaxt/otaru"
)

var (
	Key = []byte("0123456789abcdef")
)

func TestCipher() Cipher {
	c, err := NewCipher(Key)
	if err != nil {
		panic("Failed to init Cipher for testing")
	}
	return c
}

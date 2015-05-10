package otaru

import (
	"crypto/sha1"

	"golang.org/x/crypto/pbkdf2"
)

func KeyFromPassword(password string) []byte {
	return pbkdf2.Key([]byte(password), []byte("otaru"), 4096, 32, sha1.New)
}

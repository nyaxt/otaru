package otaru

import (
	"crypto/rand"
	"io"
)

func Int64Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func Int64Max(a, b int64) int64 {
	if a < b {
		return b
	}
	return a
}

func IntMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func IntMax(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func RandomBytes(size int) []byte {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}
	return nonce
}

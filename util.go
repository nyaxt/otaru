package otaru

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
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

func StringFromFile(filename string) (string, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Failed to read file \"%s\": %v", filename, err)
	}
	return strings.TrimRight(string(b), "\n"), nil
}

func StringFromFileOrDie(filename string) string {
	s, err := StringFromFile(filename)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return s
}

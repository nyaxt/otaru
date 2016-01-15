package util

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
	p := make([]byte, size)
	ReadRandomBytes(p)
	return p
}

func ReadRandomBytes(p []byte) {
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		panic(err)
	}
}

func StringFromFile(filename string) (string, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Failed to read file \"%s\": %v", filename, err)
	}
	return strings.TrimRight(string(b), "\n"), nil
}

func StringFromFileOrDie(filename string, usage string) string {
	s, err := StringFromFile(filename)
	if err != nil {
		log.Fatalf("While fetching %s: %v", usage, err)
	}
	return s
}

package otaru

// better than nothing cryptography.
// This code has not gone through any security audit, so don't trust this code / otaru encryption.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

func randomNonce(size int) []byte {
	nonce := make([]byte, size)
	var l int
	l, err := rand.Read(nonce)
	if err != nil {
		panic(err)
	}
	if l != size {
		panic("generated rand too short")
	}
	return nonce
}

func gcmFromKey(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize AES: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	return gcm, nil
}

func Encrypt(key, plain []byte) ([]byte, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce := randomNonce(nonceSize)

	envelope := gcm.Seal(nonce, nonce, plain, nil)
	return envelope, nil
}

func Decrypt(key, envelope []byte) ([]byte, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce := envelope[:nonceSize]
	encrypted := envelope[nonceSize:]
	plain, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}

	return plain, nil
}

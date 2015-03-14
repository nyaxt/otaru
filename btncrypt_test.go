package otaru

import (
	"testing"
)

func TestEncrypt(t *testing.T) {
	key := []byte("0123456789abcdef")
	envelope, err := Encrypt(key, []byte("0123456789abcdef"))
	if err != nil {
		t.Errorf("Failed to encrypt: %v", err)
	}

	plain, err := Decrypt(key, envelope)
	if err != nil {
		t.Errorf("Failed to decrypt: %v", err)
	}

	t.Errorf("content: %v", string(plain))
}

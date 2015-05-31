package btncrypt_test

import (
	"github.com/nyaxt/otaru/btncrypt"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"

	"bytes"
	"io"
	"testing"
)

func TestEncrypt_Short(t *testing.T) {
	payload := []byte("short string")
	envelope, err := btncrypt.Encrypt(tu.TestCipher(), payload)
	if err != nil {
		t.Errorf("Failed to encrypt: %v", err)
	}

	plain, err := btncrypt.Decrypt(tu.TestCipher(), envelope, len(payload))
	if err != nil {
		t.Errorf("Failed to decrypt: %v", err)
	}

	if !bytes.Equal(plain, payload) {
		t.Errorf("Failed to restore original payload")
	}
}

func TestEncrypt_Long(t *testing.T) {
	payload := util.RandomBytes(1024 * 1024)

	envelope, err := btncrypt.Encrypt(tu.TestCipher(), payload)
	if err != nil {
		t.Errorf("Failed to encrypt: %v", err)
	}

	plain, err := btncrypt.Decrypt(tu.TestCipher(), envelope, len(payload))
	if err != nil {
		t.Errorf("Failed to decrypt: %v", err)
	}

	if !bytes.Equal(payload, plain) {
		t.Errorf("Failed to restore original payload")
	}
}

func verifyWrite(t *testing.T, w io.Writer, payload []byte) {
	n, err := w.Write(payload)
	if err != nil {
		t.Errorf("Failed to write data to BtnEncryptWriter: %v", err)
	}
	if n != len(payload) {
		t.Errorf("bew.Write returned n != len(p)")
	}
}

func TestBtnEncryptWriter_WriteAtOnce(t *testing.T) {
	//payload := RandomBytes(1024 * 1024)
	payload := []byte("short string")

	var b bytes.Buffer
	bew, err := btncrypt.NewWriteCloser(&b, tu.TestCipher(), len(payload))
	if err != nil {
		t.Errorf("Failed to create BtnEncryptWriter: %v", err)
	}

	verifyWrite(t, bew, payload)
	if err := bew.Close(); err != nil {
		t.Errorf("bew.Close failed: %v", err)
	}

	plain, err := btncrypt.Decrypt(tu.TestCipher(), b.Bytes(), len(payload))
	if err != nil {
		t.Errorf("Failed to decrypt: %v", err)
	}

	if !bytes.Equal(payload, plain) {
		t.Errorf("Failed to restore original payload")
	}
}

func TestBtnEncryptWriter_PartialWrite(t *testing.T) {
	payload := util.RandomBytes(1024 * 1024)

	var b bytes.Buffer
	bew, err := btncrypt.NewWriteCloser(&b, tu.TestCipher(), len(payload))
	if err != nil {
		t.Errorf("Failed to create BtnEncryptWriter: %v", err)
	}

	verifyWrite(t, bew, payload[:3])
	verifyWrite(t, bew, payload[3:1024])
	verifyWrite(t, bew, payload[1024:4096])
	verifyWrite(t, bew, payload[4096:])

	if err := bew.Close(); err != nil {
		t.Errorf("bew.Close failed: %v", err)
	}

	plain, err := btncrypt.Decrypt(tu.TestCipher(), b.Bytes(), len(payload))
	if err != nil {
		t.Errorf("Failed to decrypt: %v", err)
	}

	if !bytes.Equal(payload, plain) {
		t.Errorf("Failed to restore original payload")
	}
}

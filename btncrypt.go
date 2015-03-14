package otaru

// better than nothing cryptography.
// This code has not gone through any security audit, so don't trust this code / otaru encryption.

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	BtnFrameMaxPayload = 256 * 1024
)

func RandomBytes(size int) []byte {
	nonce := make([]byte, size)
	var l int
	l, err := rand.Read(nonce)
	if err != nil {
		panic(err)
	}
	if l != size {
		panic("Generated random sequence is too short.")
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
	nonce := RandomBytes(nonceSize)

	envelope := gcm.Seal(nonce, nonce, plain, nil)
	return envelope, nil
}

type frameEncryptor struct {
	gcm       cipher.AEAD
	b         bytes.Buffer
	encrypted []byte
}

func newFrameEncryptor(gcm cipher.AEAD) *frameEncryptor {
	return &frameEncryptor{
		gcm:       gcm,
		encrypted: make([]byte, 0, gcm.NonceSize()+BtnFrameMaxPayload+gcm.Overhead()),
	}
}

func (f *frameEncryptor) Write(p []byte) (int, error) {
	if _, err := f.b.Write(p); err != nil {
		panic(err)
	}

	return len(p), nil
}

func (f *frameEncryptor) Written() int {
	return f.b.Len()
}

func (f *frameEncryptor) CapacityLeft() int {
	return BtnFrameMaxPayload - f.b.Len()
}

func (f *frameEncryptor) Flush() ([]byte, error) {
	if f.Written() > BtnFrameMaxPayload {
		return nil, fmt.Errorf("frame payload size exceeding max len: %d > %d", f.Written(), BtnFrameMaxPayload)
	}

	nonce := RandomBytes(f.gcm.NonceSize())

	f.encrypted = f.encrypted[:len(nonce)]
	copy(f.encrypted, nonce)

	f.encrypted = f.gcm.Seal(f.encrypted, nonce, f.b.Bytes(), nil)
	f.b.Reset()
	return f.encrypted, nil
}

type BtnEncryptWriteCloser struct {
	target   io.Writer
	key      []byte
	lenTotal int
	written  int
	*frameEncryptor
}

func NewBtnEncryptWriteCloser(target io.Writer, key []byte, lenTotal int) (*BtnEncryptWriteCloser, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}

	bew := &BtnEncryptWriteCloser{
		target:         target,
		key:            key,
		lenTotal:       lenTotal,
		written:        0,
		frameEncryptor: newFrameEncryptor(gcm),
	}
	return bew, nil
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (bew *BtnEncryptWriteCloser) flushFrame() error {
	frame, err := bew.frameEncryptor.Flush()
	if err != nil {
		return err
	}
	if _, err := bew.target.Write(frame); err != nil {
		return err
	}
	return nil
}

func (bew *BtnEncryptWriteCloser) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	left := p
	for len(left) > 0 {
		framePayloadLen := intMin(bew.frameEncryptor.CapacityLeft(), len(p))
		framePayload := left[:framePayloadLen]
		if _, err := bew.frameEncryptor.Write(framePayload); err != nil {
			panic(err)
		}
		left = left[framePayloadLen:]

		if bew.frameEncryptor.CapacityLeft() == 0 {
			if err := bew.flushFrame(); err != nil {
				return 0, err
			}
		}
		if bew.frameEncryptor.CapacityLeft() == 0 {
			panic("flushFrame should have brought back capacity")
		}
	}

	bew.written += len(p)
	return len(p), nil
}

func (bew *BtnEncryptWriteCloser) Close() error {
	if bew.lenTotal != bew.written {
		return fmt.Errorf("Incomplete data written")
	}

	if err := bew.flushFrame(); err != nil {
		return err
	}

	return nil
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

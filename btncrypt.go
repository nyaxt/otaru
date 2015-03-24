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
	"log"
)

const (
	BtnFrameMaxPayload = 256 * 1024
)

func RandomBytes(size int) []byte {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
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

func BtnEncryptedFrameSize(gcm cipher.AEAD, payloadLen int) int {
	return gcm.NonceSize() + payloadLen + gcm.Overhead()
}

type frameEncryptor struct {
	gcm       cipher.AEAD
	b         bytes.Buffer
	encrypted []byte
}

func newFrameEncryptor(gcm cipher.AEAD) *frameEncryptor {
	return &frameEncryptor{
		gcm:       gcm,
		encrypted: make([]byte, 0, BtnEncryptedFrameSize(gcm, BtnFrameMaxPayload)),
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
	if len(f.encrypted) != BtnEncryptedFrameSize(f.gcm, f.Written()) {
		log.Panicf("EncryptedFrameSize mismatch. expected: %d, actual: %v", BtnEncryptedFrameSize(f.gcm, f.Written()), len(f.encrypted))
	}
	f.b.Reset()
	return f.encrypted, nil
}

type BtnEncryptWriteCloser struct {
	dst        io.Writer
	lenTotal   int
	lenWritten int
	*frameEncryptor
}

func NewBtnEncryptWriteCloser(dst io.Writer, key []byte, lenTotal int) (*BtnEncryptWriteCloser, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}

	bew := &BtnEncryptWriteCloser{
		dst:            dst,
		lenTotal:       lenTotal,
		lenWritten:     0,
		frameEncryptor: newFrameEncryptor(gcm),
	}
	return bew, nil
}

func IntMin(a, b int) int {
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
	fmt.Printf("emit frame %d\n", len(frame))
	if _, err := bew.dst.Write(frame); err != nil {
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
		framePayloadLen := IntMin(bew.frameEncryptor.CapacityLeft(), len(p))
		framePayload := left[:framePayloadLen]
		if _, err := bew.frameEncryptor.Write(framePayload); err != nil {
			panic(err)
		}
		left = left[framePayloadLen:]
		bew.lenWritten += framePayloadLen

		if bew.frameEncryptor.CapacityLeft() == 0 {
			if err := bew.flushFrame(); err != nil {
				return 0, err
			}
		}
		if bew.frameEncryptor.CapacityLeft() == 0 {
			panic("flushFrame should have brought back capacity")
		}
	}

	return len(p), nil
}

func (bew *BtnEncryptWriteCloser) Close() error {
	if bew.lenTotal != bew.lenWritten {
		return fmt.Errorf("Incomplete data written")
	}

	if err := bew.flushFrame(); err != nil {
		return err
	}

	return nil
}

func Encrypt(key, plain []byte) ([]byte, error) {
	var b bytes.Buffer
	bew, err := NewBtnEncryptWriteCloser(&b, key, len(plain))
	if err != nil {
		return nil, err
	}
	if _, err := bew.Write(plain); err != nil {
		return nil, err
	}
	if err := bew.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// FIXME: add seek
type BtnDecryptReader struct {
	src       io.Reader
	gcm       cipher.AEAD
	lenTotal  int
	lenRead   int
	decrypted []byte
	unread    []byte
	encrypted []byte
}

func NewBtnDecryptReader(src io.Reader, key []byte, lenTotal int) (*BtnDecryptReader, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}

	bdr := &BtnDecryptReader{
		src:       src,
		gcm:       gcm,
		lenTotal:  lenTotal,
		lenRead:   0,
		decrypted: make([]byte, 0, BtnFrameMaxPayload),
		encrypted: make([]byte, 0, BtnEncryptedFrameSize(gcm, BtnFrameMaxPayload)),
	}
	return bdr, nil
}

func (bdr *BtnDecryptReader) decryptNextFrame() error {
	frameLen := IntMin(bdr.lenTotal-bdr.lenRead, BtnFrameMaxPayload)
	encryptedFrameLen := BtnEncryptedFrameSize(bdr.gcm, frameLen)
	// fmt.Printf("frameLen: %d, encryptedFrameLen: %d\n", frameLen, encryptedFrameLen)

	bdr.encrypted = bdr.encrypted[:encryptedFrameLen]
	if _, err := io.ReadFull(bdr.src, bdr.encrypted); err != nil {
		return err
	}

	nonce := bdr.encrypted[:bdr.gcm.NonceSize()]
	ciphertext := bdr.encrypted[bdr.gcm.NonceSize():]

	var err error
	bdr.decrypted = bdr.decrypted[:0]
	if bdr.decrypted, err = bdr.gcm.Open(bdr.decrypted, nonce, ciphertext, nil); err != nil {
		return err
	}
	bdr.unread = bdr.decrypted

	return nil
}

func (bdr *BtnDecryptReader) Read(p []byte) (int, error) {
	nr := IntMin(len(p), bdr.lenTotal-bdr.lenRead)
	left := p[:nr]

	if nr == 0 {
		return 0, io.EOF
	}

	n := 0
	for len(left) > 0 {
		if len(bdr.unread) == 0 {
			if err := bdr.decryptNextFrame(); err != nil {
				return n, err
			}
		}
		if len(bdr.unread) == 0 {
			panic("decryptNextFrame should have decrypted something and placed it on the buf")
		}

		consumeLen := IntMin(len(bdr.unread), len(left))
		copy(left[:consumeLen], bdr.unread[:consumeLen])
		bdr.unread = bdr.unread[consumeLen:]
		left = left[consumeLen:]
		n += consumeLen
		bdr.lenRead += consumeLen
	}

	return n, nil
}

func (bdr *BtnDecryptReader) HasReadAll() bool {
	return bdr.lenRead == bdr.lenTotal
}

func Decrypt(key, envelope []byte, lenTotal int) ([]byte, error) {
	bdr, err := NewBtnDecryptReader(bytes.NewReader(envelope), key, lenTotal)
	if err != nil {
		return nil, err
	}

	ret := make([]byte, lenTotal)
	if _, err := io.ReadFull(bdr, ret); err != nil {
		return nil, err
	}
	return ret, err
}

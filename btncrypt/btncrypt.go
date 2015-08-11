package btncrypt

// better than nothing cryptography.
// This code has not gone through any security audit, so don't trust this code / otaru encryption.

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("btncrypt")

const (
	BtnFrameMaxPayload = 256 * 1024
)

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

type Cipher struct {
	gcm cipher.AEAD
}

func NewCipher(key []byte) (Cipher, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return Cipher{}, err
	}

	return Cipher{gcm: gcm}, nil
}

func (c Cipher) FrameOverhead() int {
	return c.gcm.NonceSize() + c.gcm.Overhead()
}

func (c Cipher) EncryptedFrameSize(payloadLen int) int {
	return payloadLen + c.FrameOverhead()
}

type frameEncryptor struct {
	c         Cipher
	b         bytes.Buffer
	encrypted []byte
}

func newFrameEncryptor(c Cipher) *frameEncryptor {
	lenEncrypted := c.EncryptedFrameSize(BtnFrameMaxPayload)
	return &frameEncryptor{
		c:         c,
		encrypted: make([]byte, 0, lenEncrypted),
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

func (f *frameEncryptor) Sync() ([]byte, error) {
	if f.Written() > BtnFrameMaxPayload {
		return nil, fmt.Errorf("frame payload size exceeding max len: %d > %d", f.Written(), BtnFrameMaxPayload)
	}

	nonce := util.RandomBytes(f.c.gcm.NonceSize())

	f.encrypted = f.encrypted[:len(nonce)]
	copy(f.encrypted, nonce)

	f.encrypted = f.c.gcm.Seal(f.encrypted, nonce, f.b.Bytes(), nil)
	if len(f.encrypted) != f.c.EncryptedFrameSize(f.Written()) {
		logger.Panicf(mylog, "EncryptedFrameSize mismatch. expected: %d, actual: %v", f.c.EncryptedFrameSize(f.Written()), len(f.encrypted))
	}
	f.b.Reset()
	return f.encrypted, nil
}

type WriteCloser struct {
	dst        io.Writer
	lenTotal   int
	lenWritten int
	*frameEncryptor
}

func NewWriteCloser(dst io.Writer, c Cipher, lenTotal int) (*WriteCloser, error) {
	bew := &WriteCloser{
		dst:            dst,
		lenTotal:       lenTotal,
		lenWritten:     0,
		frameEncryptor: newFrameEncryptor(c),
	}
	return bew, nil
}

func (bew *WriteCloser) flushFrame() error {
	if bew.frameEncryptor.Written() == 0 {
		// Don't emit a frame with empty payload
		return nil
	}

	frame, err := bew.frameEncryptor.Sync()
	if err != nil {
		return err
	}
	// fmt.Printf("emit frame %d\n", len(frame))
	if _, err := bew.dst.Write(frame); err != nil {
		return err
	}
	return nil
}

func (bew *WriteCloser) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	left := p
	for len(left) > 0 {
		framePayloadLen := util.IntMin(bew.frameEncryptor.CapacityLeft(), len(left))
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

func (bew *WriteCloser) Close() error {
	if bew.lenTotal != bew.lenWritten {
		return fmt.Errorf("Frame len different from declared. %d / %d bytes", bew.lenWritten, bew.lenTotal)
	}

	if err := bew.flushFrame(); err != nil {
		return err
	}

	return nil
}

func Encrypt(c Cipher, plain []byte) ([]byte, error) {
	var b bytes.Buffer
	bew, err := NewWriteCloser(&b, c, len(plain))
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

type Reader struct {
	src       io.Reader
	c         Cipher
	lenTotal  int
	lenRead   int
	decrypted []byte
	unread    []byte
	encrypted []byte
}

func NewReader(src io.Reader, c Cipher, lenTotal int) (*Reader, error) {
	bdr := &Reader{
		src:       src,
		c:         c,
		lenTotal:  lenTotal,
		lenRead:   0,
		decrypted: make([]byte, 0, BtnFrameMaxPayload),
		encrypted: make([]byte, 0, c.EncryptedFrameSize(BtnFrameMaxPayload)),
	}
	return bdr, nil
}

func (bdr *Reader) decryptNextFrame() error {
	frameLen := util.IntMin(bdr.lenTotal-bdr.lenRead, BtnFrameMaxPayload)
	encryptedFrameLen := bdr.c.EncryptedFrameSize(frameLen)
	// fmt.Printf("frameLen: %d, encryptedFrameLen: %d\n", frameLen, encryptedFrameLen)

	bdr.encrypted = bdr.encrypted[:encryptedFrameLen]
	if _, err := io.ReadFull(bdr.src, bdr.encrypted); err != nil {
		return err
	}

	nonceSize := bdr.c.gcm.NonceSize()
	nonce := bdr.encrypted[:nonceSize]
	ciphertext := bdr.encrypted[nonceSize:]

	var err error
	bdr.decrypted = bdr.decrypted[:0]
	if bdr.decrypted, err = bdr.c.gcm.Open(bdr.decrypted, nonce, ciphertext, nil); err != nil {
		return err
	}
	bdr.unread = bdr.decrypted

	return nil
}

func (bdr *Reader) Read(p []byte) (int, error) {
	nr := util.IntMin(len(p), bdr.lenTotal-bdr.lenRead)
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

		consumeLen := util.IntMin(len(bdr.unread), len(left))
		copy(left[:consumeLen], bdr.unread[:consumeLen])
		bdr.unread = bdr.unread[consumeLen:]
		left = left[consumeLen:]
		n += consumeLen
		bdr.lenRead += consumeLen
	}

	return n, nil
}

func (bdr *Reader) HasReadAll() bool {
	return bdr.lenRead == bdr.lenTotal
}

func Decrypt(c Cipher, envelope []byte, lenTotal int) ([]byte, error) {
	bdr, err := NewReader(bytes.NewReader(envelope), c, lenTotal)
	if err != nil {
		return nil, err
	}

	ret := make([]byte, lenTotal)
	if _, err := io.ReadFull(bdr, ret); err != nil {
		return nil, err
	}
	return ret, err
}

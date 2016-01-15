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
	gcm           cipher.AEAD
	frameOverhead int
}

func NewCipher(key []byte) (*Cipher, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}

	return &Cipher{
		gcm:           gcm,
		frameOverhead: gcm.NonceSize() + gcm.Overhead(),
	}, nil
}

func (c *Cipher) FrameOverhead() int {
	return c.frameOverhead
}

func (c *Cipher) EncryptedFrameSize(payloadLen int) int {
	return payloadLen + c.frameOverhead
}

func (c *Cipher) EncryptFrame(dst []byte, p []byte) []byte {
	expectedLen := c.EncryptedFrameSize(len(p))
	if cap(dst) < expectedLen {
		logger.Panicf(mylog, "dst should be large enough to hold encyrpted frame! cap(dst) = %d, expectedLen = %d", cap(dst), expectedLen)
	}

	dst = dst[:c.gcm.NonceSize()]
	util.ReadRandomBytes(dst)

	dst = c.gcm.Seal(dst, dst, p, nil)
	if len(dst) != expectedLen {
		logger.Panicf(mylog, "EncryptedFrameSize mismatch. expected: %d, actual: %v", expectedLen, len(dst))
	}
	return dst
}

type WriteCloser struct {
	dst        io.Writer
	lenTotal   int
	lenWritten int

	c *Cipher
	b bytes.Buffer

	encBuf []byte
	remBuf []byte
}

func NewWriteCloser(dst io.Writer, c *Cipher, lenTotal int) (*WriteCloser, error) {
	bew := &WriteCloser{
		dst:        dst,
		lenTotal:   lenTotal,
		lenWritten: 0,
		c:          c,

		encBuf: make([]byte, c.EncryptedFrameSize(BtnFrameMaxPayload)), // FIXME: sync.Pool
		remBuf: make([]byte, 0, BtnFrameMaxPayload),                    // FIXME: sync.Pool
	}
	return bew, nil
}

func (bew *WriteCloser) emitFrame(p []byte) error {
	enc := bew.c.EncryptFrame(bew.encBuf[:0], p)
	if _, err := bew.dst.Write(enc); err != nil {
		return err
	}
	return nil
}

func (bew *WriteCloser) Write(p []byte) (int, error) {
	totalLen := len(p)
	if totalLen == 0 {
		return 0, nil
	}

	if len(bew.remBuf) > 0 {
		if len(bew.remBuf)+len(p) < BtnFrameMaxPayload {
			bew.remBuf = append(bew.remBuf, p...)
			bew.lenWritten += totalLen
			return totalLen, nil
		} else {
			consume := BtnFrameMaxPayload - len(bew.remBuf)
			bew.remBuf = append(bew.remBuf, p[:consume]...)
			if err := bew.emitFrame(bew.remBuf); err != nil {
				return 0, err
			}
			p = p[consume:]
			bew.remBuf = bew.remBuf[:0]
		}
	}
	for len(p) >= BtnFrameMaxPayload {
		if err := bew.emitFrame(p[:BtnFrameMaxPayload]); err != nil {
			return 0, err
		}
		p = p[BtnFrameMaxPayload:]
	}
	if len(p) > 0 {
		bew.remBuf = append(bew.remBuf, p...)
	}

	bew.lenWritten += totalLen
	return totalLen, nil
}

func (bew *WriteCloser) Close() error {
	if bew.lenTotal != bew.lenWritten {
		return fmt.Errorf("Frame len different from declared. %d / %d bytes", bew.lenWritten, bew.lenTotal)
	}

	if len(bew.remBuf) > 0 {
		if err := bew.emitFrame(bew.remBuf); err != nil {
			return err
		}
		bew.remBuf = bew.remBuf[:0]
	}

	return nil
}

func Encrypt(c *Cipher, plain []byte) ([]byte, error) {
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
	c         *Cipher
	lenTotal  int
	lenRead   int
	decrypted []byte
	unread    []byte
	encrypted []byte
}

func NewReader(src io.Reader, c *Cipher, lenTotal int) (*Reader, error) {
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

func Decrypt(c *Cipher, envelope []byte, lenTotal int) ([]byte, error) {
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

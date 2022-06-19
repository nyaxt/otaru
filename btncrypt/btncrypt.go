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
	"go.uber.org/zap"
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

	poolEncryptedFrameBuf *util.GuaranteedPool
	poolDecryptedFrameBuf *util.GuaranteedPool

	poolWriteCloser *util.GuaranteedPool
	poolReader      *util.GuaranteedPool
}

func NewCipher(key []byte) (*Cipher, error) {
	gcm, err := gcmFromKey(key)
	if err != nil {
		return nil, err
	}

	c := &Cipher{
		gcm:           gcm,
		frameOverhead: gcm.NonceSize() + gcm.Overhead(),
	}
	c.poolEncryptedFrameBuf = util.NewGuaranteedPool(func() interface{} {
		return make([]byte, 0, c.EncryptedFrameSize(BtnFrameMaxPayload))
	}, 4)
	c.poolDecryptedFrameBuf = util.NewGuaranteedPool(func() interface{} {
		return make([]byte, 0, BtnFrameMaxPayload)
	}, 4)
	c.poolWriteCloser = util.NewGuaranteedPool(func() interface{} {
		return &WriteCloser{
			c:      c,
			encBuf: c.GetEncryptedFrameBuf(),
			remBuf: c.GetDecryptedFrameBuf(),
		}
	}, 1)
	c.poolReader = util.NewGuaranteedPool(func() interface{} {
		return &Reader{
			c:         c,
			decrypted: c.GetDecryptedFrameBuf(),
			encrypted: c.GetEncryptedFrameBuf(),
		}
	}, 1)
	return c, nil
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
		zap.S().Panicf("dst should be large enough to hold encyrpted frame! cap(dst) = %d, expectedLen = %d", cap(dst), expectedLen)
	}

	dst = dst[:c.gcm.NonceSize()]
	util.ReadRandomBytes(dst)

	dst = c.gcm.Seal(dst, dst, p, nil)
	if len(dst) != expectedLen {
		zap.S().Panicf("EncryptedFrameSize mismatch. expected: %d, actual: %v", expectedLen, len(dst))
	}
	return dst
}

func (c *Cipher) DecryptFrame(dst []byte, p []byte) ([]byte, error) {
	expectedLen := len(p) - c.FrameOverhead()
	if cap(dst) < expectedLen {
		zap.S().Panicf("dst should be large enough to hold decyrpted frame! cap(dst) = %d, expectedLen = %d", cap(dst), expectedLen)
	}

	nonceSize := c.gcm.NonceSize()
	nonce := p[:nonceSize]
	ciphertext := p[nonceSize:]

	dst = dst[:0]
	dst, err := c.gcm.Open(dst, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func (c *Cipher) GetEncryptedFrameBuf() []byte {
	return c.poolEncryptedFrameBuf.Get().([]byte)
}

func (c *Cipher) PutEncryptedFrameBuf(p []byte) {
	c.poolEncryptedFrameBuf.Put(p)
}

func (c *Cipher) GetDecryptedFrameBuf() []byte {
	return c.poolDecryptedFrameBuf.Get().([]byte)
}

func (c *Cipher) PutDecryptedFrameBuf(p []byte) {
	c.poolDecryptedFrameBuf.Put(p)
}

type WriteCloser struct {
	c *Cipher

	dst        io.Writer
	lenTotal   int
	lenWritten int

	encBuf []byte
	remBuf []byte
}

func (c *Cipher) NewWriteCloser(dst io.Writer, lenTotal int) (*WriteCloser, error) {
	bew := c.poolWriteCloser.Get().(*WriteCloser)
	bew.dst = dst
	bew.lenTotal = lenTotal
	bew.lenWritten = 0
	bew.remBuf = bew.remBuf[:0]

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

	bew.dst = nil
	bew.c.poolWriteCloser.Put(bew)

	return nil
}

func Encrypt(c *Cipher, plain []byte) ([]byte, error) {
	var b bytes.Buffer
	bew, err := c.NewWriteCloser(&b, len(plain))
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
	c *Cipher

	src      io.Reader
	lenTotal int
	lenRead  int

	decrypted []byte
	unread    []byte
	encrypted []byte
}

func (c *Cipher) NewReader(src io.Reader, lenTotal int) (*Reader, error) {
	bdr := c.poolReader.Get().(*Reader)

	bdr.src = src
	bdr.lenTotal = lenTotal
	bdr.lenRead = 0

	bdr.decrypted = bdr.decrypted[:0]
	bdr.encrypted = bdr.encrypted[:0]

	return bdr, nil
}

func (bdr *Reader) decryptNextFrame() error {
	frameLen := util.IntMin(bdr.lenTotal-bdr.lenRead, BtnFrameMaxPayload)
	encryptedFrameLen := bdr.c.EncryptedFrameSize(frameLen)

	bdr.encrypted = bdr.encrypted[:encryptedFrameLen]
	if _, err := io.ReadFull(bdr.src, bdr.encrypted); err != nil {
		return err
	}

	dec, err := bdr.c.DecryptFrame(bdr.decrypted[:0], bdr.encrypted)
	if err != nil {
		return err
	}
	bdr.unread = dec

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

func (bdr *Reader) Close() error {
	bdr.src = nil
	bdr.c.poolReader.Put(bdr)
	return nil
}

func Decrypt(c *Cipher, envelope []byte, lenTotal int) ([]byte, error) {
	bdr, err := c.NewReader(bytes.NewReader(envelope), lenTotal)
	if err != nil {
		return nil, err
	}
	defer bdr.Close()

	ret := make([]byte, lenTotal)
	if _, err := io.ReadFull(bdr, ret); err != nil {
		return nil, err
	}
	return ret, err
}

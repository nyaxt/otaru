package otaru

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type BlobStore interface {
	OpenWriter(blobpath string) (io.WriteCloser, error)
}

type FileBlobStore struct{}

func (f *FileBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	return os.Create(blobpath)
}

type ChunkHeader struct {
	PayloadLen   int64
	OrigFilename string
	OrigOffset   int64
}

type ChunkWriter struct {
	w           io.WriteCloser
	key         []byte
	bew         *BtnEncryptWriteCloser
	wroteHeader bool
}

func NewChunkWriter(w io.WriteCloser, key []byte) *ChunkWriter {
	return &ChunkWriter{w: w, key: key, wroteHeader: false}
}

func (cw *ChunkWriter) WriteHeader(h *ChunkHeader) error {
	if cw.wroteHeader {
		return errors.New("Already wrote header")
	}

	hjson, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("Serializing header failed: %v", err)
	}

	bew, err := NewBtnEncryptWriteCloser(cw.w, cw.key, len(hjson))
	if err != nil {
		return fmt.Errorf("Failed to initialize frame encryptor: %v", err)
	}
	if _, err := bew.Write(hjson); err != nil {
		return fmt.Errorf("Header frame write failed: %v", err)
	}
	if err := bew.Close(); err != nil {
		return fmt.Errorf("Header frame close failed: %v", err)
	}

	cw.wroteHeader = true
	return nil
}

func (cw *ChunkWriter) Write(p []byte) (int, error) {
	if !cw.wroteHeader {
		return 0, errors.New("Header is not yet written to chunk")
	}

	unwritten := p
	for len(unwritten) > 0 {
		if cw.bew == nil || cw.bew.CapacityLeft() == 0 {
			if cw.bew != nil {
				cw.bew.Close()
			}
			framelen := len(p) // FIXME: This should be calculated from cw.CapacityLeft
			var err error
			cw.bew, err = NewBtnEncryptWriteCloser(cw.w, cw.key, framelen)
			if err != nil {
				return 0, fmt.Errorf("Failed to initialize frame encryptor: %v", err)
			}
		}

		wlen := IntMin(cw.bew.CapacityLeft(), len(p))
		if _, err := cw.bew.Write(unwritten[:wlen]); err != nil {
			return 0, fmt.Errorf("Failed to write encrypted frame: %v", err)
		}
		unwritten = unwritten[wlen:]
	}

	return len(p), nil
}

func (cw *ChunkWriter) Close() error {
	if cw.bew != nil {
		if err := cw.bew.Close(); err != nil {
			return err
		}
	}
	if err := cw.w.Close(); err != nil {
		return err
	}
	return nil
}

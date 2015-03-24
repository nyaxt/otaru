package otaru

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	MaxMarshaledPrologueLength = 4096
)

type BlobStore interface {
	OpenWriter(blobpath string) (io.WriteCloser, error)
	OpenReader(blobpath string) (io.ReadCloser, error)
}

type FileBlobStore struct{}

func (f *FileBlobStore) OpenWriter(blobpath string) (io.WriteCloser, error) {
	return os.Create(blobpath)
}

func (f *FileBlobStore) OpenReader(blobpath string) (io.ReadCloser, error) {
	return os.Open(blobpath)
}

type ChunkPrologue struct {
	PayloadLen   int
	OrigFilename string
	OrigOffset   int64
}

type ChunkWriter struct {
	w             io.WriteCloser
	key           []byte
	bew           *BtnEncryptWriteCloser
	wroteHeader   bool
	wroteEpilogue bool
	lenTotal      int
}

func NewChunkWriter(w io.WriteCloser, key []byte) *ChunkWriter {
	return &ChunkWriter{
		w: w, key: key,
		wroteHeader:   false,
		wroteEpilogue: false,
	}
}

func (cw *ChunkWriter) WriteHeaderAndPrologue(p *ChunkPrologue) error {
	if cw.wroteHeader {
		return errors.New("Already wrote header")
	}

	pjson, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("Serializing header failed: %v", err)
	}
	cw.lenTotal = p.PayloadLen

	if len(pjson) > MaxMarshaledPrologueLength {
		panic("marshaled prologue too long!")
	}
	prologueLength := uint16(len(pjson))

	hdr := ChunkHeader{
		Format:             0x01,
		FrameEncapsulation: 0x01,
		PrologueLength:     prologueLength,
		EpilogueLength:     0,
	}
	bhdr, err := hdr.MarshalBinary()
	if err != nil {
		return fmt.Errorf("Failed to marshal ChunkHeader: %v", err)
	}
	if _, err := cw.w.Write(bhdr); err != nil {
		return fmt.Errorf("Header write failed: %v", err)
	}

	bew, err := NewBtnEncryptWriteCloser(cw.w, cw.key, len(pjson))
	if err != nil {
		return fmt.Errorf("Failed to initialize frame encryptor: %v", err)
	}
	if _, err := bew.Write(pjson); err != nil {
		return fmt.Errorf("Prologue frame write failed: %v", err)
	}
	if err := bew.Close(); err != nil {
		return fmt.Errorf("Prologue frame close failed: %v", err)
	}

	cw.wroteHeader = true
	return nil
}

func (cw *ChunkWriter) Write(p []byte) (int, error) {
	if !cw.wroteHeader {
		return 0, errors.New("Header is not yet written to chunk")
	}

	if cw.bew == nil {
		var err error
		cw.bew, err = NewBtnEncryptWriteCloser(cw.w, cw.key, cw.lenTotal)
		if err != nil {
			return 0, fmt.Errorf("Failed to initialize frame encryptor: %v", err)
		}
	}

	if _, err := cw.bew.Write(p); err != nil {
		return 0, fmt.Errorf("Failed to write encrypted frame: %v", err)
	}

	return len(p), nil
}

func (cw *ChunkWriter) Close() error {
	if cw.bew != nil {
		if err := cw.bew.Close(); err != nil {
			return err
		}
	}

	// FIXME: Write epilogue

	if err := cw.w.Close(); err != nil {
		return err
	}
	return nil
}

type ChunkReader struct {
	r   io.ReadCloser
	key []byte

	bdr *BtnDecryptReader

	didReadHeader   bool
	header          ChunkHeader
	didReadPrologue bool
	prologue        ChunkPrologue

	lenTotal int
}

func NewChunkReader(r io.ReadCloser, key []byte) *ChunkReader {
	return &ChunkReader{
		r: r, key: key,
		didReadHeader: false, didReadPrologue: false,
	}
}

func (cr *ChunkReader) ReadHeader() error {
	if cr.didReadHeader {
		return errors.New("Already read header.")
	}

	b := make([]byte, MarshaledChunkHeaderLength)
	if _, err := io.ReadFull(cr.r, b); err != nil {
		return fmt.Errorf("Failed to read ChunkHeader: %v", err)
	}

	if err := cr.header.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("Failed to unmarshal ChunkHeader: %v", err)
	}

	cr.didReadHeader = true
	return nil
}

func (cr *ChunkReader) Header() ChunkHeader {
	if !cr.didReadHeader {
		panic("Tried to access header before reading it.")
	}
	return cr.header
}

func (cr *ChunkReader) ReadPrologue() error {
	if cr.didReadPrologue {
		return errors.New("Already read prologue.")
	}
	if !cr.didReadHeader {
		return errors.New("Tried to read prologue before reading header.")
	}

	bdr, err := NewBtnDecryptReader(cr.r, cr.key, int(cr.header.PrologueLength))
	if err != nil {
		return err
	}

	mpro := make([]byte, cr.header.PrologueLength)
	if _, err := io.ReadFull(bdr, mpro); err != nil {
		return fmt.Errorf("Failed to read prologue frame: %v", err)
	}
	if !bdr.HasReadAll() {
		panic("Incomplete read in prologue frame !?!?")
	}

	if err := json.Unmarshal(mpro, &cr.prologue); err != nil {
		return fmt.Errorf("Failed to unmarshal prologue: %v", err)
	}

	cr.didReadPrologue = true
	return nil
}

func (cr *ChunkReader) Prologue() ChunkPrologue {
	if !cr.didReadPrologue {
		panic("Tried to access prologue before reading it.")
	}
	return cr.prologue
}

// Length returns length of content.
func (cr *ChunkReader) Length() int {
	return cr.prologue.PayloadLen
}

func (cr *ChunkReader) Read(p []byte) (int, error) {
	if !cr.didReadPrologue {
		return 0, errors.New("Tried to read content before reading prologue.")
	}

	if cr.bdr == nil {
		var err error
		cr.bdr, err = NewBtnDecryptReader(cr.r, cr.key, cr.Length())
		if err != nil {
			return 0, err
		}
	}

	nr, err := cr.bdr.Read(p)
	return nr, err
}

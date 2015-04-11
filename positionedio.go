package otaru

// PReader implements positioned read
type PReader interface {
	PRead(offset int64, p []byte) error
}

type ZeroFillPReader struct{}

func (ZeroFillPReader) PRead(offset int64, p []byte) error {
	for i := range p {
		p[i] = 0
	}
	return nil
}

// PWriter implements positioned write
type PWriter interface {
	PWrite(offset int64, p []byte) error
}

type RandomAccessIO interface {
	PReader
	PWriter
}

// OffsetReader provides io.Reader from PReader
type OffsetReader struct {
	PReader
	Offset int64
}

func (r *OffsetReader) Read(p []byte) (int, error) {
	if err := r.PReader.PRead(r.Offset, p); err != nil {
		return 0, err
	}

	r.Offset += int64(len(p))
	return len(p), nil
}

// OffsetWriter provides io.Reader from PWriter
type OffsetWriter struct {
	PWriter
	Offset int64
}

func (w *OffsetWriter) Write(p []byte) (int, error) {
	if err := w.PWriter.PWrite(w.Offset, p); err != nil {
		return 0, err
	}

	w.Offset += int64(len(p))
	return len(p), nil
}

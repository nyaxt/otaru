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

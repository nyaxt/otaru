package otaru

import (
//	"github.com/nyaxt/otaru/intn"
)

const (
	ChunkSplitSize = 256 * 1024 * 1024 // 256MB
)

type ChunkedFileIO struct {
	bs RandomAccessBlobStore
	fn *FileNode

	key []byte
}

func NewChunkedFileIO(bs RandomAccessBlobStore, fn *FileNode, key []byte) *ChunkedFileIO {
	return &ChunkedFileIO{bs: bs, fn: fn, key: key}
}

func GenerateNewBlobPath() string {
	return "fixme"
}

// FIXME: make this WriteV(ps intn.Patches)
func (io *ChunkedFileIO) PWrite(offset int64, p []byte) error {
	fn := io.fn

	remo := offset
	remp := p
	if len(remp) == 0 {
		return nil
	}

	writeToChunk := func(c *FileChunk, maxChunkLen int64) error {
		bh, err := io.bs.Open(c.BlobPath)
		if err != nil {
			return err
		}
		cw := NewChunkIO(bh, io.key)

		coffset := remo - c.Offset
		part := remp
		right := coffset + int64(len(remp))
		if right > maxChunkLen {
			part = part[:maxChunkLen-right]
		}
		if err := cw.PWrite(coffset, part); err != nil {
			return err
		}
		if err := cw.Close(); err != nil {
			return err
		}

		remo += int64(len(part))
		remp = remp[len(part):]
		return nil
	}

	i := 0
	for i < len(fn.Chunks) {
		c := &fn.Chunks[i]
		if c.Offset > remo {
			// Insert a new chunk @ i
			newo := remo / ChunkSplitSize * ChunkSplitSize
			maxlen := int64(ChunkSplitSize)
			if i > 0 {
				prev := fn.Chunks[i-1]
				pright := prev.Offset + prev.Length
				if newo < pright {
					maxlen -= pright - newo
					newo = pright
				}
			}
			if i < len(fn.Chunks)-1 {
				next := fn.Chunks[i+1]
				if newo+maxlen > next.Offset {
					maxlen = next.Offset - newo
				}
			}

			newc := FileChunk{Offset: newo, Length: 0, BlobPath: GenerateNewBlobPath()}
			fn.Chunks = append(fn.Chunks, FileChunk{})
			copy(fn.Chunks[i+1:], fn.Chunks[i:])
			fn.Chunks[i] = newc

			if err := writeToChunk(&newc, maxlen); err != nil {
				return err
			}
			if len(remp) == 0 {
				return nil
			}

			continue
		}

		if remo >= c.Offset+c.Length {
			continue
		}

		// Write to the chunk
		maxlen := int64(ChunkSplitSize)
		if i < len(fn.Chunks)-1 {
			next := fn.Chunks[i+1]
			if c.Offset+maxlen > next.Offset {
				maxlen = next.Offset - c.Offset
			}
		}
		if err := writeToChunk(c, maxlen); err != nil {
			return err
		}
		if len(remp) == 0 {
			return nil
		}
	}

	for len(remp) > 0 {
		// Append a new chunk at the end
		newo := remo / ChunkSplitSize * ChunkSplitSize
		maxlen := int64(ChunkSplitSize)

		if len(fn.Chunks) > 0 {
			last := fn.Chunks[len(fn.Chunks)-1]
			lastRight := last.Offset + last.Length
			if newo < lastRight {
				maxlen -= lastRight - newo
				newo = lastRight
			}
		}

		newc := FileChunk{Offset: newo, Length: 0, BlobPath: GenerateNewBlobPath()}
		if err := writeToChunk(&newc, maxlen); err != nil {
			return err
		}

		fn.Chunks = append(fn.Chunks, newc)
	}

	return nil
}

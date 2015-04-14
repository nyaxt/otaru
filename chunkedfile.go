package otaru

import (
	//	"github.com/nyaxt/otaru/intn"
	"fmt"
)

const (
	ChunkSplitSize = 256 * 1024 * 1024 // 256MB
)

type ChunkedFileIO struct {
	bs RandomAccessBlobStore
	fn *FileNode
	c  Cipher
}

func NewChunkedFileIO(bs RandomAccessBlobStore, fn *FileNode, c Cipher) *ChunkedFileIO {
	return &ChunkedFileIO{bs: bs, fn: fn, c: c}
}

func GenerateNewBlobPath() string {
	return "fixme"
}

func (cfio *ChunkedFileIO) PWrite(offset int64, p []byte) error {
	remo := offset
	remp := p
	if len(remp) == 0 {
		return nil
	}

	writeToChunk := func(c *FileChunk, maxChunkLen int64) error {
		bh, err := cfio.bs.Open(c.BlobPath)
		if err != nil {
			return err
		}
		cw := NewChunkIO(bh, cfio.c)

		coff := remo - c.Offset
		part := remp
		right := coff + int64(len(remp))
		if right > maxChunkLen {
			part = part[:maxChunkLen-right]
		}
		if err := cw.PWrite(coff, part); err != nil {
			return err
		}
		if err := cw.Close(); err != nil {
			return err
		}
		c.Length = int64(cw.PayloadLen())

		remo += int64(len(part))
		remp = remp[len(part):]
		return nil
	}

	fn := cfio.fn

	i := 0
	for i < len(fn.Chunks) {
		c := &fn.Chunks[i]
		if c.Left() > remo {
			// Insert a new chunk @ i
			newo := remo / ChunkSplitSize * ChunkSplitSize
			maxlen := int64(ChunkSplitSize)
			if i > 0 {
				prev := fn.Chunks[i-1]
				pright := prev.Right()
				if newo < pright {
					maxlen -= pright - newo
					newo = pright
				}
			}
			if i < len(fn.Chunks)-1 {
				next := fn.Chunks[i+1]
				if newo+maxlen > next.Left() {
					maxlen = next.Left() - newo
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

		if remo >= c.Right() {
			continue
		}

		// Write to the chunk
		maxlen := int64(ChunkSplitSize)
		if i < len(fn.Chunks)-1 {
			next := fn.Chunks[i+1]
			if c.Left()+maxlen > next.Left() {
				maxlen = next.Left() - c.Left()
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
			lastRight := last.Right()
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

func (cfio *ChunkedFileIO) PRead(offset int64, p []byte) error {
	remo := offset
	remp := p

	if offset < 0 {
		return fmt.Errorf("negative offset %d given", offset)
	}

	cs := cfio.fn.Chunks
	fmt.Printf("Chunks: %v\n", cs)
	for i := 0; i < len(cs) && len(remp) > 0; i++ {
		c := cs[i]
		if c.Left() > remo+int64(len(remp)) {
			break
		}
		if c.Right() <= remo {
			continue
		}

		coff := remo - c.Left()
		if coff < 0 {
			// Fill gap with zero
			n := Int64Min(int64(len(remp)), -coff)
			for j := int64(0); j < n; j++ {
				remp[j] = 0
			}
			remo += n
			coff = 0
			if len(remp) == 0 {
				return nil
			}
		}

		bh, err := cfio.bs.Open(c.BlobPath)
		if err != nil {
			return err
		}
		cw := NewChunkIO(bh, cfio.c)

		n := Int64Min(int64(len(p)), c.Length-coff)
		if err := cw.PRead(coff, remp[:n]); err != nil {
			return err
		}
		if err := cw.Close(); err != nil {
			return err
		}

		remo += n
		remp = remp[n:]

		if len(remp) == 0 {
			return nil
		}
	}

	return fmt.Errorf("Attempt to read over file size by %d", len(remp))
}

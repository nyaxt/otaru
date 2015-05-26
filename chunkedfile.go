package otaru

import (
	"fmt"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/inodedb"
	. "github.com/nyaxt/otaru/util" // FIXME
)

const (
	ChunkSplitSize = 256 * 1024 * 1024 // 256MB
)

const (
	NewChunk      = true
	ExistingChunk = false
)

type ChunksArrayIO interface {
	Read() ([]inodedb.FileChunk, error)
	Write(cs []inodedb.FileChunk) error
	Close() error
}

type INodeDBChunksArrayIO struct {
	db    inodedb.DBHandler
	nlock inodedb.NodeLock
	fn    inodedb.FileNodeView
}

var _ = ChunksArrayIO(&INodeDBChunksArrayIO{})

func NewINodeDBChunksArrayIO(db inodedb.DBHandler, nlock inodedb.NodeLock, fn inodedb.FileNodeView) (*INodeDBChunksArrayIO, error) {
	if nlock.Ticket == inodedb.NoTicket {
		return nil, fmt.Errorf("NewINodeDBChunksArrayIO requires valid node lock. Use NewReadOnlyINodeDBChunksArrayIO if need read access")
	}

	return &INodeDBChunksArrayIO{db: db, nlock: nlock, fn: fn}, nil
}

func (caio *INodeDBChunksArrayIO) Read() ([]inodedb.FileChunk, error) {
	if caio.nlock.Ticket != inodedb.NoTicket {
		return caio.fn.GetChunks(), nil
	}

	v, _, err := caio.db.QueryNode(caio.nlock.ID, false)
	if err != nil {
		return nil, err
	}

	fn, ok := v.(inodedb.FileNodeView)
	if !ok {
		return nil, fmt.Errorf("Target node view is not a file.")
	}

	return fn.GetChunks(), nil
}

func (caio *INodeDBChunksArrayIO) Write(cs []inodedb.FileChunk) error {
	if caio.nlock.Ticket == inodedb.NoTicket {
		return fmt.Errorf("No ticket lock is acquired.")
	}

	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.UpdateChunksOp{NodeLock: caio.nlock, Chunks: cs},
	}}
	if _, err := caio.db.ApplyTransaction(tx); err != nil {
		return fmt.Errorf("Failed to apply tx for updating cs: %v", err)
	}
	return nil
}

func (caio *INodeDBChunksArrayIO) Close() error {
	if caio.nlock.Ticket != inodedb.NoTicket {
		if err := caio.db.UnlockNode(caio.nlock); err != nil {
			return err
		}
	}
	return nil
}

type ChunkedFileIO struct {
	bs blobstore.RandomAccessBlobStore
	c  Cipher

	caio ChunksArrayIO

	newChunkIO func(blobstore.BlobHandle, Cipher) blobstore.BlobHandle
}

func NewChunkedFileIO(bs blobstore.RandomAccessBlobStore, c Cipher, caio ChunksArrayIO) *ChunkedFileIO {
	return &ChunkedFileIO{
		bs: bs,
		c:  c,

		caio: caio,

		newChunkIO: func(bh blobstore.BlobHandle, c Cipher) blobstore.BlobHandle { return NewChunkIO(bh, c) },
	}
}

func (cfio *ChunkedFileIO) OverrideNewChunkIOForTesting(newChunkIO func(blobstore.BlobHandle, Cipher) blobstore.BlobHandle) {
	cfio.newChunkIO = newChunkIO
}

func (cfio *ChunkedFileIO) newFileChunk(newo int64) (inodedb.FileChunk, error) {
	bpath, err := blobstore.GenerateNewBlobPath(cfio.bs)
	if err != nil {
		return inodedb.FileChunk{}, err
	}
	fc := inodedb.FileChunk{Offset: newo, Length: 0, BlobPath: bpath}
	fmt.Printf("new chunk %v\n", fc)
	return fc, nil
}

func (cfio *ChunkedFileIO) PWrite(offset int64, p []byte) error {
	remo := offset
	remp := p
	if len(remp) == 0 {
		return nil
	}

	cs, err := cfio.caio.Read()
	if err != nil {
		return fmt.Errorf("Failed to read cs array: %v", err)
	}
	csUpdated := false

	writeToChunk := func(c *inodedb.FileChunk, isNewChunk bool, maxChunkLen int64) error {
		if !blobstore.IsReadWriteAllowed(cfio.bs.Flags()) {
			return EPERM
		}

		flags := blobstore.O_RDWR
		if isNewChunk {
			flags |= blobstore.O_CREATE | blobstore.O_EXCL
		}
		bh, err := cfio.bs.Open(c.BlobPath, flags)
		if err != nil {
			return fmt.Errorf("Failed to open path \"%s\" for writing (isNewChunk: %t): %v", c.BlobPath, isNewChunk, err)
		}
		cio := cfio.newChunkIO(bh, cfio.c)

		coff := remo - c.Offset
		n := IntMin(len(remp), int(maxChunkLen-coff))
		if n < 0 {
			return nil
		}
		if err := cio.PWrite(coff, remp[:n]); err != nil {
			return err
		}
		if err := cio.Close(); err != nil {
			return err
		}
		oldLength := c.Length
		c.Length = int64(cio.Size())
		if oldLength != c.Length {
			csUpdated = true
		}

		remo += int64(n)
		remp = remp[n:]
		return nil
	}

	for i := 0; i < len(cs); i++ {
		c := &cs[i]
		if c.Left() > remo {
			// Insert a new chunk @ i

			// try best to align offset at ChunkSplitSize
			newo := remo / ChunkSplitSize * ChunkSplitSize
			maxlen := int64(ChunkSplitSize)
			if i > 0 {
				prev := cs[i-1]
				pright := prev.Right()
				if newo < pright {
					maxlen -= pright - newo
					newo = pright
				}
			}
			if i < len(cs)-1 {
				next := cs[i+1]
				if newo+maxlen > next.Left() {
					maxlen = next.Left() - newo
				}
			}

			newc, err := cfio.newinodedb.FileChunk(newo)
			if err != nil {
				return err
			}
			cs = append(cs, inodedb.FileChunk{})
			copy(cs[i+1:], cs[i:])
			cs[i] = newc
			csUpdated = true

			if err := writeToChunk(&newc, NewChunk, maxlen); err != nil {
				return err
			}
			if len(remp) == 0 {
				return nil
			}

			continue
		}

		// Write to the chunk
		maxlen := int64(ChunkSplitSize)
		if i < len(cs)-1 {
			next := cs[i+1]
			if c.Left()+maxlen > next.Left() {
				maxlen = next.Left() - c.Left()
			}
		}
		if err := writeToChunk(c, ExistingChunk, maxlen); err != nil {
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

		if len(cs) > 0 {
			last := cs[len(cs)-1]
			lastRight := last.Right()
			if newo < lastRight {
				maxlen -= lastRight - newo
				newo = lastRight
			}
		}

		newc, err := cfio.newinodedb.FileChunk(newo)
		if err != nil {
			return err
		}
		if err := writeToChunk(&newc, NewChunk, maxlen); err != nil {
			return err
		}

		cs = append(cs, newc)
		csUpdated = true
	}

	if csUpdated {
		if err := cfio.caio.Write(cs); err != nil {
			return fmt.Errorf("Failed to write updated cs array: %v", err)
		}
	}

	return nil
}

func (cfio *ChunkedFileIO) PRead(offset int64, p []byte) error {
	remo := offset
	remp := p

	if offset < 0 {
		return fmt.Errorf("negative offset %d given", offset)
	}

	cs := cfio.cs
	// fmt.Printf("cs: %v\n", cs)
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

		if !blobstore.IsReadAllowed(cfio.bs.Flags()) {
			return EPERM
		}

		bh, err := cfio.bs.Open(c.BlobPath, blobstore.O_RDONLY)
		if err != nil {
			return fmt.Errorf("Failed to open path \"%s\" for reading: %v", c.BlobPath, err)
		}
		cio := cfio.newChunkIO(bh, cfio.c)

		n := Int64Min(int64(len(p)), c.Length-coff)
		if err := cio.PRead(coff, remp[:n]); err != nil {
			return err
		}
		if err := cio.Close(); err != nil {
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

func (cfio *ChunkedFileIO) Size() int64 {
	cs := cfio.cs
	if len(cs) == 0 {
		return 0
	}
	return cs[len(cs)-1].Right()
}

func (cfio *ChunkedFileIO) Close() error {
	return nil
}

func (cfio *ChunkedFileIO) Truncate(size int64) error {
	if !blobstore.IsReadWriteAllowed(cfio.bs.Flags()) {
		return EPERM
	}

	cs, err := cfio.caio.Read()
	if err != nil {
		return fmt.Errorf("Failed to read cs array: %v", err)
	}

	for i := len(cs) - 1; i >= 0; i-- {
		c := &cs[i]

		if c.Left() >= size {
			// drop the chunk
			continue
		}

		if c.Right() > size {
			// trim the chunk
			chunksize := size - c.Left()

			bh, err := cfio.bs.Open(c.BlobPath, blobstore.O_RDWR)
			if err != nil {
				return err
			}
			cio := cfio.newChunkIO(bh, cfio.c)
			if err := cio.Truncate(chunksize); err != nil {
				return err
			}
			if err := cio.Close(); err != nil {
				return err
			}
			c.Length = int64(cio.Size())
		}

		cs = cs[:i+1]
		if err := cfio.caio.Write(cs); err != nil {
			return fmt.Errorf("Failed to write updated cs array: %v", err)
		}
		return nil
	}
	if err := cfio.caio.Write([]inodedb.FileChunk{}); err != nil {
		return fmt.Errorf("Failed to write updated cs array (empty): %v", err)
	}
	return nil
}

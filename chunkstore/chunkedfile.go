package chunkstore

import (
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

var ChunkSplitSize int64 = 256 * 1024 * 1024 // 256MB

const (
	NewChunk      = true
	ExistingChunk = false
)

type ChunksArrayIO interface {
	Read() ([]inodedb.FileChunk, error)
	Write(cs []inodedb.FileChunk) error
}

type SimpleDBChunksArrayIO struct {
	Cs []inodedb.FileChunk
}

var _ = ChunksArrayIO(&SimpleDBChunksArrayIO{})

func NewSimpleDBChunksArrayIO() *SimpleDBChunksArrayIO {
	return &SimpleDBChunksArrayIO{make([]inodedb.FileChunk, 0)}
}

func (caio *SimpleDBChunksArrayIO) Read() ([]inodedb.FileChunk, error) {
	return caio.Cs, nil
}

func (caio *SimpleDBChunksArrayIO) Write(cs []inodedb.FileChunk) error {
	caio.Cs = cs
	return nil
}

func (caio *SimpleDBChunksArrayIO) Close() error { return nil }

type ChunkedFileIO struct {
	bs blobstore.RandomAccessBlobStore
	c  *btncrypt.Cipher

	caio       ChunksArrayIO
	newChunkIO func(blobstore.BlobHandle, *btncrypt.Cipher, int64) blobstore.BlobHandle

	origFilename string

	cs []inodedb.FileChunk

	cachedBh          blobstore.BlobHandle
	cachedCio         blobstore.BlobHandle
	cachedCioBlobpath string
}

func NewChunkedFileIO(bs blobstore.RandomAccessBlobStore, c *btncrypt.Cipher, caio ChunksArrayIO) *ChunkedFileIO {
	cfio := &ChunkedFileIO{
		bs: bs,
		c:  c,

		caio: caio,

		origFilename: "<unknown>",

		cs: nil,

		cachedBh:          nil,
		cachedCio:         nil,
		cachedCioBlobpath: "",
	}
	cfio.newChunkIO = func(bh blobstore.BlobHandle, c *btncrypt.Cipher, offset int64) blobstore.BlobHandle {
		return NewChunkIOWithMetadata(
			bh, c,
			ChunkHeader{OrigFilename: cfio.origFilename, OrigOffset: offset},
		)
	}

	cs, err := cfio.caio.Read()
	if err != nil {
		// FIXME!
		zap.S().Errorf("Failed to read cs array: %v", err)
		return nil
	}
	cfio.cs = cs

	return cfio
}

func (cfio *ChunkedFileIO) OverrideNewChunkIOForTesting(newChunkIO func(blobstore.BlobHandle, *btncrypt.Cipher, int64) blobstore.BlobHandle) {
	cfio.newChunkIO = newChunkIO
}

func (cfio *ChunkedFileIO) SetOrigFilename(name string) { cfio.origFilename = name }

func (cfio *ChunkedFileIO) newFileChunk(newo int64) (inodedb.FileChunk, error) {
	bpath, err := blobstore.GenerateNewBlobPath(cfio.bs)
	if err != nil {
		return inodedb.FileChunk{}, fmt.Errorf("Failed to generate new blobpath: %v", err)
	}
	fc := inodedb.FileChunk{Offset: newo, Length: 0, BlobPath: bpath}
	zap.S().Debugf("new chunk %+v", fc)
	return fc, nil
}

type ChunkLenUpdatedType bool

const (
	ChunkLenNotUpdated ChunkLenUpdatedType = false
	ChunkLenUpdated    ChunkLenUpdatedType = true
)

func (cfio *ChunkedFileIO) closeCachedChunkIO() error {
	var ret error

	if cfio.cachedCio != nil {
		if err := cfio.cachedCio.Close(); err != nil {
			ret = multierr.Append(ret, fmt.Errorf("Failed to close previously cached cio (blobpath: \"%s\"): %v", cfio.cachedCioBlobpath, err))
		}

		cfio.cachedCio = nil
	}
	if cfio.cachedBh != nil {
		if err := cfio.cachedBh.Close(); err != nil {
			ret = multierr.Append(ret, fmt.Errorf("Failed to close previously cached bh (blobpath: \"%s\"): %v", cfio.cachedCioBlobpath, err))
		}

		cfio.cachedBh = nil
	}
	cfio.cachedCioBlobpath = ""

	return ret
}

func (cfio *ChunkedFileIO) openChunkIO(blobpath string, isNewChunk bool, offset int64) (blobstore.BlobHandle, error) {
	if blobpath == cfio.cachedCioBlobpath {
		if isNewChunk {
			zap.S().Panicf("isNewChunk specified, but cache hit! blobpath: %v", blobpath)
		}

		if cfio.cachedCio == nil {
			zap.S().Panicf("blobpath match, but cachedCio nil! blobpath: %v", blobpath)
		}
		return cfio.cachedCio, nil
	}

	if err := cfio.closeCachedChunkIO(); err != nil {
		return nil, err
	}

	f := fl.O_RDONLY
	if fl.IsWriteAllowed(cfio.bs.Flags()) {
		f = fl.O_RDWR
	}
	if isNewChunk {
		f |= fl.O_CREATE | fl.O_EXCL
	}

	bh, err := cfio.bs.Open(blobpath, f)
	if err != nil {
		return nil, fmt.Errorf("Failed to open path \"%s\" (flags: %v, isNewChunk: %t): %v", blobpath, fl.FlagsToString(f), isNewChunk, err)
	}

	cfio.cachedBh = bh
	cfio.cachedCio = cfio.newChunkIO(bh, cfio.c, offset)
	cfio.cachedCioBlobpath = blobpath

	if cfio.cachedCio == nil {
		zap.S().Panicf("newChunkIO result nil! blobpath: %v", blobpath)
	}
	return cfio.cachedCio, nil
}

func (cfio *ChunkedFileIO) writeToChunk(c *inodedb.FileChunk, isNewChunk bool, maxChunkLen int64, p []byte, offset int64) (int, ChunkLenUpdatedType) {
	cio, err := cfio.openChunkIO(c.BlobPath, isNewChunk, c.Left())
	if err != nil {
		zap.S().Errorf("Failed to openChunkIO: %v", err)
		return 0, ChunkLenNotUpdated
	}

	coff := offset - c.Left()
	n := util.IntMin(len(p), int(maxChunkLen-coff))
	if n < 0 {
		zap.S().Panicf("Attempt to write negative len = %d. len(p) = %d, maxChunkLen = %d, offset = %d", n, len(p), maxChunkLen, offset)
	}
	if err := cio.PWrite(p[:n], coff); err != nil {
		zap.S().Errorf("cio PWrite failed: %v", err)
		return 0, ChunkLenNotUpdated
	}
	oldLength := c.Length
	c.Length = int64(cio.Size())

	return n, oldLength != c.Length
}

func (cfio *ChunkedFileIO) PWrite(p []byte, offset int64) error {
	zap.S().Debugf("PWrite: offset=%d, len=%d", offset, len(p))

	if !fl.IsReadWriteAllowed(cfio.bs.Flags()) {
		return util.EACCES
	}

	remo := offset
	remp := p
	if len(remp) == 0 {
		return nil
	}

	needCSUpdate := false

	for i := 0; i < len(cfio.cs) && len(remp) > 0; i++ {
		c := &cfio.cs[i]
		if c.Left() > remo {
			// Insert a new chunk @ i

			// try best to align offset at ChunkSplitSize
			newo := remo / ChunkSplitSize * ChunkSplitSize
			maxlen := int64(ChunkSplitSize)
			if i > 0 {
				prev := cfio.cs[i-1]
				pright := prev.Right()
				if newo < pright {
					maxlen -= pright - newo
					newo = pright
				}
			}
			if i < len(cfio.cs)-1 {
				next := cfio.cs[i+1]
				if newo+maxlen > next.Left() {
					maxlen = next.Left() - newo
				}
			}

			newc, err := cfio.newFileChunk(newo)
			if err != nil {
				return err
			}
			cfio.cs = append(cfio.cs, inodedb.FileChunk{})
			copy(cfio.cs[i+1:], cfio.cs[i:])
			cfio.cs[i] = newc
			needCSUpdate = true

			n, _ := cfio.writeToChunk(&newc, NewChunk, maxlen, remp, remo)
			remo += int64(n)
			remp = remp[n:]
			continue
		}

		// Write to the chunk
		maxlen := int64(ChunkSplitSize)
		if i < len(cfio.cs)-1 {
			next := cfio.cs[i+1]
			if c.Left()+maxlen > next.Left() {
				maxlen = next.Left() - c.Left()
			}
		}

		cRight := c.Left() + maxlen
		if cRight < remo {
			continue
		}

		n, updated := cfio.writeToChunk(c, ExistingChunk, maxlen, remp, remo)
		if updated == ChunkLenUpdated {
			needCSUpdate = true
		}
		remo += int64(n)
		remp = remp[n:]
	}

	for len(remp) > 0 {
		// Append a new chunk at the end
		newo := remo / ChunkSplitSize * ChunkSplitSize
		maxlen := int64(ChunkSplitSize)

		if len(cfio.cs) > 0 {
			last := cfio.cs[len(cfio.cs)-1]
			lastRight := last.Right()
			if newo < lastRight {
				maxlen -= lastRight - newo
				newo = lastRight
			}
		}

		newc, err := cfio.newFileChunk(newo)
		if err != nil {
			return err
		}

		n, _ := cfio.writeToChunk(&newc, NewChunk, maxlen, remp, remo)
		remo += int64(n)
		remp = remp[n:]

		cfio.cs = append(cfio.cs, newc)
		needCSUpdate = true
	}

	if needCSUpdate {
		if err := cfio.caio.Write(cfio.cs); err != nil {
			zap.S().Errorf("Failed to write updated cs array: %v", err)
		}
	}
	return nil
}

func (cfio *ChunkedFileIO) readFromChunk(c inodedb.FileChunk, p []byte, coff int64) (int64, error) {
	cio, err := cfio.openChunkIO(c.BlobPath, false, c.Left())
	if err != nil {
		return 0, err
	}

	//zap.S().Debugf("cio: %p, len(p) = %d, c.Length = %d, coff = %d", cio, len(p), c.Length, coff)
	n := util.Int64Min(int64(len(p)), c.Length-coff)
	if err := cio.PRead(p[:n], coff); err != nil {
		return 0, fmt.Errorf("cio PRead failed: %v", err)
	}

	return n, nil
}

func (cfio *ChunkedFileIO) ReadAt(p []byte, offset int64) (int, error) {
	remo := offset
	remp := p

	if offset < 0 {
		return 0, fmt.Errorf("negative offset %d given", offset)
	}

	if !fl.IsReadAllowed(cfio.bs.Flags()) {
		return 0, util.EACCES
	}

	// zap.S().Debugf("cs: %v\n", cs)
	for i := 0; i < len(cfio.cs) && len(remp) > 0; i++ {
		c := cfio.cs[i]
		if c.Left() > remo+int64(len(remp)) {
			break
		}
		if c.Right() <= remo {
			continue
		}

		coff := remo - c.Left()
		if coff < 0 {
			// Fill gap with zero
			n := util.Int64Min(int64(len(remp)), -coff)
			for j := int64(0); j < n; j++ {
				remp[j] = 0
			}
			remo += n
			coff = 0
			if len(remp) == 0 {
				return int(remo - offset), nil
			}
		}

		n, err := cfio.readFromChunk(c, remp, coff)
		if err != nil {
			return int(remo - offset), fmt.Errorf("readFromChunk failed. offset %d chunk offset %d len %d cfio.cs +%v. err: %v", remo, coff, n, cfio.cs, err)
		}

		remo += n
		remp = remp[n:]
	}

	// zap.S().Debugf("cfio.cs: %+v", cfio.cs)
	return int(remo - offset), nil
}

func (cfio *ChunkedFileIO) Size() int64 {
	if len(cfio.cs) == 0 {
		return 0
	}
	return cfio.cs[len(cfio.cs)-1].Right()
}

func (cfio *ChunkedFileIO) Close() error {
	return cfio.closeCachedChunkIO()
}

func (cfio *ChunkedFileIO) Truncate(size int64) error {
	if !fl.IsReadWriteAllowed(cfio.bs.Flags()) {
		return util.EACCES
	}

	for i := len(cfio.cs) - 1; i >= 0; i-- {
		c := &cfio.cs[i]

		if c.Left() >= size {
			// drop the chunk
			continue
		}

		if c.Right() > size {
			// trim the chunk
			chunksize := size - c.Left()

			cio, err := cfio.openChunkIO(c.BlobPath, false, c.Left())
			if err != nil {
				return err
			}
			if err := cio.Truncate(chunksize); err != nil {
				return err
			}
			c.Length = int64(cio.Size())
		}

		cfio.cs = cfio.cs[:i+1]
		goto updateAndExit
	}
	cfio.cs = []inodedb.FileChunk{}

updateAndExit:
	if err := cfio.caio.Write(cfio.cs); err != nil {
		return fmt.Errorf("Failed to write updated cfio.cs array (empty): %v", err)
	}
	return nil
}

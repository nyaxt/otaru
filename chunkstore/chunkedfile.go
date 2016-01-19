package chunkstore

import (
	"fmt"
	"syscall"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

const EPERM = syscall.Errno(syscall.EPERM)

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

	cachedBh          blobstore.BlobHandle
	cachedCio         blobstore.BlobHandle
	cachedCioBlobpath string
}

func NewChunkedFileIO(bs blobstore.RandomAccessBlobStore, c *btncrypt.Cipher, caio ChunksArrayIO) *ChunkedFileIO {
	cio := &ChunkedFileIO{
		bs: bs,
		c:  c,

		caio: caio,

		origFilename: "<unknown>",

		cachedBh:          nil,
		cachedCio:         nil,
		cachedCioBlobpath: "",
	}
	cio.newChunkIO = func(bh blobstore.BlobHandle, c *btncrypt.Cipher, offset int64) blobstore.BlobHandle {
		return NewChunkIOWithMetadata(
			bh, c,
			ChunkHeader{OrigFilename: cio.origFilename, OrigOffset: offset},
		)
	}
	return cio
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
	logger.Debugf(mylog, "new chunk %+v", fc)
	return fc, nil
}

type ChunkLenUpdatedType bool

const (
	ChunkLenNotUpdated ChunkLenUpdatedType = false
	ChunkLenUpdated    ChunkLenUpdatedType = true
)

func (cfio *ChunkedFileIO) closeCachedChunkIO() error {
	if cfio.cachedCio != nil {
		if err := cfio.cachedCio.Close(); err != nil {
			return fmt.Errorf("Failed to close previously cached cio (blobpath: \"%s\"): %v", cfio.cachedCioBlobpath, err)
		}

		cfio.cachedCio = nil
	}
	if cfio.cachedBh != nil {
		if err := cfio.cachedBh.Close(); err != nil {
			return fmt.Errorf("Failed to close previously cached bh (blobpath: \"%s\"): %v", cfio.cachedCioBlobpath, err)
		}

		cfio.cachedBh = nil
	}
	return nil
}

func (cfio *ChunkedFileIO) openChunkIO(blobpath string, isNewChunk bool, offset int64) (blobstore.BlobHandle, error) {
	if blobpath == cfio.cachedCioBlobpath {
		if isNewChunk {
			logger.Panicf(mylog, "isNewChunk specified, but cache hit! blobpath: %v", blobpath)
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

	return cfio.cachedCio, nil
}

func (cfio *ChunkedFileIO) writeToChunk(c *inodedb.FileChunk, isNewChunk bool, maxChunkLen int64, p []byte, offset int64) (int, ChunkLenUpdatedType) {
	cio, err := cfio.openChunkIO(c.BlobPath, isNewChunk, c.Left())
	if err != nil {
		logger.Criticalf(mylog, "Failed to openChunkIO: %v", err)
		return 0, ChunkLenNotUpdated
	}

	coff := offset - c.Left()
	n := util.IntMin(len(p), int(maxChunkLen-coff))
	if n < 0 {
		logger.Panicf(mylog, "Attempt to write negative len = %d. len(p) = %d, maxChunkLen = %d, offset = %d", n, len(p), maxChunkLen, offset)
	}
	if err := cio.PWrite(p[:n], coff); err != nil {
		logger.Criticalf(mylog, "cio PWrite failed: %v", err)
		return 0, ChunkLenNotUpdated
	}
	oldLength := c.Length
	c.Length = int64(cio.Size())

	return n, oldLength != c.Length
}

func (cfio *ChunkedFileIO) PWrite(p []byte, offset int64) error {
	logger.Debugf(mylog, "PWrite: offset=%d, len=%d", offset, len(p))

	if !fl.IsReadWriteAllowed(cfio.bs.Flags()) {
		return EPERM
	}

	remo := offset
	remp := p
	if len(remp) == 0 {
		return nil
	}

	cs, err := cfio.caio.Read()
	if err != nil {
		return fmt.Errorf("Failed to read cs array: %v", err)
	}
	//logger.Debugf(mylog, "cs: %v", cs)

	needCSUpdate := false

	for i := 0; i < len(cs) && len(remp) > 0; i++ {
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

			newc, err := cfio.newFileChunk(newo)
			if err != nil {
				return err
			}
			cs = append(cs, inodedb.FileChunk{})
			copy(cs[i+1:], cs[i:])
			cs[i] = newc
			needCSUpdate = true

			n, _ := cfio.writeToChunk(&newc, NewChunk, maxlen, remp, remo)
			remo += int64(n)
			remp = remp[n:]
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

		if len(cs) > 0 {
			last := cs[len(cs)-1]
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

		cs = append(cs, newc)
		needCSUpdate = true
	}

	if needCSUpdate {
		if err := cfio.caio.Write(cs); err != nil {
			logger.Criticalf(mylog, "Failed to write updated cs array: %v", err)
		}
	}
	return nil
}

func (cfio *ChunkedFileIO) readFromChunk(c inodedb.FileChunk, p []byte, coff int64) (int64, error) {
	cio, err := cfio.openChunkIO(c.BlobPath, false, c.Left())
	if err != nil {
		return 0, err
	}

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

	cs, err := cfio.caio.Read()
	if err != nil {
		return 0, fmt.Errorf("Failed to read cs array: %v", err)
	}

	if !fl.IsReadAllowed(cfio.bs.Flags()) {
		return 0, EPERM
	}

	// logger.Debugf(mylog, "cs: %v\n", cs)
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
			return int(remo - offset), fmt.Errorf("readFromChunk failed. offset %d chunk offset %d len %d cs +%v. err: %v", remo, coff, n, cs, err)
		}

		remo += n
		remp = remp[n:]
	}

	// logger.Debugf(mylog, "cs: %+v", cs)
	return int(remo - offset), nil
}

func (cfio *ChunkedFileIO) Size() int64 {
	cs, err := cfio.caio.Read()
	if err != nil {
		logger.Criticalf(mylog, "Failed to read cs array: %v", err)
		return 0
	}
	if len(cs) == 0 {
		return 0
	}
	return cs[len(cs)-1].Right()
}

func (cfio *ChunkedFileIO) Close() error {
	return cfio.closeCachedChunkIO()
}

func (cfio *ChunkedFileIO) Truncate(size int64) error {
	if !fl.IsReadWriteAllowed(cfio.bs.Flags()) {
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

			cio, err := cfio.openChunkIO(c.BlobPath, false, c.Left())
			if err != nil {
				return err
			}
			if err := cio.Truncate(chunksize); err != nil {
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

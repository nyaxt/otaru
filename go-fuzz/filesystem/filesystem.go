// +build gofuzz
package filesystem

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/chunkstore"
	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/filewritecache"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("otaru-fuzz-filesystem")

func init() { tu.EnsureLogger() }

const (
	InvalidInput     = -1
	NeutralInput     = 0
	InterestingInput = 1
)

type cmdpack struct {
	OpType    uint8
	FileIndex uint8
	Offset    uint32
	OpLen     uint32
}

const (
	OpOpenFile uint8 = iota
	OpCloseFile
	OpWrite
	OpRead
	OpTruncate
	OpMAX
)

func Fuzz(data []byte) int {
	chunkstore.ChunkSplitSize = 1 * 1024 * 1024
	filewritecache.MaxPatches = 2
	filewritecache.MaxPatchContentLen = 16 * 1024
	const AbsoluteMaxLen uint32 = 4 * 1024 * 1024

	snapshotio := inodedb.NewSimpleDBStateSnapshotIO()
	txio := inodedb.NewSimpleDBTransactionLogIO()
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		panic(err)
	}

	bs := blobstore.NewMemBlobStore()
	fs := otaru.NewFileSystem(idb, bs, tu.TestCipher())

	const NumFs = 2
	const NumFHs = NumFs * 2
	fhs := make([]*otaru.FileHandle, NumFHs)

	iobuf := make([]byte, AbsoluteMaxLen)

	reader := bytes.NewBuffer(data)
	cmdp := cmdpack{}
	for n := byte(0); true; n++ {
		if err := binary.Read(reader, binary.BigEndian, &cmdp); err != nil {
			if n < 4 {
				return InvalidInput
			} else {
				return NeutralInput
			}
		}
		fileidx := int(cmdp.FileIndex) % NumFHs
		fh := fhs[fileidx]
		logger.Infof(mylog, "Cmd %d FileIndex %d", n, fileidx)
		/*
			for i, _ := range iobuf {
				iobuf[i] = n
			}
		*/

		switch cmdp.OpType % OpMAX {
		case OpOpenFile:
			if fh != nil {
				fh.Close()
			}
			fp := fmt.Sprintf("/%d.txt", fileidx/2)
			isWrite := fileidx%2 == 0
			var openFlags int
			if isWrite {
				openFlags = flags.O_CREATE | flags.O_RDWR
			} else {
				openFlags = flags.O_RDONLY
			}
			fh, err := fs.OpenFileFullPath(fp, openFlags, 0666)
			if err != nil {
				if err == util.ENOENT {
					logger.Infof(mylog, "fp %v open ENOENT.", fp)
					break
				}
				panic(err)
			}
			fhs[fileidx] = fh

		case OpCloseFile:
			if fh == nil {
				logger.Infof(mylog, "fhs[%d] nil. No op.", fileidx)
				break
			}
			fh.Close()
			fhs[fileidx] = nil

		case OpWrite:
			if fh == nil {
				logger.Infof(mylog, "fhs[%d] nil. No op.", fileidx)
				break
			}

			offset := uint32(0)
			if currLen := fh.Size(); currLen > 0 {
				offset = cmdp.Offset % uint32(currLen)
			}
			opLen := cmdp.OpLen % (AbsoluteMaxLen - offset)
			if err := fh.PWrite(iobuf[:opLen], int64(offset)); err != nil {
				if err == util.EBADF {
					logger.Infof(mylog, "fhs[%d] PWrite EBADF.", fileidx)
					break
				}

				panic(err)
			}

		case OpRead:
			if fh == nil {
				logger.Infof(mylog, "fhs[%d] nil. No op.", fileidx)
				break
			}

			currLen := fh.Size()
			if currLen == 0 {
				logger.Infof(mylog, "fhs[%d] currSize 0.", fileidx)
				break
			}
			offset := cmdp.Offset % uint32(currLen)
			maxLen := uint32(currLen) - offset
			opLen := cmdp.OpLen % maxLen

			if _, err := fh.ReadAt(iobuf[:opLen], int64(offset)); err != nil {
				panic(err)
			}

		case OpTruncate:
			if fh == nil {
				logger.Infof(mylog, "fhs[%d] nil. No op.", fileidx)
				break
			}

			opLen := cmdp.OpLen % AbsoluteMaxLen
			if err := fh.Truncate(int64(opLen)); err != nil {
				if err == util.EBADF {
					logger.Infof(mylog, "fhs[%d] Truncate  EBADF.", fileidx)
					break
				}
				panic(err)
			}
		}
	}

	if err := fs.Sync(); err != nil {
		panic(err)
	}
	return NeutralInput
}

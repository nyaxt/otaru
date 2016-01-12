package chunkstore

import (
	"bytes"
	"encoding/binary"

	"github.com/nyaxt/otaru/chunkstore"
	tu "github.com/nyaxt/otaru/testutils"
)

func init() { tu.EnsureLogger() }

const (
	InvalidInput     = -1
	NeutralInput     = 0
	InterestingInput = 1
)

var testLockManager = chunkstore.NewLockManager()

type cmdpack struct {
	IsWrite uint8
	Offset  uint32
	OpLen   uint32
}

func Fuzz(data []byte) int {
	chunkstore.ChunkSplitSize = 1 * 1024 * 1024
	const AbsoluteMaxLen uint32 = 1 * 1024 * 1024

	caio := chunkstore.NewSimpleDBChunksArrayIO()
	fbs := tu.TestFileBlobStore()
	cfio := chunkstore.NewChunkedFileIO(fbs, tu.TestCipher(), testLockManager, caio)

	currLen := uint32(0)

	reader := bytes.NewBuffer(data)
	cmdp := cmdpack{}
	iobuf := make([]byte, AbsoluteMaxLen)
	for n := byte(0); true; n++ {
		if err := binary.Read(reader, binary.BigEndian, &cmdp); err != nil {
			return InvalidInput
		}

		isWrite := (cmdp.IsWrite & 1) == 1
		offset := uint32(0)
		if currLen > 0 {
			offset = cmdp.Offset % currLen
		}
		for i, _ := range iobuf {
			iobuf[i] = n
		}
		if isWrite {
			opLen := cmdp.OpLen % (AbsoluteMaxLen - offset)
			if err := cfio.PWrite(iobuf[:opLen], int64(offset)); err != nil {
				panic(err)
			}
			if currLen < offset+opLen {
				currLen = offset + opLen
			}
		} else {
			maxLen := currLen - offset
			if maxLen == 0 {
				return InvalidInput
			}
			opLen := cmdp.OpLen % maxLen

			if _, err := cfio.ReadAt(iobuf[:opLen], int64(offset)); err != nil {
				panic(err)
			}
		}
	}
	return NeutralInput
}

package filewritecache

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/filewritecache"
	"github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/logger"
	tu "github.com/nyaxt/otaru/testutils"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("otaru-fuzz-filewritecache")

func init() { tu.EnsureLogger() }

const (
	InvalidInput     = -1
	NeutralInput     = 0
	InterestingInput = 1
)

type ReadAtAdaptor struct {
	bh blobstore.BlobHandle
}

func (a ReadAtAdaptor) ReadAt(p []byte, offset int64) (int, error) {
	n := len(p)

	currLen := a.bh.Size()
	if offset+int64(len(p)) > currLen {
		zoff := util.Int64Max(currLen-offset, 0)
		logger.Debugf(mylog, "offset: %d, len(p): %d, currLen: %d, zoff: %d", offset, len(p), currLen, zoff)
		z := p[zoff:]
		for i, _ := range z {
			z[i] = 0
		}
		p = p[:zoff]
	}
	if len(p) == 0 {
		return n, nil
	}

	if err := a.bh.PRead(p, offset); err != nil {
		return 0, err
	}
	return n, nil
}

type cmdpack struct {
	IsWrite uint8
	Offset  uint32
	OpLen   uint32
}

func Fuzz(data []byte) int {
	filewritecache.MaxPatches = 4
	filewritecache.MaxPatchContentLen = 16
	const AbsoluteMaxLen uint32 = 32

	bs := blobstore.NewMemBlobStore()
	bh, err := bs.Open("hoge", flags.O_RDWRCREATE)
	if err != nil {
		panic(err)
	}

	wc := filewritecache.New()

	currLen := uint32(0)

	cmdReader := bytes.NewBuffer(data)
	cmdp := cmdpack{}
	wbuf := make([]byte, AbsoluteMaxLen)
	rbuf := make([]byte, AbsoluteMaxLen)
	mirror := make([]byte, AbsoluteMaxLen)
	for n := byte(0); true; n++ {
		if err := binary.Read(cmdReader, binary.BigEndian, &cmdp); err != nil {
			if n < 4 {
				return InvalidInput
			} else {
				return NeutralInput
			}
		}
		logger.Infof(mylog, "Cmd %d %+v", n, cmdp)

		isWrite := (cmdp.IsWrite & 1) == 1
		if isWrite {
			offset := cmdp.Offset % AbsoluteMaxLen
			opLen := cmdp.OpLen % (AbsoluteMaxLen - offset)

			w := wbuf[:opLen]
			for i, _ := range w {
				w[i] = n
			}
			logger.Debugf(mylog, "PWrite offset %d opLen %d currLen %d", offset, opLen, currLen)
			if err := wc.PWrite(w[:opLen], int64(offset)); err != nil {
				panic(err)
			}
			if wc.NeedsSync() {
				logger.Debugf(mylog, "NeedsSync!")
				if err := wc.Sync(bh); err != nil {
					panic(err)
				}
			}
			copy(mirror[offset:offset+opLen], w)

			if currLen < offset+opLen {
				currLen = offset + opLen
			}
		} else {
			if currLen == 0 {
				return InvalidInput
			}
			offset := cmdp.Offset % currLen
			maxLen := currLen - offset
			if maxLen == 0 {
				return InvalidInput
			}
			opLen := cmdp.OpLen % maxLen

			r := rbuf[:opLen]
			adaptor := ReadAtAdaptor{bh}
			logger.Debugf(mylog, "ReadAtThrough offset %d opLen %d currLen %d", offset, opLen, currLen)
			if _, err := wc.ReadAtThrough(r, int64(offset), adaptor); err != nil {
				panic(err)
			}
			r2 := mirror[offset : offset+opLen]
			if !bytes.Equal(r, r2) {
				logger.Warningf(mylog, "mismatch!!! | wc     := %+v", r)
				logger.Warningf(mylog, "mismatch!!! | mirror := %+v", r2)
				panic(errors.New("mismatch"))
			}
		}
	}
	{
		r := rbuf[:currLen]
		adaptor := ReadAtAdaptor{bh}
		if _, err := wc.ReadAtThrough(r, 0, adaptor); err != nil {
			panic(err)
		}
		r2 := mirror[:currLen]
		if !bytes.Equal(r, r2) {
			logger.Warningf(mylog, "mismatch!!! | wc     := %+v", r)
			logger.Warningf(mylog, "mismatch!!! | mirror := %+v", r2)
			panic(errors.New("mismatch"))
		}
	}
	return NeutralInput
}

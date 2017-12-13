package cachedblobstore

import (
	"fmt"

	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/util"
)

type CachedBlobHandle struct {
	be         *CachedBlobEntry
	flags      int
	stacktrace []byte
}

func (bh *CachedBlobHandle) Flags() int { return bh.flags }

func (bh *CachedBlobHandle) PRead(p []byte, offset int64) error {
	if !fl.IsReadAllowed(bh.flags) {
		return util.EACCES
	}

	return bh.be.PRead(p, offset)
}

func (bh *CachedBlobHandle) PWrite(p []byte, offset int64) error {
	if !fl.IsWriteAllowed(bh.flags) {
		return util.EACCES
	}

	return bh.be.PWrite(p, offset)
}

func (bh *CachedBlobHandle) Size() int64 {
	return bh.be.Size()
}

func (bh *CachedBlobHandle) Truncate(newsize int64) error {
	if !fl.IsWriteAllowed(bh.flags) {
		return util.EACCES
	}

	return bh.be.Truncate(newsize)
}

var _ = util.Syncer(&CachedBlobHandle{})

func (bh *CachedBlobHandle) Sync() error {
	if !fl.IsWriteAllowed(bh.flags) {
		return nil
	}

	return bh.be.Sync()
}

func (bh *CachedBlobHandle) Close() error {
	bh.be.CloseHandle(bh)

	return nil
}

func (bh *CachedBlobHandle) String() string {
	return fmt.Sprintf("CachedBlobHandle{be.blobpath: \"%s\", %v, stack: %v}", bh.be.blobpath, fl.FlagsToString(bh.flags), string(bh.stacktrace))
}

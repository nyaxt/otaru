package fuse

import (
	"os"

	bfuse "github.com/nyaxt/fuse"

	"github.com/nyaxt/otaru/filesystem"
	oflags "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
)

func otaruSetattr(fs *filesystem.FileSystem, id inodedb.ID, req *bfuse.SetattrRequest) error {
	var valid filesystem.ValidAttrFields
	var a filesystem.Attr

	if req.Valid.Uid() {
		valid |= filesystem.UidValid
		a.Uid = req.Uid
	}
	if req.Valid.Gid() {
		valid |= filesystem.GidValid
		a.Gid = req.Gid
	}
	if req.Valid.Mode() {
		valid |= filesystem.PermModeValid
		a.PermMode = uint16(req.Mode & os.ModePerm)
	}
	if req.Valid.Atime() {
		// otaru fs doesn't keep atime. set mtime instead.
		valid |= filesystem.ModifiedTValid
		a.ModifiedT = req.Atime
	}
	if req.Valid.Mtime() {
		valid |= filesystem.ModifiedTValid
		a.ModifiedT = req.Mtime
	}

	if valid != 0 {
		if err := fs.SetAttr(id, a, valid); err != nil {
			return err
		}
	}

	return nil
}

func Bazil2OtaruFlags(bf bfuse.OpenFlags) int {
	ret := 0
	if bf.IsReadOnly() {
		ret = oflags.O_RDONLY
	} else if bf.IsWriteOnly() {
		ret = oflags.O_WRONLY
	} else if bf.IsReadWrite() {
		ret = oflags.O_RDWR
	}

	if bf&bfuse.OpenAppend != 0 {
		ret |= oflags.O_APPEND
	}
	if bf&bfuse.OpenCreate != 0 {
		ret |= oflags.O_CREATE
	}
	if bf&bfuse.OpenExclusive != 0 {
		ret |= oflags.O_EXCL
	}
	if bf&bfuse.OpenSync != 0 {
		logger.Criticalf(mylog, "FIXME: OpenSync not supported yet !!!!!!!!!!!")
	}
	if bf&bfuse.OpenTruncate != 0 {
		ret |= oflags.O_TRUNCATE
	}

	return ret
}

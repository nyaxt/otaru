package fuse

import (
	"os"

	"github.com/nyaxt/otaru"
	"github.com/nyaxt/otaru/inodedb"

	bfuse "bazil.org/fuse"
)

func otaruSetattr(fs *otaru.FileSystem, id inodedb.ID, req *bfuse.SetattrRequest) error {
	var valid otaru.ValidAttrFields
	var a otaru.Attr

	if req.Valid.Uid() {
		valid |= otaru.UidValid
		a.Uid = req.Uid
	}
	if req.Valid.Gid() {
		valid |= otaru.GidValid
		a.Gid = req.Gid
	}
	if req.Valid.Mode() {
		valid |= otaru.PermModeValid
		a.PermMode = uint16(req.Mode & os.ModePerm)
	}
	if req.Valid.Atime() {
		// otaru fs doesn't keep atime. set mtime instead.
		valid |= otaru.ModifiedTValid
		a.ModifiedT = req.Atime
	}
	if req.Valid.Mtime() {
		valid |= otaru.ModifiedTValid
		a.ModifiedT = req.Mtime
	}

	if valid != 0 {
		if err := fs.SetAttr(id, a, valid); err != nil {
			return err
		}
	}

	return nil
}

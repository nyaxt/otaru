package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

func (fs *FileSystem) FindNodeFullPath(fullpath string) (inodedb.ID, error) {
	fullpath = filepath.Clean(fullpath)
	if len(fullpath) < 1 || fullpath[0] != '/' {
		return 0, fmt.Errorf("Path must start with /, but given: %v", fullpath)
	}

	if fullpath == "/" {
		return inodedb.ID(1), nil
	}

	parentPath := filepath.Dir(fullpath)
	parentId, err := fs.FindNodeFullPath(parentPath)
	if err != nil {
		return 0, err
	}

	entries, err := fs.DirEntries(parentId)
	base := filepath.Base(fullpath)
	id, ok := entries[base]
	if !ok {
		return 0, util.ENOENT
	}

	return id, nil
}

func (fs *FileSystem) CreateFileFullPath(fullpath string, perm uint16, uid, gid uint32, modifiedT time.Time) (inodedb.ID, error) {
	perm &= 0777

	if len(fullpath) < 1 || fullpath[0] != '/' {
		return 0, fmt.Errorf("Path must start with /, but given: %v", fullpath)
	}

	dirname := filepath.Dir(fullpath)
	basename := filepath.Base(fullpath)

	dirID, err := fs.FindNodeFullPath(dirname)
	if err != nil {
		return 0, err
	}

	return fs.CreateFile(dirID, basename, uint16(perm), uid, gid, modifiedT)
}

func (fs *FileSystem) OpenFileFullPath(fullpath string, flags int, perm uint16) (*FileHandle, error) {
	perm &= 0777

	if len(fullpath) < 1 || fullpath[0] != '/' {
		return nil, fmt.Errorf("Path must start with /, but given: %v", fullpath)
	}

	dirname := filepath.Dir(fullpath)
	basename := filepath.Base(fullpath)

	dirID, err := fs.FindNodeFullPath(dirname)
	if err != nil {
		return nil, err
	}

	entries, err := fs.DirEntries(dirID)
	if err != nil {
		return nil, err
	}

	var id inodedb.ID
	id, ok := entries[basename]
	if !ok {
		if flags|os.O_CREATE != 0 {
			id, err = fs.CreateFile(dirID, basename, uint16(perm), 0, 0, time.Now())
			if err != nil {
				return nil, err
			}
		} else {
			return nil, util.ENOENT
		}
	}

	if id == 0 {
		logger.Panicf(fslog, "inode id must != 0 here, but got %v", id)
	}

	fh, err := fs.OpenFile(id, flags)
	if err != nil {
		return nil, err
	}

	return fh, nil
}

func (fs *FileSystem) WriteFile(fullpath string, content []byte, perm uint16) error {
	h, err := fs.OpenFileFullPath(fullpath, fl.O_RDWRCREATE, perm)
	if err != nil {
		return err
	}
	defer h.Close()

	return h.PWrite(content, 0)
}

func (fs *FileSystem) CreateDirFullPath(fullpath string, permmode os.FileMode) error {
	parent := filepath.Dir(fullpath)
	id, err := fs.FindNodeFullPath(parent)
	if err != nil {
		return fmt.Errorf("Failed to find parent \"%s\": %v", parent, err)
	}

	name := filepath.Base(fullpath)
	_, err = fs.CreateDir(id, name, uint16(permmode), 0, 0, time.Now())
	if err != nil {
		return fmt.Errorf("CreateDir err: %v", err)
	}
	return nil
}

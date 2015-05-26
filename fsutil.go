package otaru

/*
import (
	"fmt"
	"os"
	"path/filepath"
)

func (fs *FileSystem) OpenDirFullPath(fullpath string) (*DirHandle, error) {
	if len(fullpath) < 1 || fullpath[0] != '/' {
		return nil, fmt.Errorf("Path must start with /, but given: %v", fullpath)
	}

	if fullpath != "/" {
		panic("FIXME: implement me!!!!")
	}

	dh, err := fs.OpenDir(1)
	return dh, err
}

func (fs *FileSystem) OpenFileFullPath(fullpath string, flag int, perm os.FileMode) (*FileHandle, error) {
	perm &= os.ModePerm

	if len(fullpath) < 1 || fullpath[0] != '/' {
		return nil, fmt.Errorf("Path must start with /, but given: %v", fullpath)
	}

	dirname := filepath.Dir(fullpath)
	basename := filepath.Base(fullpath)

	dh, err := fs.OpenDirFullPath(dirname)
	if err != nil {
		return nil, err
	}

	id := INodeID(0)

	entries := dh.Entries()
	id, ok := entries[basename]
	if !ok {
		if flag|os.O_CREATE != 0 {
			// FIXME: apply perm

			id, err = dh.CreateFile(basename)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, ENOENT
		}
	}

	if id == 0 {
		panic("inode id must != 0 here!")
	}

	// FIXME: handle flag
	fh, err := fs.OpenFile(id)
	if err != nil {
		return nil, err
	}

	return fh, nil
}
*/

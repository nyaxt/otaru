package webdav

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"

	"github.com/nyaxt/otaru/third_party/webdav"

	"github.com/nyaxt/otaru"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

var mylog = logger.Registry().Category("webdav")

type webdavFile struct {
	h      *otaru.FileHandle
	offset int64
	size   int64
}

var _ = webdav.File(&webdavFile{})

func (wf *webdavFile) Close() error {
	wf.h.Close()
	wf.h = nil
	return nil
}

func (wf *webdavFile) Read(p []byte) (int, error) {
	// FIXME: not sure if this handles eof correctly
	n, err := wf.h.ReadAt(p, wf.offset)
	wf.offset += int64(n)
	return n, err
}

func (wf *webdavFile) Write(p []byte) (int, error) {
	return 0, util.EACCES
}

func (wf *webdavFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if offset < 0 || wf.size < offset {
			return wf.offset, os.ErrInvalid
		}
		wf.offset = offset
		break
	case io.SeekCurrent:
		logger.Panicf(mylog, "No implemention for Seek(whence=io.SeekCurrent), as we don't expect net/http to use this")
		break
	case io.SeekEnd:
		wf.offset = wf.size
		break
	default:
		return wf.offset, os.ErrInvalid
	}

	return wf.offset, nil
}

func (wf *webdavFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

type webdavFileInfo struct {
	name string
	attr *otaru.Attr
}

var _ = os.FileInfo(&webdavFileInfo{})

func (fi *webdavFileInfo) Name() string {
	if fi.name != "" {
		return fi.name
	}
	return filepath.Base(fi.attr.OrigPath)
}
func (fi *webdavFileInfo) Size() int64        { return fi.attr.Size }
func (fi *webdavFileInfo) Mode() os.FileMode  { return os.FileMode(fi.attr.PermMode) }
func (fi *webdavFileInfo) ModTime() time.Time { return fi.attr.ModifiedT }
func (fi *webdavFileInfo) IsDir() bool        { return fi.attr.Type == inodedb.DirNodeT }
func (fi *webdavFileInfo) Sys() interface{}   { return nil }

func (wf *webdavFile) Stat() (os.FileInfo, error) {
	attr, err := wf.h.Attr()
	if err != nil {
		return nil, err
	}
	return &webdavFileInfo{name: "", attr: &attr}, nil
}

type webdavDir struct {
	fi          os.FileInfo
	childrenFIs []os.FileInfo
	offset      int
}

var _ = webdav.File(&webdavDir{})

func (wd *webdavDir) Close() error { return nil }

func (wd *webdavDir) Read(p []byte) (int, error) {
	return 0, os.ErrInvalid
}

func (wd *webdavDir) Write(p []byte) (int, error) {
	return 0, os.ErrInvalid
}

func (wd *webdavDir) Seek(offset64 int64, whence int) (int64, error) {
	offset := int(offset64)
	switch whence {
	case io.SeekStart:
		if offset < 0 || len(wd.childrenFIs) < offset {
			return int64(wd.offset), os.ErrInvalid
		}
		wd.offset = offset
		break
	case io.SeekCurrent:
		logger.Panicf(mylog, "No implemention for Seek(whence=io.SeekCurrent), as we don't expect net/http to use this")
		break
	case io.SeekEnd:
		wd.offset = len(wd.childrenFIs)
		break
	default:
		return int64(wd.offset), os.ErrInvalid
	}

	return int64(wd.offset), nil
}

func (wd *webdavDir) Readdir(count int) ([]os.FileInfo, error) {
	old := wd.offset
	if old >= len(wd.childrenFIs) {
		// The os.File Readdir docs say that at the end of a directory,
		// the error is io.EOF if count > 0 and nil if count <= 0.
		if count > 0 {
			return nil, io.EOF
		}
		return nil, nil
	}
	if count > 0 {
		wd.offset += count
		if wd.offset > len(wd.childrenFIs) {
			wd.offset = len(wd.childrenFIs)
		}
	} else {
		wd.offset = len(wd.childrenFIs)
		old = 0
	}
	return wd.childrenFIs[old:wd.offset], nil
}

func (wd *webdavDir) Stat() (os.FileInfo, error) {
	return wd.fi, nil
}

type webdavFS struct {
	ofs *otaru.FileSystem
}

var _ = webdav.FileSystem(webdavFS{})

func (fs webdavFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return util.EACCES
}

func (fs webdavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	logger.Debugf(mylog, "OpenFile name=\"%v\"", name)

	id, err := fs.ofs.FindNodeFullPath(name)
	if err != nil {
		return nil, err
	}

	isDir, err := fs.ofs.IsDir(id)
	if err != nil {
		return nil, err
	}
	if !isDir {
		h, err := fs.ofs.OpenFile(id, fl.O_RDONLY)
		if err != nil {
			return nil, err
		}

		size := h.Size()
		f := &webdavFile{h: h, offset: 0, size: size}

		return f, nil
	} else {
		entries, err := fs.ofs.DirEntries(id)
		if err != nil {
			return nil, err
		}

		attr, err := fs.ofs.Attr(id)
		if err != nil {
			return nil, err
		}
		fi := &webdavFileInfo{name: filepath.Base(name), attr: &attr}

		// net/webdav doesn't need ".", ".."
		childrenFIs := make([]os.FileInfo, 0, len(entries))
		for name, id := range entries {
			attr, err := fs.ofs.Attr(id)
			if err != nil {
				return nil, err
			}

			childrenFIs = append(childrenFIs, &webdavFileInfo{name: name, attr: &attr})
		}

		d := &webdavDir{fi: fi, childrenFIs: childrenFIs, offset: 0}
		return d, nil
	}
}

func (fs webdavFS) RemoveAll(ctx context.Context, name string) error {
	return util.EACCES
}

func (fs webdavFS) Rename(ctx context.Context, oldName, newName string) error {
	return util.EACCES
}

func (fs webdavFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	logger.Debugf(mylog, "Stat name=\"%v\"", name)

	id, err := fs.ofs.FindNodeFullPath(name)
	if err != nil {
		return nil, err
	}

	attr, err := fs.ofs.Attr(id)
	if err != nil {
		return nil, err
	}
	return &webdavFileInfo{name: filepath.Base(name), attr: &attr}, nil
}

func Serve(ofs *otaru.FileSystem) error {
	handler := &webdav.Handler{
		Prefix:     "",
		FileSystem: webdavFS{ofs},
		LockSystem: webdav.NewMemLS(),
		Logger: func(req *http.Request, err error) {
			logger.Debugf(mylog, "req: %v, err: %v", req, err)
		},
	}
	httpsrv := http.Server{
		Addr:    ":8005",
		Handler: handler,
	}
	return httpsrv.ListenAndServe()
}

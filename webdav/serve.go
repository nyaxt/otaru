package webdav

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/webdav"

	"github.com/nyaxt/otaru"
	fl "github.com/nyaxt/otaru/flags"
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
	attr *otaru.Attr
}

var _ = os.FileInfo(&webdavFileInfo{})

func (fi *webdavFileInfo) Name() string       { return filepath.Base(fi.attr.OrigPath) }
func (fi *webdavFileInfo) Size() int64        { return fi.attr.Size }
func (fi *webdavFileInfo) Mode() os.FileMode  { return os.FileMode(fi.attr.PermMode) }
func (fi *webdavFileInfo) ModTime() time.Time { return fi.attr.ModifiedT }
func (fi *webdavFileInfo) IsDir() bool        { return false }
func (fi *webdavFileInfo) Sys() interface{}   { return nil }

func (wf *webdavFile) Stat() (os.FileInfo, error) {
	attr, err := wf.h.Attr()
	if err != nil {
		return nil, err
	}
	return &webdavFileInfo{&attr}, nil
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

	h, err := fs.ofs.OpenFileFullPath("/fuse.txt", fl.O_RDONLY, 0444)
	if err != nil {
		return nil, err
	}
	size := h.Size()
	f := &webdavFile{h: h, offset: 0, size: size}
	return f, nil
}

func (fs webdavFS) RemoveAll(ctx context.Context, name string) error {
	return util.EACCES
}

func (fs webdavFS) Rename(ctx context.Context, oldName, newName string) error {
	return util.EACCES
}

func (fs webdavFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
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

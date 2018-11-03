// Code generated by vfsgen; DO NOT EDIT.

package json

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	pathpkg "path"
	"time"
)

// Assets statically implements the virtual filesystem provided to vfsgen.
var Assets = func() http.FileSystem {
	fs := vfsgen۰FS{
		"/": &vfsgen۰DirInfo{
			name:    "/",
			modTime: time.Date(2018, 7, 18, 7, 15, 2, 285973597, time.UTC),
		},
		"/otaru.swagger.json": &vfsgen۰CompressedFileInfo{
			name:             "otaru.swagger.json",
			modTime:          time.Date(2018, 11, 3, 22, 57, 59, 104775396, time.UTC),
			uncompressedSize: 19519,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5c\xdd\x6e\xdb\xc6\x12\xbe\xf7\x53\x2c\x78\xce\xc5\x39\x80\x11\x3a\x41\xd1\x0b\x03\x01\x9a\x38\x76\x20\xc0\x68\x53\xbb\x48\x2f\x8a\x80\x58\x91\x43\x6a\x93\xe5\x2e\xb3\x3b\xb4\xa3\xc6\x7a\xf7\x62\x49\x4a\xa2\xf8\x23\xfe\xdb\x4c\x93\x8b\x00\x8a\xc8\x19\x7e\xf3\xed\xcc\xec\xcc\x70\xe5\xaf\x27\x84\x58\xfa\x9e\x06\x01\x28\xeb\x9c\x58\x2f\x9e\x9d\x59\xa7\xe6\x3b\x26\x7c\x69\x9d\x13\x73\x9d\x10\x0b\x19\x72\x30\xd7\x7f\x43\xaa\x62\xf2\xea\xdd\x22\xb9\x8b\x10\xeb\x0e\x94\x66\x52\x98\x6b\xcf\x33\x59\x42\x2c\x57\x0a\xa4\x2e\xee\x14\x10\x62\x09\x1a\xe6\x34\x44\x4a\x7e\x04\x17\xb3\xfb\x09\xb1\x62\xc5\xcd\xd5\x15\x62\xa4\xcf\x6d\x3b\x60\xb8\x8a\x97\xcf\x5c\x19\xda\x62\x4d\xbf\xa0\x2d\x8d\xd8\xfe\x76\x08\x29\x4b\x04\x62\x10\xf2\x97\xe4\x16\x8d\x10\x19\x01\x2b\xb9\x67\x73\x42\xc8\x26\xb1\x44\xbb\x2b\x08\x41\x5b\xe7\xe4\xaf\x14\x5c\xf2\x0c\x73\xd7\x87\xe4\xba\x2b\x85\x8e\x0f\x6e\xa0\x51\xc4\x99\x4b\x91\x49\x61\x7f\xd4\x52\xec\xef\x8d\x94\xf4\x62\xb7\xe5\xbd\x14\x57\x7a\x4f\xa1\x4d\x23\x66\xdf\x3d\xb7\x97\x5c\x2e\x35\x4a\x05\xb6\x2b\x85\xcf\x82\x3c\x47\x01\xe4\x29\x23\xc4\x92\x11\xa8\x44\xf7\xc2\x33\xc6\xbe\x05\xbc\x48\x85\x4e\xf7\xf7\x28\xd0\x91\x14\x1a\xf4\x81\x28\x21\xd6\x8b\xb3\xb3\xc2\x57\x84\x58\x1e\x68\x57\xb1\x08\xb3\x35\x7b\x45\x74\xec\xba\xa0\xb5\x1f\x73\xb2\xd5\xf4\x2c\xa7\x3e\x11\x4a\x28\xa4\x25\x65\x84\x58\xff\x55\xe0\x1b\x3d\xff\xb1\x3d\xf0\x99\x60\x46\xaf\xb6\xa3\xe5\x5b\xc0\xd7\x5b\x43\x53\xc8\x37\x99\x72\xeb\x40\xc5\xe6\xa4\xea\xf3\x26\x67\x1e\xd2\x60\x4f\x77\xf6\xdd\x4e\xf5\x2d\xa8\x3b\xe6\xe6\x74\x7e\x38\xc9\xeb\xca\xf4\x54\x70\x0f\x02\x15\x3b\xa0\xac\x0d\xf9\x97\x99\xd4\x37\xc0\x7e\x06\x75\x5e\xac\x2b\x30\xd1\xe3\xb8\xd4\x5d\x41\x9e\xfa\x48\xea\xe3\xdc\xdf\x24\x82\x17\x89\xdc\xdc\xc9\xcf\x61\xed\xcb\x7e\x44\x15\x0d\x01\x41\x15\xd7\xa0\x60\xd1\x36\xa7\x2e\xa5\xb7\x2e\x02\x67\xa2\xee\x8a\x82\xcf\x31\x53\x60\x88\x45\x15\xc3\xb8\x06\x7f\x8e\x41\x63\x1b\x7b\x3f\x4c\xe4\x6d\x3e\xe3\xa0\xd7\x1a\x21\xb4\x29\xa2\xb2\xbf\x32\x6f\xd3\x25\xcc\x5f\x21\xaa\xd9\xfb\x98\x01\xf9\x58\xce\xc5\xbc\x22\xde\x82\x91\x0b\x9f\x3c\x30\xef\x81\xbc\x7c\x49\xce\x4e\x09\xae\x40\x90\xcf\x31\xa8\x35\x31\x3b\x60\xc9\xd8\xd4\x2f\xcd\xa5\x6e\x7e\x89\xeb\x28\x81\xa3\x51\x31\x11\x14\x65\x7d\xa9\x42\x8a\x49\x41\xc0\x04\xfe\xfc\x53\x9e\x94\xcd\x69\xb3\x91\x55\x78\x52\xa4\x89\x29\x47\xa0\xfa\x94\xeb\x06\xac\xbd\xfc\xff\x8a\x71\xb8\x4d\xfc\xb8\x7f\x00\x98\x8f\x9d\x03\xe0\x06\xa8\x67\x1e\x3e\xfb\x20\xd8\x02\x7d\xba\x40\x98\xa7\x2f\x4b\xdf\xd7\x80\x93\x79\xf3\xc8\x68\x39\x88\x60\x8a\xd8\x63\x02\xc1\x34\x35\xb5\x70\xcb\x68\x47\x8d\xcd\x5d\xab\x12\xc5\xc7\xe3\xed\x4f\xc5\x10\xbe\x89\x80\xdb\x21\xfd\x11\x71\x87\x38\x9f\xb6\xfe\xca\x2d\xcb\x68\xd5\xd7\x38\xbb\x8f\xf0\x7e\x95\x1e\x74\xd9\x7c\xae\x32\x99\xab\x98\xf3\x77\x87\xab\x3c\xcf\x98\x28\x02\x7e\xac\xd0\xf8\x77\x16\x2c\xbc\x53\x47\x7e\xcd\x34\xbe\x61\xf3\xaf\xd6\x33\x9c\xb3\x2a\xd8\x39\x88\xff\x31\xef\xff\xdd\xaa\xf6\x21\xae\x45\x95\xa2\xe5\x24\x89\x10\x16\x97\x8a\x34\xe6\xeb\x86\x8c\x5d\xe0\xf4\xbb\xad\xfe\x45\x21\xf7\x36\x4e\x59\x2e\x14\x50\x9c\x7f\x19\x92\xc2\xfc\x3e\x66\x2b\x5b\x5b\x67\xb5\xb1\x1b\xc7\xb2\x55\xd8\x71\x82\x17\xca\xbb\xf9\xfb\x56\x0a\xf3\xfb\xf0\xad\xad\xad\x4f\xef\x5b\xcc\x78\x94\xb7\xb4\x35\x52\xec\x3a\x93\x5f\x98\xea\xeb\xcd\xeb\xdb\x44\x74\xee\xee\x55\xc0\x3b\xe6\x74\x7e\xab\xb7\x1b\xf3\x5c\x06\x01\x28\xdb\xa5\x08\x81\xec\xf1\x46\xe4\x62\x2f\xf8\x0d\x70\xbf\x47\x3b\x26\xf3\xd7\x09\x87\x83\x88\x5f\xdb\x5f\xb7\x9f\x36\x9d\xd2\xea\xed\xce\xa8\xf5\xec\x17\x20\x87\xf5\xb1\x12\xac\x5b\xe6\x86\x8c\x38\x46\xe8\x58\x66\x3e\x4e\xba\x3f\x3e\x00\x3b\x3e\x02\xeb\x9b\xf9\x87\x84\x00\xa7\x08\x1a\x1d\x2e\x03\x07\x04\xaa\xb5\xc3\xbc\x8e\x49\xe8\x3a\xd1\x70\x2d\x83\x4b\x23\xbf\xf0\x66\x1f\x09\x15\x98\xe7\x92\x90\xb8\x0c\x3a\xed\x01\xbf\x9b\x06\xe9\xda\x08\xcd\x9d\xf4\x1d\xd2\xc7\x4a\x3e\x21\x13\x4e\xdd\x04\xf3\x91\x27\xdb\x2d\x92\xd3\xf1\x54\xf9\x64\x9d\x7f\xfd\x1a\xb5\x79\xb7\xc0\x42\x36\xc1\x8b\x90\x29\x5f\x2d\xf4\x8a\xdd\xac\x2f\x3b\x38\xb3\xd5\x2e\x75\xa6\xe5\xfa\xc2\x08\xce\x3d\x7e\xf7\x50\xc7\xcc\x95\x7b\xad\xfd\x38\xdf\x9f\x7f\xeb\x42\xfb\xfb\x4c\x6a\xee\x9c\x67\x38\x9f\x86\xf0\xdd\x09\xbe\x1c\xa6\xfd\x79\xba\xc2\x50\xd5\xfc\xd7\xe4\x8b\xdc\x3a\x6c\xc3\x55\x2e\x0f\x8f\x1a\x46\xca\x2c\x08\xb2\x02\xd5\x96\xc7\xd4\x61\xed\x41\x2a\xf2\x51\x9e\xd2\x23\x53\xc8\x3c\x0b\x49\x55\x53\xa7\xb6\x9c\x1c\x6b\x52\x63\xed\x1a\x25\x8d\xdf\x7b\x06\xf7\xd5\x09\x67\x53\xe9\xc5\x85\x23\x24\x03\x58\xab\xb4\xae\x03\xd6\x3a\x7c\x87\x93\xaf\xa7\x5d\xd6\x83\xab\x15\xc3\xf4\xf4\x11\x07\xb3\x74\x34\xf0\xc9\x83\xd9\x89\x1e\x08\xd5\xc4\x8f\x39\x4f\x27\xeb\x95\x3e\x92\xed\x58\xc7\x20\x56\xca\xc5\xf5\x96\x55\xed\x52\xf5\x7b\x54\x5e\x69\x30\x85\xd2\x08\x54\xe8\x84\x87\x23\xe9\x91\x54\x87\xd2\x63\x3e\x03\xcf\x41\xd6\xc8\x62\x77\xed\x99\x82\xf6\xfe\xfd\x87\x11\x68\xed\xdf\xc3\x23\x70\x8a\x9c\xc5\xb4\x23\xe0\xbe\x4e\xef\x52\x4a\x0e\x54\xd4\x29\xde\x5e\x6e\xe4\xa0\xf6\xe5\xe9\x1c\xd8\xa8\xc1\x7c\xe4\x64\xf3\x00\xd4\x4b\xea\x7e\x02\xe1\x39\x2c\x8c\xb8\xd3\x37\x19\x6c\x95\xf8\x9c\x06\xa5\x32\xa2\x8d\x82\xe4\x60\xee\x30\x0c\xa9\x8a\x6e\x08\xea\x99\xae\x18\xd8\x0d\x20\x79\xd7\xe2\x4c\xb6\x17\xa7\xd5\xfb\x6e\x22\xd7\x69\x43\xae\x38\xb4\x3d\xf6\xb6\x3c\x9e\xa1\x65\xb0\xc9\x1c\x63\xa8\xc5\x97\x05\xd4\xdd\xe3\x88\xcb\xa5\x93\xcc\xf5\x7a\xf8\xae\x46\x8a\xfd\x02\xcf\x3c\x95\x83\x18\x7d\xef\xb9\xa3\x9c\x79\x93\x68\xd6\x6b\xe1\x3a\xae\x8c\x05\x8e\xae\x9a\x53\x8d\x4e\xac\xa1\x67\x22\x6e\xd2\x7c\xaf\x58\xe3\x2a\xf5\x54\x6d\x48\x19\x5d\xb3\x88\xc3\x25\x28\x47\xfa\x29\x72\xe5\xac\xa8\xf0\x78\xb9\xcf\x1b\xf1\x41\xa3\x3e\xa1\x3e\x78\x2b\x5f\x65\x0d\x08\xde\xe9\xd6\x20\xd1\x8c\x5f\xa6\xd1\x3b\x45\xfd\x55\x1e\x2b\x8c\xa4\x38\x65\x82\xb9\x9f\xa0\x67\xe0\x1f\x53\xbe\xf7\x40\x21\x3d\x70\xb8\x74\x3f\xd5\x3a\x61\xc7\x72\xbf\xde\x0b\x6b\x27\xe9\x93\x14\x91\xe3\xc0\xde\xf7\x09\x15\x20\x0b\xfc\x5b\x20\xe2\xf0\x60\x8a\x62\x5d\x2d\xae\x2f\xf3\xb3\xa3\x37\x8b\x9b\xed\x33\x77\xf3\x45\xcb\x03\x9f\xc6\x3c\x41\x95\xdc\x5f\x05\x21\x69\xc5\xe7\x50\x6c\x8f\xd0\x0a\x0f\xed\xd3\xf2\x9b\x23\xfb\x7b\xfc\x1d\xe6\x47\xab\x3e\x4d\xab\x5e\x13\x62\xc5\x43\x97\x43\xf6\xa5\xd2\x6c\x91\x8c\x59\x4d\xd7\x4c\x32\x73\xa2\xcd\xa5\x74\xa1\xf1\x98\xb0\x49\x3a\x16\x82\x1c\xee\x80\x4f\x9b\x39\xcb\x6f\xf1\xe6\xdb\x24\x95\xb0\xf6\xe8\x91\x6a\x74\xcc\x61\x6f\xcb\x2f\xbc\xac\x8d\x8e\xe3\xc3\x82\x79\xf8\x5a\x7e\x17\x99\x62\x84\xc8\x65\xfa\x17\x02\x86\x4e\x44\x4a\xbf\x78\x1b\xd2\x29\x4b\xaf\x89\xf7\xba\xf1\xde\x1a\x5b\xcc\x37\x2b\x7e\x15\x3c\x64\x88\xaf\xd6\x8e\x8a\x6b\x09\xec\x3c\x90\xcc\x2d\x8e\x07\x9a\x29\xf0\x9c\x16\x9b\x7e\x37\xa3\x87\xaf\x51\xf6\x52\x6f\x0a\xab\x41\x29\xa9\x9c\x10\xb4\xa6\xc1\x08\x66\xe7\x4f\x92\xfe\x78\x57\xd3\x9e\xb1\x46\x1f\x29\xc8\x55\x9d\xa1\x6b\x2f\x5c\x7e\x83\x3e\x60\xad\x02\xe9\xb4\xeb\x52\x2b\x49\x94\xbd\x06\xd4\x54\xb9\xbd\x26\x7b\x22\x0e\x9d\x40\x2a\x19\x23\x13\x13\x54\xb5\x2b\xa9\xb1\x6f\xdb\x12\x4d\xd1\x43\x1d\xe9\x35\xfa\x2b\x0d\x21\x74\x28\xe7\xb2\xe7\x78\xa6\x49\xb5\x5e\xf7\x1c\x58\x35\x4c\x24\x9c\xa0\x16\x70\xff\x15\x37\x7a\x7d\x6f\xe2\xe1\x46\xf1\xf4\xc5\x90\x60\x65\xe8\xb8\x32\x0c\x59\xd3\xd4\xa7\xd2\xda\x65\xcc\xb8\xe7\xac\x8a\x87\x91\x3b\x49\xb7\x28\xac\x1a\x09\x29\xfd\xd2\x75\x66\xa3\x8b\xec\x97\xf7\xa3\xeb\x9d\xba\x56\x2b\xff\xb0\xbb\x69\x4f\x39\x31\xff\x36\x27\xff\x04\x00\x00\xff\xff\x25\xb6\x60\x98\x3f\x4c\x00\x00"),
		},
	}
	fs["/"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/otaru.swagger.json"].(os.FileInfo),
	}

	return fs
}()

type vfsgen۰FS map[string]interface{}

func (fs vfsgen۰FS) Open(path string) (http.File, error) {
	path = pathpkg.Clean("/" + path)
	f, ok := fs[path]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}

	switch f := f.(type) {
	case *vfsgen۰CompressedFileInfo:
		gr, err := gzip.NewReader(bytes.NewReader(f.compressedContent))
		if err != nil {
			// This should never happen because we generate the gzip bytes such that they are always valid.
			panic("unexpected error reading own gzip compressed bytes: " + err.Error())
		}
		return &vfsgen۰CompressedFile{
			vfsgen۰CompressedFileInfo: f,
			gr: gr,
		}, nil
	case *vfsgen۰DirInfo:
		return &vfsgen۰Dir{
			vfsgen۰DirInfo: f,
		}, nil
	default:
		// This should never happen because we generate only the above types.
		panic(fmt.Sprintf("unexpected type %T", f))
	}
}

// vfsgen۰CompressedFileInfo is a static definition of a gzip compressed file.
type vfsgen۰CompressedFileInfo struct {
	name              string
	modTime           time.Time
	compressedContent []byte
	uncompressedSize  int64
}

func (f *vfsgen۰CompressedFileInfo) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("cannot Readdir from file %s", f.name)
}
func (f *vfsgen۰CompressedFileInfo) Stat() (os.FileInfo, error) { return f, nil }

func (f *vfsgen۰CompressedFileInfo) GzipBytes() []byte {
	return f.compressedContent
}

func (f *vfsgen۰CompressedFileInfo) Name() string       { return f.name }
func (f *vfsgen۰CompressedFileInfo) Size() int64        { return f.uncompressedSize }
func (f *vfsgen۰CompressedFileInfo) Mode() os.FileMode  { return 0444 }
func (f *vfsgen۰CompressedFileInfo) ModTime() time.Time { return f.modTime }
func (f *vfsgen۰CompressedFileInfo) IsDir() bool        { return false }
func (f *vfsgen۰CompressedFileInfo) Sys() interface{}   { return nil }

// vfsgen۰CompressedFile is an opened compressedFile instance.
type vfsgen۰CompressedFile struct {
	*vfsgen۰CompressedFileInfo
	gr      *gzip.Reader
	grPos   int64 // Actual gr uncompressed position.
	seekPos int64 // Seek uncompressed position.
}

func (f *vfsgen۰CompressedFile) Read(p []byte) (n int, err error) {
	if f.grPos > f.seekPos {
		// Rewind to beginning.
		err = f.gr.Reset(bytes.NewReader(f.compressedContent))
		if err != nil {
			return 0, err
		}
		f.grPos = 0
	}
	if f.grPos < f.seekPos {
		// Fast-forward.
		_, err = io.CopyN(ioutil.Discard, f.gr, f.seekPos-f.grPos)
		if err != nil {
			return 0, err
		}
		f.grPos = f.seekPos
	}
	n, err = f.gr.Read(p)
	f.grPos += int64(n)
	f.seekPos = f.grPos
	return n, err
}
func (f *vfsgen۰CompressedFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.seekPos = 0 + offset
	case io.SeekCurrent:
		f.seekPos += offset
	case io.SeekEnd:
		f.seekPos = f.uncompressedSize + offset
	default:
		panic(fmt.Errorf("invalid whence value: %v", whence))
	}
	return f.seekPos, nil
}
func (f *vfsgen۰CompressedFile) Close() error {
	return f.gr.Close()
}

// vfsgen۰DirInfo is a static definition of a directory.
type vfsgen۰DirInfo struct {
	name    string
	modTime time.Time
	entries []os.FileInfo
}

func (d *vfsgen۰DirInfo) Read([]byte) (int, error) {
	return 0, fmt.Errorf("cannot Read from directory %s", d.name)
}
func (d *vfsgen۰DirInfo) Close() error               { return nil }
func (d *vfsgen۰DirInfo) Stat() (os.FileInfo, error) { return d, nil }

func (d *vfsgen۰DirInfo) Name() string       { return d.name }
func (d *vfsgen۰DirInfo) Size() int64        { return 0 }
func (d *vfsgen۰DirInfo) Mode() os.FileMode  { return 0755 | os.ModeDir }
func (d *vfsgen۰DirInfo) ModTime() time.Time { return d.modTime }
func (d *vfsgen۰DirInfo) IsDir() bool        { return true }
func (d *vfsgen۰DirInfo) Sys() interface{}   { return nil }

// vfsgen۰Dir is an opened dir instance.
type vfsgen۰Dir struct {
	*vfsgen۰DirInfo
	pos int // Position within entries for Seek and Readdir.
}

func (d *vfsgen۰Dir) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && whence == io.SeekStart {
		d.pos = 0
		return 0, nil
	}
	return 0, fmt.Errorf("unsupported Seek in directory %s", d.name)
}

func (d *vfsgen۰Dir) Readdir(count int) ([]os.FileInfo, error) {
	if d.pos >= len(d.entries) && count > 0 {
		return nil, io.EOF
	}
	if count <= 0 || count > len(d.entries)-d.pos {
		count = len(d.entries) - d.pos
	}
	e := d.entries[d.pos : d.pos+count]
	d.pos += count
	return e, nil
}

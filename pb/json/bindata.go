// Code generated by go-bindata.
// sources:
// src/otaru.swagger.json
// DO NOT EDIT!

package json

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"os"
	"time"
	"io/ioutil"
	"path/filepath"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name string
	size int64
	mode os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _otaruSwaggerJson = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xcc\x58\x4d\x8f\xa3\x46\x13\xbe\xfb\x57\x8c\xfc\xbe\xc7\xd1\x32\xbb\x8a\x72\xd8\x53\x92\xc9\x24\x1a\x69\xa5\x44\x3b\x52\x2e\xd1\x08\x35\x50\xe0\xde\x34\xdd\x6c\x7f\x78\xe3\xac\xfc\xdf\x53\xdd\x06\x1b\x30\x60\xd3\xe0\x0d\x87\x5d\x79\xa8\xaa\x87\xaa\xa7\xeb\xa3\x8b\xaf\xab\xbb\xbb\xb5\xfa\x42\xb2\x0c\xe4\xfa\xfd\xdd\xfa\xdd\x9b\x87\xf5\xbd\x7d\x46\x79\x2a\xf0\x81\x95\xe3\x5f\x9a\x6a\x06\x56\xfe\x9b\x26\xd2\xdc\xfd\xf8\xfb\xb3\xd3\x42\xc9\x16\xa4\xa2\x82\x5b\xd9\xdb\xd2\x16\x9f\xc6\x82\x6b\x12\xeb\x23\x00\x3e\xe2\x24\xaf\x21\x14\x52\x7c\x02\x54\xb8\xaf\xc4\x46\x32\x2b\xdd\x68\x5d\xa8\xf7\x41\x90\x51\xbd\x31\xd1\x9b\x58\xe4\x01\xdf\x91\xbf\x75\x20\xac\xd9\x49\x1d\x72\x42\x9d\x81\x01\x2e\x7e\x70\x2a\x4a\x43\x61\x0d\xd6\x4e\x67\x8f\xff\xef\x5d\x24\x2a\xde\x40\x0e\x0a\x95\xff\x3c\x38\xe7\xde\x61\xb5\x5e\x9d\x1c\x7d\x55\xa6\xa1\x40\x8a\x82\xd1\x98\x68\x8c\x2b\xf8\xa4\x30\xb8\xa3\x2e\xba\x9d\x98\xf8\x4a\x5d\xa2\x37\xea\x44\x61\x40\x0a\x1a\x6c\xdf\x06\x11\x13\x91\xd2\x42\x42\x80\x2f\x4e\x69\x56\xe7\x28\x83\x3a\x65\xf8\x40\x14\x20\x1d\xf6\x73\x62\x83\xfd\x15\xf4\xe3\xc1\xe8\xfe\xa4\x23\x41\x15\x18\x02\xa8\x86\x29\x0a\xde\x3d\x3c\xb4\x1e\xe1\xc3\x04\x54\x2c\x69\xa1\xcb\x33\xab\x01\x39\xb1\x23\x8b\x9c\x99\xa1\xe4\xff\x12\x52\x6b\xf1\xbf\x20\x81\x94\x72\x6a\x11\x54\x50\x44\xe8\xd3\x4f\x55\x48\x07\xe7\x3e\x96\x0e\xad\x1b\x10\xfb\x55\xd7\xef\x7d\x2d\x10\x4d\xb2\x13\xb1\xe5\xb3\x23\xf4\x0b\xc8\x2d\x8d\x6b\x98\xaf\xab\x3a\x56\x89\xd3\xc1\x32\x70\x2d\x69\x83\x9c\x6b\x68\x7e\x2a\xad\x16\xc5\x73\xe9\xd4\xb2\xf8\x95\x60\x2b\x22\x8c\x09\x46\x54\x27\xb9\x10\x6a\x98\xe5\x8f\xce\xf0\xd1\xd9\x2d\x87\xe6\x9a\x57\xbe\x3c\x17\x44\x62\xaf\xd3\xd8\x18\x5b\x6c\xb7\x7c\xaf\x3a\x62\x24\x92\x5d\xdb\x71\xca\xfb\x24\x12\x3e\x1b\x8a\xa4\xa3\x5c\x4b\x03\xf3\x06\xfc\xd9\x00\x9e\xda\x15\xf1\xbe\xde\x28\xaf\x52\xca\x40\xed\xb0\x95\xe7\x01\x1b\x55\xb3\x1f\xa8\xd2\x3f\x53\xb9\xa0\x4c\x2a\x3d\xfa\x56\x59\x64\xc7\x4d\x77\x16\xe1\xa1\xca\xa1\x34\x4a\x09\x53\xed\x3c\xd2\xbb\xc2\xa1\x2a\xec\x38\x3c\x5b\x7b\xe5\xc0\x2f\x78\x96\x2f\xee\x2c\x47\x26\x01\x13\xf6\x3a\x12\xe0\x54\x85\x4c\x78\x34\xef\xc7\x93\xe1\x72\xd2\xa1\xe1\xd7\x9c\x2d\xfc\x83\x63\x6b\x12\xc5\xbb\xe0\x6b\xf5\x6b\x3f\xaa\x87\xbf\x1c\x83\xda\x2d\x88\xea\x9a\x57\xdf\xaa\xfa\xe2\x73\x16\x9c\xfc\x50\x81\x5d\xb5\x39\xdc\xc7\x07\xea\xef\x7e\x29\x13\xa5\xf2\x91\x72\x8c\x1d\x64\x0b\x16\x15\x52\x21\x73\xa2\x4b\x95\xef\xbf\x9b\x63\xae\x78\x25\x7b\x39\x50\x1a\x2b\xcd\x75\x9d\xe4\xd0\xbe\x9e\xad\xe1\x82\xd2\xfb\xe8\xd4\x9c\x6d\xe4\x84\xea\xc7\xee\x69\x11\x1c\x43\xf0\x1f\xa5\xd5\x72\xd8\x2d\x3d\xfa\x6f\xa8\x5d\x55\x4b\x6b\xcd\xa7\xd3\x0a\x39\xb8\x6c\xd5\x78\xaf\x0a\x53\x44\xcd\x1d\x1b\xb7\x57\x3c\x00\x4d\x5b\xd4\xae\x23\x12\xff\x05\x3c\x09\x69\x5e\xb0\xb0\xec\x21\x0d\xe6\xfb\xba\x51\x3d\xf0\x0a\x24\x65\x07\x06\x46\x03\xb8\x0d\x62\x9a\x0f\x07\x88\x71\x1e\x74\xe6\x76\xdf\xb8\x9e\x40\xf2\x71\x42\xf4\x38\x46\xa4\x24\xcd\xd6\xbc\xa6\x98\x35\xed\x40\x86\x2e\x9c\xae\x37\x1e\xe7\x71\x77\xaa\xf6\x87\xdb\xde\x2e\x27\xc4\x6a\x77\xee\x1b\x06\x7a\xee\xec\x93\x7b\xe1\xc4\x88\x9f\x5a\x5e\x8f\xaf\x23\x2c\xcd\xd0\xcd\x7a\x8f\xdc\x55\x1a\x0f\xce\xab\xf0\xec\x5b\x19\xf0\x0b\xb6\x0d\xc2\x7b\x27\x73\x1d\x78\x4b\x18\x4d\x6e\x82\xac\x76\x3c\x0e\x63\x61\xb8\x9e\x1d\x9a\x11\xa5\x43\xa3\xdc\x6d\xe6\x06\xc8\x5f\x24\xbd\x78\x4a\x9e\xd0\x96\x94\xd9\x91\xb9\xc9\x23\x90\xa1\x48\x0f\x9e\xcb\x70\x43\x78\xc2\xce\x67\xeb\x8c\x2f\x9a\xf5\x0d\x3d\xc5\xdb\xde\xad\x97\xdb\xab\x5a\x9e\x7a\x34\xaa\x4e\x84\x09\x01\x53\xcf\xd2\x30\x43\xa7\xef\x39\xb0\x4b\x9d\xf1\xcd\x92\xfe\x33\x7f\x15\x9a\x7e\x66\xba\x56\x9c\xeb\x40\xb3\x5b\x80\xe2\xb9\xe6\x61\x2e\x92\x5e\x0e\xfc\xa1\x11\x95\xa6\x14\x92\x50\xd3\x8b\x47\x3a\xb1\x86\x9b\xd7\x95\x1b\x5e\xad\x86\x32\x89\xc1\x16\xd8\x4c\x34\xf6\x04\xda\xf1\x85\x75\x42\xb0\x89\xdc\x85\xd2\xf4\xce\xe4\x48\x08\x06\x84\xf7\xb9\x5c\x89\x3b\xb9\xc0\x5d\xca\x7e\x07\x08\xaf\xa8\xae\x71\x41\x4f\xef\xd2\xca\xc4\x31\xa8\xde\xa1\x32\x25\x6a\x90\x52\xc8\x30\x47\x74\x92\x4d\x0e\xbb\xeb\xd3\x53\x7f\xd8\x6d\xe3\xf3\xc5\x7e\x02\x65\x99\x08\xcf\x77\xf2\xe1\xa0\x6a\xac\x08\xaf\x1d\x8e\xc8\xd8\xeb\xf2\x8b\xb7\x87\x10\x39\x13\x46\x53\x7e\x83\xa6\xb6\x11\x4a\xfb\x8e\xa8\xe2\x16\xf3\x72\x60\xd4\xf8\x83\xe6\x90\x87\x84\x31\xe1\x79\x8b\xbc\x04\xad\x76\x9e\x77\xba\xc1\x8b\x83\x3d\xf8\x5e\x87\xfd\x4f\xdc\xe2\xa6\x49\xaf\xc3\xf3\x34\xf6\xf6\xa7\xa2\x29\xc5\x4a\x35\x2e\x44\x79\x4e\x2f\x6d\x44\x9d\xd1\x46\x86\xb2\x24\xdc\xb4\xbf\xd6\x8f\xb2\xbe\x62\xde\xf7\x11\xb2\xb2\xff\xf6\xab\x7f\x03\x00\x00\xff\xff\x41\x98\xce\x41\x4c\x22\x00\x00")

func otaruSwaggerJsonBytes() ([]byte, error) {
	return bindataRead(
		_otaruSwaggerJson,
		"otaru.swagger.json",
	)
}

func otaruSwaggerJson() (*asset, error) {
	bytes, err := otaruSwaggerJsonBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "otaru.swagger.json", size: 8780, mode: os.FileMode(420), modTime: time.Unix(1512207944, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if (err != nil) {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"otaru.swagger.json": otaruSwaggerJson,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"otaru.swagger.json": &bintree{otaruSwaggerJson, map[string]*bintree{
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
        data, err := Asset(name)
        if err != nil {
                return err
        }
        info, err := AssetInfo(name)
        if err != nil {
                return err
        }
        err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
        if err != nil {
                return err
        }
        err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
        if err != nil {
                return err
        }
        err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
        if err != nil {
                return err
        }
        return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
        children, err := AssetDir(name)
        // File
        if err != nil {
                return RestoreAsset(dir, name)
        }
        // Dir
        for _, child := range children {
                err = RestoreAssets(dir, filepath.Join(name, child))
                if err != nil {
                        return err
                }
        }
        return nil
}

func _filePath(dir, name string) string {
        cannonicalName := strings.Replace(name, "\\", "/", -1)
        return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}


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

var _otaruSwaggerJson = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xec\x5a\x4b\x8f\xdb\x36\x10\xbe\xfb\x57\x2c\xdc\x1e\x17\x51\x12\x14\x3d\xe4\xd4\x66\x93\x16\x0b\x2c\xfa\xc8\x02\xed\xa1\x58\x10\xb4\x34\x92\x99\x50\xa2\x42\x52\x9b\xa8\x0b\xff\xf7\x0e\x69\xd9\x96\xf5\x5a\x3d\x5d\x02\xed\xa1\x8d\x57\xe4\x7c\xfa\xe6\xe3\x70\x38\x24\xf5\xb4\xba\xba\x5a\xab\x2f\x34\x8a\x40\xae\xdf\x5c\xad\x5f\xbf\x78\xb9\xbe\x36\xcf\x58\x12\x0a\x7c\x60\xda\xf1\x2f\xcd\x34\x07\xd3\xfe\xab\xa6\x32\xbb\xfa\xf1\xb7\x5b\xdb\x0b\x5b\x1e\x41\x2a\x26\x12\xd3\xf6\xaa\xb0\xc5\xa7\xbe\x48\x34\xf5\xf5\x11\x00\x1f\x25\x34\x2e\x21\xa4\x52\x7c\x04\xec\x70\x7d\x68\xce\x24\x37\xad\x5b\xad\x53\xf5\xc6\xf3\x22\xa6\xb7\xd9\xe6\x85\x2f\x62\x2f\xc9\xe9\x57\xed\x09\x63\x76\xea\x0e\x31\x65\xd6\x20\x83\x44\xfc\x60\xbb\x28\x0d\xa9\x31\x58\xdb\x3e\x3b\xfc\xff\xce\x7a\xa2\xfc\x2d\xc4\xa0\xb0\xf3\x5f\x7b\x72\xf6\x1d\xa6\xd7\x83\x6d\x47\xae\x2a\x3b\xeb\x40\xd3\x94\x33\x9f\x6a\xf4\xcb\xfb\xa8\xd0\xb9\x63\x5f\xa4\x1d\x64\x7e\xcf\xbe\x54\x6f\xd5\x49\x42\x8f\xa6\xcc\x7b\x7c\xe5\x6d\xb8\xd8\x28\x2d\x24\x78\xf8\xe2\x90\x45\x65\x8d\x22\x28\x4b\x86\x0f\x44\x0a\xd2\x62\xdf\x06\xc6\xd9\x9f\x41\xdf\xec\x8d\xae\x4f\x7d\x24\xa8\x14\x5d\x00\x75\x66\x8a\x0d\xaf\x5f\xbe\xac\x3c\xc2\x87\x01\x28\x5f\xb2\x54\x17\x63\x56\x02\xb2\xcd\x56\x2c\x5a\x33\xc3\x96\x6f\x25\x84\xc6\xe2\x1b\x2f\x80\x90\x25\xcc\x20\x28\x2f\xdd\x20\xa7\xb7\x07\x97\xf6\xe4\x3e\x14\x84\xd6\x67\x10\xbb\x55\xd3\xef\x5d\xc9\x11\x4d\xa3\x93\xb0\xc5\xb3\x23\xf4\x3d\xc8\x47\xe6\x97\x30\x1f\x56\x65\xac\x02\xa7\x41\x65\x48\xb4\x64\x67\xe2\xf4\x91\xf9\x7d\x61\xe5\x94\xce\x05\x29\xb7\xf4\x95\x60\x66\x04\xf1\x29\x7a\x54\x16\x39\x15\xaa\x5b\xe5\x0f\xd6\xf0\xc6\xda\xb9\x23\x73\x89\xd5\x58\x9d\x53\x2a\x31\xd7\x69\x4c\x8c\x15\xb5\x2b\xdc\x0f\x19\x71\x23\x82\xbc\x4a\x9c\x25\x6d\x2d\x12\x3e\x67\x0c\x45\xc7\x76\x2d\x33\x98\xd7\xe1\xcf\x19\xe0\xa8\xf5\xf0\xf7\x61\xa1\xb8\x0a\x19\x07\x95\x63\x2a\x8f\xed\xcf\x41\x01\x75\x23\x81\x6a\xf8\xc9\x98\xb9\x13\x4f\x27\x52\xff\x8d\x70\x2a\xfb\x3b\x5b\x34\x19\xb8\x7b\x1b\x15\xd3\xc2\xc9\x7b\x62\xc1\xee\x2c\xa6\xb2\xee\x90\xfa\x53\x32\xe7\x22\xea\xc8\xe9\x52\x01\xc5\x82\xe6\x70\x32\x05\xce\xb0\x70\xd2\x79\x6a\x11\x15\xae\x63\x49\x54\xb5\x0d\x85\x8c\xa9\xb6\x45\x1d\x4b\xf4\xf7\xdf\x95\xfd\xda\x5d\xbb\x1e\xf8\xa5\x61\x71\x2a\xee\xf9\xa0\xd2\xe7\x8e\x29\xfd\x8e\x49\x87\xc2\xbd\x60\x74\xa9\x60\x6f\x0a\xea\x7d\x10\xe1\xa8\xca\xae\x28\x0a\x29\x57\xcf\x04\xfc\x65\x83\x80\x25\x22\x80\x60\xe3\x29\x4d\xf5\xd0\xfa\xf7\xf6\x17\xb4\x7d\xf7\xf6\xde\x9a\xba\x13\x0c\x15\x66\x73\x56\xc2\x07\xdc\x61\x1a\x73\x61\x76\xce\x1e\x6e\x00\x21\x12\x23\xf6\x19\x37\x27\x43\xa7\x54\x3e\xf1\x9a\x53\xe3\x3b\xab\xd6\x24\x89\x73\xef\xe9\xf0\x6b\x37\xa8\x3a\xbc\x3f\x3a\x95\x3b\x24\x75\x89\xd5\xa5\x32\x9c\x5f\x57\xc1\xb6\xcf\xb4\xa8\x3b\xb9\x68\x1f\x38\x62\x5d\x01\xe6\xa4\xeb\xba\xda\xe1\x54\x7b\xd4\x4a\x8f\xb1\x8b\xf7\x94\x60\xe7\x38\x46\x4a\x13\xfc\x8b\x98\xf3\x8b\x9c\xb0\x60\x60\x62\xb9\xb3\x08\xc8\xc1\x1c\x1a\xe4\xb7\x81\x43\x31\xdf\xc0\xce\x95\x24\x83\xff\x0c\xca\xe0\xbf\x9b\x9a\xe0\xce\x18\xb9\x23\xef\x91\xd3\xa5\x12\x4a\xcc\x12\xd2\xb6\x47\x98\x52\x34\x35\x4f\xd6\xae\xa9\xda\x27\xe1\x74\xa7\xbf\x29\x7c\xa9\x94\xb4\x0e\x8b\x75\x5b\x35\x10\xce\xac\xea\x69\xb3\x32\x46\x3d\x7c\xe2\x2c\x66\xda\x89\x01\x58\x36\x3b\x16\x5b\x9a\xb3\xbb\x89\x7e\xe9\x70\x5f\x40\xdf\x1a\x43\x77\x66\xea\x89\xd4\x9c\xf9\xef\x84\x3a\x4e\xdd\xd3\x8d\xce\x10\x81\xff\x28\xac\xdc\x51\xb7\x60\xf4\xef\x48\xbb\x3a\xdc\x3e\x95\x38\x1d\xc9\xaf\x9b\x8e\xc9\x4a\x72\x1f\xe6\x9f\xd8\x9c\xdf\x91\xa5\xd2\xe8\xae\x59\x45\xd1\x75\xc0\xe4\x79\x81\x70\x86\xd2\x70\xd8\xd2\x71\xd4\x52\x96\xa0\x48\x2f\x5d\xa8\x8d\x76\x59\x3b\x99\xa6\x94\xd2\x9e\x50\xca\xa0\xd1\x12\xa0\x28\x67\x4c\x62\xdc\xf8\xcd\x0f\x8d\xa8\x2c\x64\x10\x10\xcd\x9e\x55\xb1\x27\x7a\xe3\xec\x6d\x3c\x62\x9e\x10\x4c\xb3\x05\x52\x0b\xdb\x8e\xfb\xc2\x09\xac\x37\xd4\xff\x04\x49\x40\x58\x9c\x72\x32\x36\x6e\x0f\x20\x21\xa7\x51\x2d\x69\xf5\x01\xb0\x97\x60\xd3\x38\xec\x21\x86\x31\x68\x57\xba\x61\x1b\x3f\x41\xe4\x63\xe9\xd4\x42\xac\x5e\x00\xb5\x94\x3f\xed\x87\x7d\xb6\x2a\x38\xee\xd3\x9b\x93\x74\xbb\xbb\xd5\x0b\xd2\x09\xbe\xda\x6d\xd7\x72\x8e\xd6\xc9\xda\x9d\xd0\x54\x8f\xdf\x57\x58\x0f\x9f\x47\x38\x35\x89\x3d\x03\x18\x11\xbb\xe6\xa0\x71\xdc\xc4\x33\x6f\xe5\x90\xcc\x93\x26\x4b\xc0\x8f\x94\xb3\x60\x11\x64\x95\x27\x3e\xf1\x45\x96\xe8\xd9\xa1\x39\xc5\xad\x7f\xa6\x60\x64\x22\x7e\x0e\xf9\x8b\xb9\xa8\x58\x06\xda\x88\x32\x3b\x72\x92\xc5\x1b\x90\x44\x84\x7b\xe6\x92\x6c\x69\x12\xf0\x7a\x55\x39\xe3\x8b\x66\x7d\x43\xfb\xe4\x6d\x3c\xca\x9e\x30\x79\x97\x1b\x03\x8b\xac\xbf\x2e\x83\xbb\x44\xe9\x5a\xdf\xc4\xcc\x04\xbc\x57\x82\x61\xa1\x30\x72\xe2\x77\x16\xdc\xc7\x08\x34\x57\x37\x84\x0b\xff\x53\x6b\x10\x0e\xac\x4c\xdb\xa3\xb0\xf5\x2c\x6e\x91\x22\x72\x1e\xda\xd5\x4b\x41\x77\x17\xfa\x0a\xd3\x11\xab\x7c\x23\x82\x0b\x05\xfe\x0c\x3b\xc5\xa2\xcf\xf0\x4a\x83\xfd\x3d\xff\x12\xf6\xff\xb6\xf5\xa2\xdb\xd6\x4a\xad\xbf\xe0\xbe\xa4\x2b\x92\x38\x3c\x02\x5f\x36\x59\xd5\x0f\xe4\xdd\x4d\x57\x35\xae\x23\x12\x56\x0b\x86\x0b\xcb\x49\x79\xe0\x45\x34\x26\x5e\x5c\x89\xb5\x72\x12\x9d\x6d\xa6\x9e\xc9\xb3\xff\xa8\xbd\xb7\x9f\x2d\xc1\xd0\xf0\x79\xe9\x94\x93\x4e\x99\x13\x99\xb5\x92\xda\x08\xc1\x81\x26\x6d\xee\x1e\x9a\x1b\x1d\x0e\x40\x99\x2b\x11\xd2\x63\x69\x19\xe6\xf4\xf4\x39\xaf\x32\xdf\x07\xd5\x5a\x09\x4e\xf1\x1a\xa4\x14\x92\xc4\x88\x4e\xa3\xc9\x6e\x37\x7d\xcc\xd0\xee\x76\xd5\xb8\x7e\x19\x32\x41\xb2\x48\x90\x7e\x5b\x80\x46\x55\xc4\xa8\xd3\x3f\x2a\xfd\x51\xc7\x26\x58\xf5\x13\xd4\x4c\x64\x9a\x25\x0b\xac\xe8\x5b\xa1\xf4\xd8\xfa\x2c\x5d\xa2\x58\xec\xa8\xb3\xc6\x83\xc6\x10\x13\xca\x31\x6b\x2d\x03\xad\xf2\x91\xa7\x01\xcf\x6c\xf7\x48\xd4\x4a\x78\xfc\x88\x1b\xdc\x30\x58\x78\xe7\x58\xbd\x5e\x9b\x32\x59\x99\x26\xbe\x88\xcd\x1d\xf6\x98\xc3\xc3\x8c\xf1\x80\x6c\xab\xdf\x7f\x0d\xb2\xee\xb1\x84\x3e\x2b\x48\xed\x03\x60\xc7\xf6\x68\x22\x0c\xd5\x12\x87\x16\xf6\x3b\xad\x51\xa8\x9b\x5c\xc3\x20\x5d\x7b\xae\x29\x2b\xf3\xdf\x6e\xf5\x4f\x00\x00\x00\xff\xff\x61\x94\xa7\x9b\xdc\x38\x00\x00")

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

	info := bindataFileInfo{name: "otaru.swagger.json", size: 14556, mode: os.FileMode(420), modTime: time.Unix(1515790091, 0)}
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

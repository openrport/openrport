// Code generated for package api_token by go-bindata DO NOT EDIT. (@generated)
// sources:
// 001_init.down.sql
// 001_init.up.sql
package api_token

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xd2\xd5\x55\x70\x09\xf2\x0f\x50\xf0\xf4\x73\x71\x8d\x50\xc8\x4c\xa9\x88\x4f\x2c\xc8\x8c\x2f\xc9\xcf\x4e\xcd\x8b\x2f\x2d\x4e\x2d\xca\x4b\xcc\x4d\x8d\x2f\x28\x4a\x4d\xcb\xac\xb0\xe6\xe2\x02\x2b\x0d\x71\x74\xf2\x71\x55\x80\x2b\xb3\x06\x04\x00\x00\xff\xff\x4e\xb1\xc5\x90\x43\x00\x00\x00")

func _001_initDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__001_initDownSql,
		"001_init.down.sql",
	)
}

func _001_initDownSql() (*asset, error) {
	bytes, err := _001_initDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "001_init.down.sql", size: 67, mode: os.FileMode(420), modTime: time.Unix(1671270782, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x91\xcd\x6e\x82\x40\x14\x85\xf7\x3c\xc5\xe9\x4a\x4d\xe5\x05\x6a\xba\xa0\x70\x1b\x89\x08\x96\x5e\xa2\xae\xc8\x14\xae\x71\xaa\x02\xe5\xa7\xc1\xb7\x6f\x10\xff\x9a\x38\xbb\xc9\x99\x6f\x72\xbe\x7b\xed\x90\x2c\x26\xb0\xf5\xe6\x11\x54\xa1\xe3\x3a\xdf\x49\x86\xa1\x01\x00\x4d\x25\x65\xa6\x0e\x02\xa6\x15\xc3\x0f\x18\x7e\xe4\x79\xb0\xa7\x64\xcf\x30\xbc\xa6\x4f\xaf\x18\x0c\x46\xe3\x13\x52\x94\xb2\xd1\xed\x63\xe0\x9c\xdd\x3f\x4f\x4a\x51\xb5\xa4\xb1\xaa\xe1\x58\x4c\xec\xce\xe9\x86\x39\xf4\x6e\x45\x1e\xc3\x8e\xc2\x90\x7c\x8e\xbb\xf4\x93\xad\xf9\x62\x8c\x13\x2c\x6d\xa1\x4b\xa9\xee\xe1\xfe\xd7\x2a\xc9\x8b\xbe\x74\x7f\xef\x9d\xfe\x75\xea\x83\x45\xe8\xce\xad\x70\x8d\x19\xad\x6f\x3e\xe3\xb3\xc4\xc8\x18\x61\xe9\xf2\x34\x88\x18\x61\xb0\x74\x9d\x09\x4c\xf3\x36\x93\xe7\x8b\xeb\xa1\xa9\x6a\x6c\xd5\xaf\x40\xa1\xc9\xf4\x4f\x23\xd8\xc9\xd1\x30\x4c\x13\xe4\x70\xe0\x04\x2f\x48\xb6\x92\xec\xa0\x37\x38\xe6\x0d\xaa\x5a\xef\xf7\xc8\x44\x52\xd4\x39\x52\x49\xf6\xaa\x14\xa8\x0c\x3a\x4b\xa5\x45\xc7\x9d\xb7\x12\xf9\xee\x47\x44\x70\x7d\x87\x56\xd0\x69\x1b\x5f\x17\x14\x5f\x6a\xc4\x7d\x89\x0e\xea\x4e\xe0\xe3\x3b\xff\xaa\x1e\xc8\x4c\xfe\x02\x00\x00\xff\xff\xad\x7d\xf4\xbd\xea\x01\x00\x00")

func _001_initUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__001_initUpSql,
		"001_init.up.sql",
	)
}

func _001_initUpSql() (*asset, error) {
	bytes, err := _001_initUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "001_init.up.sql", size: 490, mode: os.FileMode(420), modTime: time.Unix(1671615264, 0)}
	a := &asset{bytes: bytes, info: info}
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
	if err != nil {
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
	"001_init.down.sql": _001_initDownSql,
	"001_init.up.sql":   _001_initUpSql,
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
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"001_init.down.sql": &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":   &bintree{_001_initUpSql, map[string]*bintree{}},
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

// Code generated for package main by go-bindata DO NOT EDIT. (@generated)
// sources:
// sql/001_init.down.sql
// sql/001_init.up.sql
package main

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

var _sql001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\xf0\xf4\x73\x71\x8d\x50\xc8\x4c\xa9\x88\x4f\x2c\xc8\x8c\x2f\xc9\xcf\x4e\xcd\x2b\x8e\x2f\x2d\x4e\x2d\xca\x4b\xcc\x4d\x8d\x2f\x28\x4a\x4d\xcb\xac\xb0\xe6\xe2\x02\xab\x0d\x71\x74\xf2\x71\x55\x40\xa8\xb3\x06\x04\x00\x00\xff\xff\x28\xfb\x72\x2b\x42\x00\x00\x00")

func sql001_initDownSqlBytes() ([]byte, error) {
	return bindataRead(
		_sql001_initDownSql,
		"sql/001_init.down.sql",
	)
}

func sql001_initDownSql() (*asset, error) {
	bytes, err := sql001_initDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "sql/001_init.down.sql", size: 66, mode: os.FileMode(420), modTime: time.Unix(1669731333, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _sql001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x90\x41\x4b\xc3\x40\x10\x85\xef\xf9\x15\xef\xd8\x62\xfb\x0b\x7a\x8a\x66\xc0\xc5\x74\xa3\x61\x42\xdb\xd3\xb2\x36\x23\xae\xa5\x49\xcc\x66\x25\xfe\x7b\x21\xdb\x50\x2d\x1e\x87\xf7\x31\xef\xe3\x3d\x94\x94\x32\x81\xd3\xfb\x9c\x60\x3b\x67\x86\xf6\x24\x8d\xc7\x22\x01\x00\x57\x83\x69\xcf\x78\x2e\xd5\x36\x2d\x0f\x78\xa2\x03\x74\xc1\xd0\x55\x9e\xaf\x26\x22\x78\xe9\x1b\x7b\x96\xc8\xfd\xcd\xba\x5e\xde\xdc\xf8\x5f\x72\xec\xc5\x0e\x52\x1b\x3b\x20\x4b\x99\x58\x6d\xe9\x86\x90\xb1\x73\xbd\xf8\xdf\x44\x0c\xfc\xb1\xed\x62\x5b\xbc\x27\xdf\xdb\x8e\x25\x76\x8a\x1f\x8b\x8a\x51\x16\x3b\x95\x6d\xb0\x5e\x5f\x4d\xef\x66\xb1\x73\xf0\x03\xde\xed\x97\xc0\x22\x34\xee\x33\x08\x4e\xf2\x9d\x24\x97\x4d\x2a\xad\x5e\x2a\x82\xd2\x19\xed\xe1\xea\xd1\x5c\xe7\x31\xf3\x33\x13\x5f\x4d\x26\x85\xc6\x47\xfb\xea\xb1\x98\xc3\xd5\xa5\x68\xb9\xf9\x09\x00\x00\xff\xff\xaf\x08\x92\x35\x66\x01\x00\x00")

func sql001_initUpSqlBytes() ([]byte, error) {
	return bindataRead(
		_sql001_initUpSql,
		"sql/001_init.up.sql",
	)
}

func sql001_initUpSql() (*asset, error) {
	bytes, err := sql001_initUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "sql/001_init.up.sql", size: 358, mode: os.FileMode(420), modTime: time.Unix(1669715192, 0)}
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
	"sql/001_init.down.sql": sql001_initDownSql,
	"sql/001_init.up.sql":   sql001_initUpSql,
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
	"sql": &bintree{nil, map[string]*bintree{
		"001_init.down.sql": &bintree{sql001_initDownSql, map[string]*bintree{}},
		"001_init.up.sql":   &bintree{sql001_initUpSql, map[string]*bintree{}},
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

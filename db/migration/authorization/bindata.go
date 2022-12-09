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

var _sql001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xd2\xd5\x55\x70\x09\xf2\x0f\x50\xf0\xf4\x73\x71\x8d\x50\xc8\x4c\xa9\x88\x4f\x2c\xc8\x8c\x2f\xc9\xcf\x4e\xcd\x8b\x2f\x2d\x4e\x2d\xca\x4b\xcc\x4d\x8d\x2f\x28\x4a\x4d\xcb\xac\xb0\xe6\xe2\x02\x2b\x0d\x71\x74\xf2\x71\x55\x80\x2b\xb3\x06\x04\x00\x00\xff\xff\x4e\xb1\xc5\x90\x43\x00\x00\x00")

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

	info := bindataFileInfo{name: "sql/001_init.down.sql", size: 67, mode: os.FileMode(420), modTime: time.Unix(1670500937, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _sql001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x90\x51\x4f\x83\x30\x1c\xc4\xdf\xf9\x14\xf7\xb8\xc5\xf1\x05\xdc\x13\x4a\x13\x1b\x19\x55\x52\xb2\xed\x89\x54\xf8\x2f\xab\xb0\x82\xb4\x18\xf6\xed\x0d\x76\xa8\x31\xf4\xad\xb9\xff\xe5\x7e\x77\x8f\x19\x8b\x24\x83\x8c\x1e\x12\x06\xd5\xe9\xc2\xb5\x35\x19\xac\x02\x00\x18\x2c\xf5\x46\x5d\x08\x92\x1d\x24\x52\x21\x91\xe6\x49\xb2\xf9\xd6\xba\x9e\x4e\x7a\x5c\x52\xca\x9e\x94\xa3\xaa\x50\x0e\x71\x24\x99\xe4\x3b\xf6\xef\x82\xc6\x4e\xf7\x64\xff\x5e\x78\xc1\x96\x6d\xe7\xd3\xfc\xdf\xc3\x2c\x64\xbc\x64\x7c\x17\x65\x47\x3c\xb3\x23\x56\x33\xe6\xe6\x06\xb5\x0e\xd6\xd8\x73\xf9\x24\x72\x89\x4c\xec\x79\xbc\x45\x18\xfe\x96\xb9\x9b\xd9\x2f\x83\x75\x38\xab\x4f\x82\xc2\x60\xf4\xc7\x40\xa8\xe9\x1a\x04\x61\x08\x16\x4b\x11\x8b\x7b\x94\x67\x2a\x6b\xe8\x13\xae\xed\x00\xeb\x74\xd3\xc0\x10\x55\x70\x2d\x2a\x2a\x1b\xd5\x13\x94\x81\x36\x15\x8d\x98\x7c\xb7\x39\xf3\x94\xbf\xe6\x0c\x3c\x8d\xd9\x01\xba\x1a\x8b\x9f\x65\x8b\x19\xa3\xf0\x10\x93\x69\x7a\x22\xc5\x7b\xfb\x66\x17\xca\x6c\xbf\x02\x00\x00\xff\xff\xf4\xd3\x1e\x7c\xa3\x01\x00\x00")

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

	info := bindataFileInfo{name: "sql/001_init.up.sql", size: 419, mode: os.FileMode(420), modTime: time.Unix(1670500925, 0)}
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

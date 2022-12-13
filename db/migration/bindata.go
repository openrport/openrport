// Code generated for package api_token by go-bindata DO NOT EDIT. (@generated)
// sources:
// sql/001_init.down.sql
// sql/001_init.up.sql
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

	info := bindataFileInfo{name: "sql/001_init.down.sql", size: 67, mode: os.FileMode(420), modTime: time.Unix(1670843559, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _sql001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x90\xc1\x72\xb2\x30\x14\x85\xf7\x3c\xc5\x59\xea\xfc\xf2\x02\xbf\x2b\x2a\xb7\x53\xa6\x08\x96\x5e\x46\x5d\x31\x29\x5c\xc7\x54\x0d\x94\x84\x0e\xbe\x7d\x07\x51\xdb\x85\xd9\x65\x4e\x4e\xee\xf7\xdd\x45\x46\x01\x13\x38\x78\x8a\x09\xaa\xd1\x85\xab\x0f\x62\x30\xf1\x00\xa0\xb3\xd2\x1a\x75\x12\x30\x6d\x18\x49\xca\x48\xf2\x38\x9e\x5d\xb2\xa6\x95\x9d\xee\x1f\x25\x65\x2b\xca\x49\x55\x28\x87\x30\x60\xe2\x68\x49\xf7\x17\x08\xe9\x39\xc8\x63\xc6\x22\xcf\x32\x4a\xb8\x18\xd2\x77\x0e\x96\xab\x19\x2e\x65\xe9\x1b\xdd\x8a\xfd\x5b\x1e\x7f\xb5\x65\xdd\x8c\x20\xe3\x7d\xe4\x7c\x30\x7e\x95\x45\xcb\x20\xdb\xe2\x95\xb6\x98\xdc\x0c\x66\x57\xde\xa9\x37\xc5\x3a\xe2\x97\x34\x67\x64\xe9\x3a\x0a\xe7\xf0\xfd\x5f\xcf\x7f\x37\xad\x53\x67\x1d\xf6\xea\x5b\xa0\xd0\x19\xfd\xd5\x09\x0e\x72\xf6\x3c\xdf\x07\x85\x9c\x86\xe9\x7f\x94\x7b\x29\x0f\xd0\x3b\x9c\xeb\x0e\xd6\xe9\xe3\x11\x46\xa4\x82\xab\x51\x49\x79\x54\xad\x40\x19\x68\x53\x49\x8f\xa1\x77\xdd\x74\x9e\x44\x6f\x39\x21\x4a\x42\xda\x40\x57\x7d\x71\x5f\x7a\x71\xc3\x28\x46\x88\xa1\x34\x9c\x34\xc1\x67\xfd\x61\x1f\xc8\xcc\x7f\x02\x00\x00\xff\xff\x32\xf0\xcb\x39\xbe\x01\x00\x00")

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

	info := bindataFileInfo{name: "sql/001_init.up.sql", size: 446, mode: os.FileMode(420), modTime: time.Unix(1670930970, 0)}
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

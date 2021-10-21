// Code generated for package library by go-bindata DO NOT EDIT. (@generated)
// sources:
// 001_init.down.sql
// 001_init.up.sql
// 002_commands.down.sql
// 002_commands.up.sql
package library

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

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x4e\x2e\xca\x2c\x28\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\x24\x6d\x54\xc3\x14\x00\x00\x00")

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

	info := bindataFileInfo{name: "001_init.down.sql", size: 20, mode: os.FileMode(436), modTime: time.Unix(1625562284, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x90\xcb\x6a\xc3\x30\x10\x45\xf7\xfa\x8a\x4b\x56\x31\xd4\xd0\xae\xb3\x32\x89\x28\xa6\xa9\xd2\xba\x32\x24\x2b\x47\xb1\x26\x60\x70\xe5\x54\x1a\x51\xfa\xf7\x25\x96\x29\x69\xe9\x4b\x3b\xcd\x9d\x73\xa4\x99\x3c\x47\xfe\xcb\x11\x79\x0e\x6d\x0e\x3d\x21\xb0\x8f\x2d\x47\x4f\x38\x0e\x1e\xa1\xf5\xdd\x89\x83\xf8\x0b\x6f\x3d\x19\x26\x70\x52\x4c\xd0\x5c\x00\x40\x67\xa1\xe5\x56\xe3\xa1\x2a\xef\x8b\x6a\x87\x3b\xb9\x83\xda\x68\xa8\x7a\xbd\xbe\x1a\x3b\x9c\x79\xa6\xd4\xe3\x06\x86\x8b\x7d\x9f\xea\x49\x6a\x1b\xc3\x58\x15\x5a\xfe\x90\x1e\xde\xbe\x63\x3b\xc7\xe4\x4f\x9e\x98\xfc\x18\x4f\xd5\xd0\x84\x68\x07\x94\x4a\xcb\x5b\x59\xcd\x6f\x32\x58\x3a\x9a\xd8\x33\xae\xbf\xea\x5f\xed\x05\x98\x46\xfa\xfc\x90\xc8\x16\x42\x2c\x2b\x79\xfe\x5a\xa9\x56\x72\x8b\xd9\x79\x92\xd9\x08\x6c\x14\xf6\xd3\x1e\xf6\x48\x8b\x48\x29\x8a\xa7\xe5\x78\xbd\xa0\x6b\x55\x3e\xd6\x1f\x92\xe8\xba\x97\x48\xcd\x7f\x5d\xd9\x42\xbc\x07\x00\x00\xff\xff\xe9\x96\x0a\x2f\xdd\x01\x00\x00")

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

	info := bindataFileInfo{name: "001_init.up.sql", size: 477, mode: os.FileMode(436), modTime: time.Unix(1625562284, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_commandsDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\xce\xcf\xcd\x4d\xcc\x4b\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\xb6\x29\x99\x09\x15\x00\x00\x00")

func _002_commandsDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_commandsDownSql,
		"002_commands.down.sql",
	)
}

func _002_commandsDownSql() (*asset, error) {
	bytes, err := _002_commandsDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_commands.down.sql", size: 21, mode: os.FileMode(436), modTime: time.Unix(1632139533, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_commandsUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x8f\xc1\x4e\xc4\x20\x14\x45\xf7\x7c\xc5\x4d\x57\x4e\x22\x5f\x30\xab\x66\x86\x45\xe3\xc8\x68\xa5\xc9\xcc\xaa\x43\x01\x93\x26\x85\x2a\x85\x85\x7f\x6f\x0a\xa9\xd1\xa4\x56\xd9\x71\xef\x3b\x27\xef\x51\x0a\xba\xf1\x08\xa5\x10\xb2\x1b\x0c\xa6\xe0\xa3\x0a\xd1\x1b\xbc\x8e\x1e\x6a\xb4\x56\x3a\x3d\x91\xbf\x78\xe5\x8d\x0c\x06\x21\x39\xbe\xa8\x3b\x02\x00\xbd\x86\x60\x17\x81\xa7\xba\x7a\x2c\xeb\x2b\x1e\xd8\x15\xfc\x2c\xc0\x9b\xd3\xe9\x3e\x4d\x38\x69\x4d\x9e\x71\x63\x80\x8b\xc3\x90\xf3\x6c\xd5\xad\x0c\x38\x96\x82\xfd\xd2\x76\x1f\x6b\x6c\x7c\xd3\x1b\xec\xd2\xae\xb3\xca\xea\x9f\x31\xd9\xed\x09\x39\xd4\x6c\x16\x55\xfc\xc8\x2e\x28\x96\x23\xdb\x76\xde\xbe\x48\xdc\x99\xe3\xb6\xe4\x37\xe4\xeb\x8b\x54\xa3\x7c\x39\xa4\xef\x37\x51\xc3\xab\xe7\x66\xc5\x17\x5d\xff\x1e\xcd\xff\xb5\xbb\x3d\xf9\x0c\x00\x00\xff\xff\x78\x59\x4f\x68\xdf\x01\x00\x00")

func _002_commandsUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_commandsUpSql,
		"002_commands.up.sql",
	)
}

func _002_commandsUpSql() (*asset, error) {
	bytes, err := _002_commandsUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_commands.up.sql", size: 479, mode: os.FileMode(436), modTime: time.Unix(1632141081, 0)}
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
	"001_init.down.sql":     _001_initDownSql,
	"001_init.up.sql":       _001_initUpSql,
	"002_commands.down.sql": _002_commandsDownSql,
	"002_commands.up.sql":   _002_commandsUpSql,
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
	"001_init.down.sql":     &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":       &bintree{_001_initUpSql, map[string]*bintree{}},
	"002_commands.down.sql": &bintree{_002_commandsDownSql, map[string]*bintree{}},
	"002_commands.up.sql":   &bintree{_002_commandsUpSql, map[string]*bintree{}},
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

// Code generated for package api_sessions by go-bindata DO NOT EDIT. (@generated)
// sources:
// 001_init.down.sql
// 001_init.up.sql
// 002_extended_session_tracking.down.sql
// 002_extended_session_tracking.up.sql
package api_sessions

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

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\xf0\xf4\x73\x71\x8d\x50\xc8\x4c\xa9\x88\x4f\xad\x28\xc8\x2c\x4a\x2d\x8e\x4f\x2c\x89\x2f\xc9\xcc\x4d\xb5\xe6\xe2\x02\x2b\x08\x71\x74\xf2\x71\x55\x48\x2c\xc8\x8c\x2f\x4e\x2d\x2e\xce\xcc\xcf\x2b\xb6\xe6\x02\x04\x00\x00\xff\xff\x83\xa6\x75\xc2\x3a\x00\x00\x00")

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

	info := bindataFileInfo{name: "001_init.down.sql", size: 58, mode: os.FileMode(420), modTime: time.Unix(1656320425, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x5c\xce\xbf\xaa\x83\x30\x14\x06\xf0\xf9\xe6\x29\xbe\x51\xe1\xbe\x81\x53\x6a\x0e\x34\x54\x93\x92\x1e\x51\xa7\x20\x34\x43\x28\x55\x69\x32\xf8\xf8\x85\xd2\x56\xe8\xfe\xfb\xfe\xd4\x8e\x24\x13\x58\x1e\x1a\xc2\xb4\x46\x9f\x42\x4a\x71\x99\x13\x0a\x01\x00\x79\xb9\x85\x19\x4c\x03\xe3\xec\x74\x2b\xdd\x88\x13\x8d\x30\x96\x61\xba\xa6\xf9\x7f\xa1\xb0\xad\xf1\x11\x92\x9f\x32\x94\x64\x62\xdd\xd2\x57\x88\x12\xbd\xe6\xa3\xed\x18\xce\xf6\x5a\x55\x42\xbc\x37\xb5\x51\x34\x20\x5e\x37\xbf\xe7\x7d\x8e\xf7\x20\xfe\xac\xf9\xf9\xf2\xa9\x2d\x76\x5a\x42\xd1\xa5\x2e\x2b\xf1\x0c\x00\x00\xff\xff\xfa\xf2\xd4\x9c\xc3\x00\x00\x00")

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

	info := bindataFileInfo{name: "001_init.up.sql", size: 195, mode: os.FileMode(420), modTime: time.Unix(1667474261, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_extended_session_trackingDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x5c\xce\x41\x8b\x83\x30\x10\x05\xe0\xf3\xe6\x57\xbc\xa3\xc2\xfe\x03\x4f\x59\x33\xcb\x86\xd5\x44\xe2\x88\x7a\x0a\x42\x73\x08\xa5\x2a\x8d\x07\x7f\x7e\xa1\xb4\xb5\xf4\x3c\xdf\xbc\xf7\x94\xb3\x0d\x58\xfe\x54\x04\xfd\x0b\x1a\x74\xcb\x2d\xa6\x35\xfa\x14\x52\x8a\xcb\x9c\x0a\x21\x4a\x47\x92\xe9\xa1\xde\x6f\xc8\x04\x00\x6c\xcb\x39\xcc\x60\x1a\x18\x8d\xd3\xb5\x74\x23\xfe\x69\x84\xb1\x0c\xd3\x55\xd5\xf7\x1d\x85\x7d\x8d\xd7\x90\xfc\xb4\x41\x49\x26\xd6\x35\xbd\x84\xc8\xd1\x6b\xfe\xb3\x1d\xc3\xd9\x5e\xab\xa3\x53\x1b\x45\x03\xe2\x69\xf7\xc7\xbf\xdf\xe2\x25\x88\x2f\x6b\x3e\xb6\x3c\x63\xb3\x83\xe6\x50\xd4\x96\x79\x21\x6e\x01\x00\x00\xff\xff\x28\x39\xa3\x87\xe7\x00\x00\x00")

func _002_extended_session_trackingDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_extended_session_trackingDownSql,
		"002_extended_session_tracking.down.sql",
	)
}

func _002_extended_session_trackingDownSql() (*asset, error) {
	bytes, err := _002_extended_session_trackingDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_extended_session_tracking.down.sql", size: 231, mode: os.FileMode(420), modTime: time.Unix(1667530753, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_extended_session_trackingUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x50\x4d\x6e\xf2\x30\x14\x5c\x7f\x3e\xc5\x88\x15\x48\x1f\x27\xc8\x2a\x25\xaf\x95\x55\x70\xa8\x71\x24\x58\x59\x2e\x7e\xaa\xac\x42\x40\xb1\x2b\x71\xfc\x4a\x81\x34\x29\x8d\xc4\xd2\x6f\x7e\x3c\x33\xf3\x39\xdc\x21\x71\x83\xe4\xde\x0f\x8c\x89\x3b\x07\x1b\x39\xc6\x70\xaa\xe3\x04\xce\x7b\x7c\x45\x6e\x6a\x77\x64\x18\xda\x1a\xa8\xd2\x40\x55\xcb\x65\x26\x1e\x2b\x6f\x2f\x1b\x3c\xa4\x32\xf4\x42\x1a\x79\x65\x4a\xa9\x16\x9a\x56\xa4\x4c\x26\x44\xa1\xcb\x35\x4c\xfe\xb4\x24\xc8\x67\xd0\x56\x6e\xcc\x06\x43\xa7\x4c\x88\x85\xa6\xdc\xd0\x8d\x35\xc4\x30\x15\x00\xc6\xbe\x59\x6b\xb9\xca\xf5\x0e\xaf\xb4\xfb\x49\xfc\xbf\x25\xa7\xd3\x27\xd7\xbf\xab\x5c\x01\xbe\x9c\x43\xc3\xd1\xba\x84\x22\x37\x64\xe4\x8a\xee\x18\xe3\x43\x5c\xb1\x83\x8b\xc9\xba\xfd\x9e\xe3\x23\x07\xeb\x3e\xb8\x4e\xad\xc7\xf5\x18\xce\xd6\x79\xdf\x70\x8c\xed\x51\xcc\xfa\xca\x95\x92\x6f\x15\x41\xaa\x82\xb6\x08\xfe\x62\xdb\xf4\x02\x28\xd5\xdd\x10\x2d\x30\x50\xf6\x92\xbe\x97\x4d\xe1\xc8\xe2\xdf\x1f\x6d\x17\x76\xda\x53\x67\x28\x68\xb3\x18\xf5\xeb\x56\x18\x4b\xd1\x61\xb3\x4c\x7c\x07\x00\x00\xff\xff\xe9\x88\x4e\x08\x58\x02\x00\x00")

func _002_extended_session_trackingUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_extended_session_trackingUpSql,
		"002_extended_session_tracking.up.sql",
	)
}

func _002_extended_session_trackingUpSql() (*asset, error) {
	bytes, err := _002_extended_session_trackingUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_extended_session_tracking.up.sql", size: 600, mode: os.FileMode(420), modTime: time.Unix(1667815364, 0)}
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
	"001_init.down.sql":                      _001_initDownSql,
	"001_init.up.sql":                        _001_initUpSql,
	"002_extended_session_tracking.down.sql": _002_extended_session_trackingDownSql,
	"002_extended_session_tracking.up.sql":   _002_extended_session_trackingUpSql,
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
	"001_init.down.sql":                      &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":                        &bintree{_001_initUpSql, map[string]*bintree{}},
	"002_extended_session_tracking.down.sql": &bintree{_002_extended_session_trackingDownSql, map[string]*bintree{}},
	"002_extended_session_tracking.up.sql":   &bintree{_002_extended_session_trackingUpSql, map[string]*bintree{}},
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

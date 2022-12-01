// Code generated for package clients by go-bindata DO NOT EDIT. (@generated)
// sources:
// 001_init.down.sql
// 001_init.up.sql
// 002_stored_tunnels.down.sql
// 002_stored_tunnels.up.sql
// 003_add_tunnel_fields.down.sql
// 003_add_tunnel_fields.up.sql
package clients

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

var __001_initDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\xf0\xf4\x73\x71\x8d\x50\xc8\x4c\xa9\x88\x4f\xc9\x2c\x4e\xce\xcf\xcb\x4b\x4d\x2e\x49\x4d\x89\x2f\xc9\xcc\x4d\x8d\x4f\xce\xc9\x4c\xcd\x2b\xb1\xe6\xe2\xc2\xa7\x12\x55\x51\x88\xa3\x93\x8f\xab\x02\x44\xac\xd8\x9a\x0b\x10\x00\x00\xff\xff\x49\xd7\x0b\xc9\x63\x00\x00\x00")

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

	info := bindataFileInfo{name: "001_init.down.sql", size: 99, mode: os.FileMode(420), modTime: time.Unix(1656320425, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __001_initUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x8f\x41\x6a\x85\x30\x14\x45\xc7\x66\x15\x77\x58\xc1\x1d\x74\x94\xea\x83\x86\x6a\x52\xd2\x27\xea\x28\x88\x09\x34\x60\xed\xc0\x14\xba\xfc\xd2\x5a\xb1\xf6\xf3\xf9\xe3\xdc\x9c\x73\x5e\x69\x49\x32\x81\xe5\x43\x4d\x98\xe6\x18\x96\xb4\xe2\x4e\x00\x40\xf4\x60\xea\x19\xcf\x56\x35\xd2\x0e\x78\xa2\x01\xda\x30\x74\x5b\xd7\x85\xc8\xb6\xb1\x1b\x3f\xd2\xab\xdb\xa7\xc7\xf3\x37\xc0\xc7\x75\x7a\x5f\x96\x30\xa5\xe0\xdd\x98\x50\x49\x26\x56\x0d\x15\x22\xf3\x21\x8d\x71\x5e\xcf\xbf\x44\x8e\x4e\xf1\xa3\x69\x19\xd6\x74\xaa\xba\x17\xe2\x37\x4f\xe9\x8a\x7a\x44\xff\xe9\x4e\xcc\x2d\xe1\xc7\x65\xf4\x51\x7f\xe1\xa5\x97\xb2\xc0\xb9\x37\xbf\x09\x4f\xf1\x2d\xec\x86\xec\x2f\x7e\x3f\xe3\xbf\x27\xbf\x26\xfa\x0a\x00\x00\xff\xff\x7e\x6a\x27\xc9\x64\x01\x00\x00")

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

	info := bindataFileInfo{name: "001_init.up.sql", size: 356, mode: os.FileMode(420), modTime: time.Unix(1656320425, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_stored_tunnelsDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x2e\xc9\x2f\x4a\x4d\x89\x2f\x29\xcd\xcb\x4b\xcd\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\x25\xf8\x88\xbe\x1b\x00\x00\x00")

func _002_stored_tunnelsDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_stored_tunnelsDownSql,
		"002_stored_tunnels.down.sql",
	)
}

func _002_stored_tunnelsDownSql() (*asset, error) {
	bytes, err := _002_stored_tunnelsDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_stored_tunnels.down.sql", size: 27, mode: os.FileMode(420), modTime: time.Unix(1656320425, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __002_stored_tunnelsUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x64\x8c\x4b\x6a\xc3\x30\x14\x45\xe7\x5e\xc5\x1d\x26\xd0\x1d\x74\xa4\xc8\xaf\x10\x2a\x2b\x45\x79\x81\x66\x24\x84\xf5\xa0\x02\x47\x0e\x92\xba\xff\x82\x3f\x83\x92\xe1\x3d\xf7\x70\xb4\x23\xc5\x04\x56\x27\x43\xa8\x6d\x2e\x12\x7d\xfb\xcd\x59\xa6\x8a\x43\x07\x00\x29\x82\xe9\x9b\xf1\xe5\xce\x83\x72\x77\x7c\xd2\x1d\xf6\xc2\xb0\x37\x63\xde\x16\x63\x9c\x92\xe4\xe6\x77\x71\x3f\xe1\xe8\x83\x1c\x59\x4d\xd7\x4d\xa9\x87\x14\x8f\xb8\x58\xf4\x64\x88\x09\x5a\x5d\xb5\xea\x69\xab\x14\x09\x4d\xa2\x0f\x0d\xbd\x62\xe2\xf3\xb0\x1d\x39\x3c\x64\x29\xaf\xb3\x8e\x3f\xf2\x0f\x14\x79\xcc\x4d\x7c\x7a\xbe\xb2\xe7\x5c\x1a\xec\x6d\x38\x91\x5b\x79\x18\xa7\xc5\xea\x8e\xef\xdd\x5f\x00\x00\x00\xff\xff\x04\x1b\x73\xc3\xfb\x00\x00\x00")

func _002_stored_tunnelsUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__002_stored_tunnelsUpSql,
		"002_stored_tunnels.up.sql",
	)
}

func _002_stored_tunnelsUpSql() (*asset, error) {
	bytes, err := _002_stored_tunnelsUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "002_stored_tunnels.up.sql", size: 251, mode: os.FileMode(420), modTime: time.Unix(1656320425, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_add_tunnel_fieldsDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00")

func _003_add_tunnel_fieldsDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_add_tunnel_fieldsDownSql,
		"003_add_tunnel_fields.down.sql",
	)
}

func _003_add_tunnel_fieldsDownSql() (*asset, error) {
	bytes, err := _003_add_tunnel_fieldsDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_add_tunnel_fields.down.sql", size: 0, mode: os.FileMode(420), modTime: time.Unix(1656320425, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __003_add_tunnel_fieldsUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x28\x2e\xc9\x2f\x4a\x4d\x89\x2f\x29\xcd\xcb\x4b\xcd\x29\x56\x70\x74\x71\x51\x28\x28\x4d\xca\xc9\x4c\x8e\x2f\xc8\x2f\x2a\x51\xf0\x0b\xf5\x75\x72\x0d\xb2\xe6\x22\xa0\x25\xad\xb4\xa8\x24\x23\xb5\x28\x3e\xbf\xa0\x24\x33\x3f\xaf\x58\x21\xc4\x35\x22\xc4\x9a\x0b\x10\x00\x00\xff\xff\x0a\x14\xe8\x34\x68\x00\x00\x00")

func _003_add_tunnel_fieldsUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__003_add_tunnel_fieldsUpSql,
		"003_add_tunnel_fields.up.sql",
	)
}

func _003_add_tunnel_fieldsUpSql() (*asset, error) {
	bytes, err := _003_add_tunnel_fieldsUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "003_add_tunnel_fields.up.sql", size: 104, mode: os.FileMode(420), modTime: time.Unix(1656320425, 0)}
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
	"001_init.down.sql":              _001_initDownSql,
	"001_init.up.sql":                _001_initUpSql,
	"002_stored_tunnels.down.sql":    _002_stored_tunnelsDownSql,
	"002_stored_tunnels.up.sql":      _002_stored_tunnelsUpSql,
	"003_add_tunnel_fields.down.sql": _003_add_tunnel_fieldsDownSql,
	"003_add_tunnel_fields.up.sql":   _003_add_tunnel_fieldsUpSql,
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
	"001_init.down.sql":              &bintree{_001_initDownSql, map[string]*bintree{}},
	"001_init.up.sql":                &bintree{_001_initUpSql, map[string]*bintree{}},
	"002_stored_tunnels.down.sql":    &bintree{_002_stored_tunnelsDownSql, map[string]*bintree{}},
	"002_stored_tunnels.up.sql":      &bintree{_002_stored_tunnelsUpSql, map[string]*bintree{}},
	"003_add_tunnel_fields.down.sql": &bintree{_003_add_tunnel_fieldsDownSql, map[string]*bintree{}},
	"003_add_tunnel_fields.up.sql":   &bintree{_003_add_tunnel_fieldsUpSql, map[string]*bintree{}},
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

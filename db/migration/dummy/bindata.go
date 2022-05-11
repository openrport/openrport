// Code generated for package clients by go-bindata DO NOT EDIT. (@generated)
package dummy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var __001_init_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x29\xcd\xcd\xad\xb4\xe6\x02\x04\x00\x00\xff\xff\x76\xf1\xe7\xa5\x12\x00\x00\x00")

func _001_init_down_sql() ([]byte, error) {
	return bindata_read(
		__001_init_down_sql,
		"001_init.down.sql",
	)
}

var __001_init_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x8d\xc1\xaa\xc2\x30\x10\x45\xf7\xfd\x8a\xbb\x7c\x0f\xfc\x03\x57\xd1\x0e\x18\x4c\x1b\x09\x53\x6a\x57\x21\x24\x01\x03\x25\x2e\x12\xff\x5f\xac\x2e\x0c\x78\xb6\x73\xce\xdc\xa3\x21\xc1\x04\x16\x07\x45\xf0\x6b\x8a\xb9\x96\xee\xaf\x03\x80\x14\xd0\xc0\x74\x65\x5c\x8c\x1c\x84\x59\x70\xa6\x05\xa3\x66\x8c\x93\x52\xbb\x4d\x7f\xc7\xd6\x3d\xea\xcd\xbe\xd2\x4d\xff\xa6\xd5\x43\x2a\xfe\x9e\x73\xf4\x35\x06\xeb\x2a\x7a\xc1\xc4\x72\xa0\xcf\x35\x56\x97\xd6\xd2\x6e\xff\x7a\xd6\xfd\x63\x96\x7c\xd2\x13\xc3\xe8\x59\xf6\xfb\x67\x00\x00\x00\xff\xff\xd8\x18\x87\x60\xd1\x00\x00\x00")

func _001_init_up_sql() ([]byte, error) {
	return bindata_read(
		__001_init_up_sql,
		"001_init.up.sql",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
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
var _bindata = map[string]func() ([]byte, error){
	"001_init.down.sql": _001_init_down_sql,
	"001_init.up.sql":   _001_init_up_sql,
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
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func     func() ([]byte, error)
	Children map[string]*_bintree_t
}

var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"001_init.down.sql": &_bintree_t{_001_init_down_sql, map[string]*_bintree_t{}},
	"001_init.up.sql":   &_bintree_t{_001_init_up_sql, map[string]*_bintree_t{}},
}}

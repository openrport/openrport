package fskv

import (
	"context"
	"os"
	"path"
)

type FSKV struct {
	basePath string
}

func NewFSKV(basePath string) (*FSKV, error) {
	err := os.MkdirAll(basePath, 0700)
	return &FSKV{basePath: basePath}, err
}

func (F FSKV) Put(ctx context.Context, key string, data []byte) error {
	p := path.Join(F.basePath, key)
	return os.WriteFile(p, data, 0600)
}

func (F FSKV) ReadAll(ctx context.Context, reader func(key string, data []byte) error) error {
	list, err := os.ReadDir(F.basePath)
	if err != nil {
		return err
	}

	for _, f := range list {
		if !f.IsDir() {
			data, err := os.ReadFile(path.Join(F.basePath, f.Name()))
			if err != nil {
				return err
			}
			err = reader(f.Name(), data)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (F FSKV) Delete(ctx context.Context, key string) error {
	return os.Remove(path.Join(F.basePath, key))
}

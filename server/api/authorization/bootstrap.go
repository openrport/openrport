package authorization

import (
	"context"
	"fmt"
	"github.com/realvnc-labs/rport/db/migration/api_token"
	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/share/simplestore"
	"github.com/realvnc-labs/rport/share/simplestore/kvs/fskv"
	"path"
)

const StorageKindFSKV = storageKind("fskv")
const StorageKindSQLite = storageKind("sqlite")

type storageKind string

type FSKVConfig struct {
}

type BoostrapConfig struct {
	StorageKind         storageKind
	FSKVConfig          *FSKVConfig
	SQLiteStorageConfig SQLiteStorageConfig
}

func BootstrapManager(ctx context.Context, config BoostrapConfig) (*Manager, error) {
	provider, err := BootstrapProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	return NewManager(provider), nil
}

type provider interface {
	DbProvider
	IterableStorage
}

func BootstrapProvider(ctx context.Context, config BoostrapConfig) (provider, error) {
	switch config.StorageKind {
	case StorageKindFSKV:
		return BoostrapWithFSKV(ctx, config.SQLiteStorageConfig)
	case StorageKindSQLite:
		return BoostrapWithSQLite(ctx, config.SQLiteStorageConfig)
	}
	return nil, fmt.Errorf("unknown storage kind: %v", config.StorageKind)
}

type StorageConfig interface {
	GetDataDir() string
}

type SQLiteStorageConfig interface {
	StorageConfig
	GetSQLiteDataSourceOptions() sqlite.DataSourceOptions
}

func BoostrapWithSQLite(ctx context.Context, config SQLiteStorageConfig) (*SqliteProvider, error) {

	apiTokenDb, err := sqlite.New(
		path.Join(config.GetDataDir(), "api_token.db"),
		api_token.AssetNames(),
		api_token.Asset,
		config.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed init api_token DB instance: %w", err)
	}

	return NewSqliteProvider(apiTokenDb), nil
}

func BoostrapWithFSKV(ctx context.Context, config StorageConfig) (*KVAPITokenProvider, error) {
	fileAPITokensStore, err := fskv.NewFSKV(path.Join(config.GetDataDir(), "api-tokens"))
	if err != nil {
		return nil, err
	}
	simpleAPITokensStore, err := simplestore.NewSimpleStore[APIToken](ctx, fileAPITokensStore)
	if err != nil {
		return nil, err
	}

	return NewKVAPITokenProvider(simpleAPITokensStore), nil
}

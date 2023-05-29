package clients

import (
	"context"
	"fmt"
	clientsmigration "github.com/realvnc-labs/rport/db/migration/clients"
	"github.com/realvnc-labs/rport/db/sqlite"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/simplestore"
	"github.com/realvnc-labs/rport/share/simplestore/kvs/fskv"
	"path"
	"time"
)

const StorageKindFSKV = storageKind("fskv")
const StorageKindSQLite = storageKind("sqlite")

type storageKind string

type FSKVConfig struct {
}

type BoostrapConfig struct {
	StorageKind             storageKind
	KeepDisconnectedClients *time.Duration
	FSKVConfig              StorageConfig
	SQLiteStorageConfig     SQLiteStorageConfig
}

func BootstrapManager(ctx context.Context, config BoostrapConfig, logger *logger.Logger) (*ClientRepository, error) {
	provider, err := BootstrapProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	return NewClientRepositoryWithDB(config.KeepDisconnectedClients, provider, logger), nil
}

func BootstrapProvider(ctx context.Context, config BoostrapConfig) (ClientStore, error) {
	switch config.StorageKind {
	case StorageKindFSKV:
		return BoostrapWithFSKV(ctx, config.SQLiteStorageConfig, config.KeepDisconnectedClients)
	case StorageKindSQLite:
		return BoostrapWithSQLite(ctx, config.SQLiteStorageConfig, config.KeepDisconnectedClients)
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

func BoostrapWithSQLite(ctx context.Context, config SQLiteStorageConfig, keepDisconnectedClients *time.Duration) (*SqliteProvider, error) {

	clientDB, err := sqlite.New(
		path.Join(config.GetDataDir(), "clients.db"),
		clientsmigration.AssetNames(),
		clientsmigration.Asset,
		config.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create clients DB instance: %v", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed init api_token DB instance: %w", err)
	}

	return newSqliteProvider(clientDB, keepDisconnectedClients), nil
}

func BoostrapWithFSKV(ctx context.Context, config StorageConfig, keepDisconnectedClients *time.Duration) (*SimpleClientStore, error) {
	fileAPITokensStore, err := fskv.NewFSKV(path.Join(config.GetDataDir(), "clients"))
	if err != nil {
		return nil, err
	}
	simpleAPITokensStore, err := simplestore.NewSimpleStore[Client](ctx, fileAPITokensStore)
	if err != nil {
		return nil, err
	}

	return NewSimpleClientStore(simpleAPITokensStore, keepDisconnectedClients), nil
}

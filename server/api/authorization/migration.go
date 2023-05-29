package authorization

import (
	"context"
	"github.com/realvnc-labs/rport/server/chconfig"
)

type IterableStorage interface {
	GetAll(ctx context.Context) ([]*APIToken, error)
	Save(ctx context.Context, item *APIToken) error
}

func Migrate(cfg chconfig.ServerConfig, from, to string) error {
	ctx := context.Background()
	fromStorage, err := BootstrapProvider(ctx, BoostrapConfig{
		StorageKind:         storageKind(from),
		SQLiteStorageConfig: &cfg,
	})
	if err != nil {
		return err
	}

	toStorage, err := BootstrapProvider(ctx, BoostrapConfig{
		StorageKind:         storageKind(to),
		SQLiteStorageConfig: &cfg,
	})
	if err != nil {
		return err
	}

	return Migrator(fromStorage, toStorage)

}

func Migrator(from, to IterableStorage) error {
	ctx := context.Background()
	all, err := from.GetAll(ctx)
	if err != nil {
		return err
	}
	for _, item := range all {
		err = to.Save(ctx, item)
		if err != nil {
			return err
		}
	}
	return nil
}

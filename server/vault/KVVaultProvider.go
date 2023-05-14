package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/realvnc-labs/rport/share/dynops"
	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/dynops/filterer"
	"github.com/realvnc-labs/rport/share/query"
)

type KVStoreDbStatus interface {
	Get(ctx context.Context, key string) (DbStatus, bool, error)
	Save(ctx context.Context, key string, status DbStatus) error
}

type KVStoreValues interface {
	Get(ctx context.Context, key string) (StoredValue, bool, error)
	GetAll(ctx context.Context) ([]StoredValue, error)
	Delete(ctx context.Context, sessionID string) error
	Save(ctx context.Context, sessionID string, session StoredValue) error
	Filter(ctx context.Context, sieve func(session StoredValue) bool) ([]StoredValue, error)
}

type KVVaultProvider struct {
	dbss                 KVStoreDbStatus
	valueStore           KVStoreValues
	sortTranslationTable map[string]dyncopy.Field
}

func NewKVVaultProvider(dbss KVStoreDbStatus, valueStore KVStoreValues) *KVVaultProvider {
	return &KVVaultProvider{dbss: dbss, valueStore: valueStore, sortTranslationTable: dyncopy.BuildTranslationTable(StoredValue{})}
}

func (K KVVaultProvider) GetStatus(ctx context.Context) (DbStatus, error) {
	get, _, err := K.dbss.Get(ctx, "status")
	return get, err
}

func (K KVVaultProvider) SetStatus(ctx context.Context, newStatus DbStatus) error {
	return K.dbss.Save(ctx, "status", newStatus)
}

func (K KVVaultProvider) GetByID(ctx context.Context, id int) (val StoredValue, found bool, err error) {
	return K.valueStore.Get(ctx, GenID(int64(id)))
}

func GenID(id int64) string {
	return fmt.Sprintf("%v", id)
}

func (K KVVaultProvider) List(ctx context.Context, options *query.ListOptions) ([]ValueKey, error) {
	if options == nil {
		return nil, fmt.Errorf("options for filtering and pagination is nil")
	}

	filter, err := filterer.CompileFromQueryListOptions[StoredValue](options.Filters)
	if err != nil {
		return nil, err
	}

	entities, err := K.valueStore.Filter(ctx, func(entity StoredValue) bool {
		return filter.Run(entity)
	})
	if err != nil {
		return nil, err
	}

	entities, err = dynops.FastSorter1(K.sortTranslationTable, entities, options.Sorts)
	if err != nil {
		return nil, err
	}

	entities = dynops.Paginator(entities, options.Pagination)

	vks := make([]ValueKey, len(entities))
	for i, entity := range entities {
		vks[i] = ValueKey{
			ID:        entity.ID,
			ClientID:  entity.ClientID,
			CreatedBy: entity.CreatedBy,
			CreatedAt: entity.CreatedAt,
			Key:       entity.Key,
		}
	}
	return vks, nil
}

func (K KVVaultProvider) FindByKeyAndClientID(ctx context.Context, key, clientID string) (val StoredValue, found bool, err error) {
	res, err := K.valueStore.Filter(ctx, func(session StoredValue) bool {
		return session.Key == key && session.ClientID == clientID
	})
	if err != nil {
		return StoredValue{}, false, err
	}
	if len(res) == 0 {
		return StoredValue{}, false, nil
	}

	return res[0], true, nil
}

func (K KVVaultProvider) Save(ctx context.Context, user string, idToUpdate int64, val *InputValue, nowDate time.Time) (int64, error) {
	original, found, err := K.valueStore.Get(ctx, GenID(idToUpdate))
	if err != nil {
		return 0, err
	}
	if !found {
		original.ID = int(idToUpdate)
		original.CreatedAt = nowDate
		original.CreatedBy = user
	}

	original.UpdatedBy = &user
	original.UpdatedAt = time.Now()
	original.InputValue = *val

	return idToUpdate, K.valueStore.Save(ctx, GenID(idToUpdate), original)
}

func (K KVVaultProvider) Delete(ctx context.Context, id int) error {
	return K.valueStore.Delete(ctx, GenID(int64(id)))
}

func (K KVVaultProvider) Close() error {
	// nop
	return nil
}

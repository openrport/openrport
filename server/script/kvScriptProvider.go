package script

import (
	"context"
	"fmt"
	"github.com/realvnc-labs/rport/share/dynops"
	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/dynops/filterer"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/random"
	"time"
)

type KVScriptStore interface {
	Get(ctx context.Context, key string) (Script, bool, error)
	GetAll(ctx context.Context) ([]Script, error)
	Delete(ctx context.Context, key string) error
	Save(ctx context.Context, key string, script Script) error
	Filter(ctx context.Context, sieve func(script Script) bool) ([]Script, error)
}

type KVScriptProvider struct {
	kv                   KVScriptStore
	sortTranslationTable map[string]dyncopy.Field
}

func NewKVScriptProvider(kv KVScriptStore) *KVScriptProvider {
	return &KVScriptProvider{kv: kv, sortTranslationTable: dyncopy.BuildTranslationTable(Script{})}
}

func (K KVScriptProvider) GetByID(ctx context.Context, id string, ro *query.RetrieveOptions) (val *Script, found bool, err error) {
	script, ok, err := K.kv.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	return &script, ok, err
}

func (K KVScriptProvider) List(ctx context.Context, options *query.ListOptions) ([]Script, error) {
	if options == nil {
		return nil, fmt.Errorf("options for filtering and pagination is nil")
	}

	filter, err := filterer.CompileFromQueryListOptions[Script](options.Filters)
	if err != nil {
		return nil, err
	}

	entities, err := K.kv.Filter(ctx, func(entity Script) bool {
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

	return entities, nil
}

func (K KVScriptProvider) Save(ctx context.Context, s *Script, nowDate time.Time) (string, error) {

	if s.ID == "" {
		newUUID, err := random.UUID4()
		if err != nil {
			return "", err
		}

		s.CreatedAt = &nowDate
		s.ID = newUUID
	}
	s.UpdatedAt = &nowDate

	return s.ID, K.kv.Save(ctx, s.ID, *s)
}

func (K KVScriptProvider) Delete(ctx context.Context, id string) error {
	return K.kv.Delete(ctx, id)
}

func (K KVScriptProvider) Close() error {
	// nop
	return nil
}

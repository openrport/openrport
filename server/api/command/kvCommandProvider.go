package command

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

type KVCommandStore interface {
	Get(ctx context.Context, key string) (Command, bool, error)
	GetAll(ctx context.Context) ([]Command, error)
	Delete(ctx context.Context, key string) error
	Save(ctx context.Context, key string, script Command) error
	Filter(ctx context.Context, sieve func(script Command) bool) ([]Command, error)
}

type KVCommandProvider struct {
	kv                   KVCommandStore
	sortTranslationTable map[string]dyncopy.Field
}

func NewKVCommandProvider(kv KVCommandStore) *KVCommandProvider {
	return &KVCommandProvider{kv: kv, sortTranslationTable: dyncopy.BuildTranslationTable(Command{})}
}

func (K KVCommandProvider) GetByID(ctx context.Context, id string, ro *query.RetrieveOptions) (val *Command, found bool, err error) {
	script, ok, err := K.kv.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	return &script, ok, err
}

func (K KVCommandProvider) List(ctx context.Context, options *query.ListOptions) ([]Command, error) {
	if options == nil {
		return nil, fmt.Errorf("options for filtering and pagination is nil")
	}

	filter, err := filterer.CompileFromQueryListOptions[Command](options.Filters)
	if err != nil {
		return nil, err
	}

	entities, err := K.kv.Filter(ctx, func(entity Command) bool {
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

func (K KVCommandProvider) Save(ctx context.Context, s *Command) (string, error) {

	now := time.Now()

	if s.ID == "" {
		newUUID, err := random.UUID4()
		if err != nil {
			return "", err
		}

		s.CreatedAt = &now
		s.ID = newUUID
	}
	s.UpdatedAt = &now

	return s.ID, K.kv.Save(ctx, s.ID, *s)
}

func (K KVCommandProvider) Delete(ctx context.Context, id string) error {
	return K.kv.Delete(ctx, id)
}

func (K KVCommandProvider) Close() error {
	// nop
	return nil
}

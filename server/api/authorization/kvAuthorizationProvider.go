package authorization

import (
	"context"
	"fmt"
	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/simpleops"
)

type KVAPITokenStore interface {
	Get(ctx context.Context, key string) (APIToken, bool, error)
	GetAll(ctx context.Context) ([]APIToken, error)
	Delete(ctx context.Context, key string) error
	Save(ctx context.Context, key string, script APIToken) error
	Filter(ctx context.Context, sieve func(script APIToken) bool) ([]APIToken, error)
}

type KVAPITokenProvider struct {
	kv                   KVAPITokenStore
	sortTranslationTable map[string]dyncopy.Field
}

func (K KVAPITokenProvider) Get(ctx context.Context, username, prefix string) (*APIToken, error) {
	get, ok, err := K.kv.Get(ctx, genTokenKey(username, prefix))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &get, err
}

func genTokenKey(username string, prefix string) string {
	return fmt.Sprintf("%v--%v", username, prefix)
}

func (K KVAPITokenProvider) GetByName(ctx context.Context, username, name string) (*APIToken, error) {

	filter, err := K.kv.Filter(ctx, func(token APIToken) bool {
		return token.Username == username && token.Name == name
	})
	if err != nil {
		return nil, err
	}

	if len(filter) == 0 {
		return nil, nil
	}

	return &filter[0], nil

}

func (K KVAPITokenProvider) GetAll(ctx context.Context, username string) ([]*APIToken, error) {

	filter, err := K.kv.Filter(ctx, func(token APIToken) bool {
		return token.Username == username
	})
	if err != nil {
		return nil, err
	}

	return simpleops.ToPointerSlice(filter), nil
}

func (K KVAPITokenProvider) Save(ctx context.Context, tokenLine *APIToken) error {
	return K.kv.Save(ctx, genTokenKey(tokenLine.Username, tokenLine.Prefix), *tokenLine)
}

func (K KVAPITokenProvider) Delete(ctx context.Context, username, prefix string) error {
	return K.kv.Delete(ctx, genTokenKey(username, prefix))
}

func NewKVAPITokenProvider(kv KVAPITokenStore) *KVAPITokenProvider {
	return &KVAPITokenProvider{kv: kv, sortTranslationTable: dyncopy.BuildTranslationTable(APIToken{})}
}

func (K KVAPITokenProvider) Close() error {
	// nop
	return nil
}

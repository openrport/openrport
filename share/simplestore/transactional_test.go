package simplestore_test

import (
	"context"
	"github.com/realvnc-labs/rport/share/simplestore"
	"github.com/realvnc-labs/rport/share/simplestore/kvs/inmemory"
	"github.com/stretchr/testify/suite"
	"sort"
	"testing"
)

type TypeA struct {
	V string
}

type TransactionalTestSuite struct {
	suite.Suite
	mockkvstore *inmemory.InMemory
	store       simplestore.TransactionalStore
}

func (suite *TransactionalTestSuite) SetupTest() {
	suite.mockkvstore = inmemory.NewInMemory()
	store, err := simplestore.NewTransactionalStore(context.Background(), "test-store", inmemory.NewInMemory())
	suite.NoError(err)
	suite.store = store
	//	var err error
	//	suite.store, err = simplestore.NewTransactional[DBStruct](context.Background(), suite.mockkvstore)
	//	suite.NoError(err)
}

func (s *TransactionalTestSuite) TestGetUnexistantKey() {
	ctx := context.Background()
	out, found, err := simplestore.WithType[TypeA](s.store).Get(ctx, "key")
	s.NoError(err)
	s.False(found)
	s.Equal(out, TypeA{})
}

type TypeB struct {
	V string
}

func (s *TransactionalTestSuite) TestAddTypeB() {
	ctx := context.Background()
	s.store.Transaction(ctx, func(ctx context.Context, tx simplestore.Transaction) error {
		return simplestore.TransactionWithType[TypeB](tx).Save("key", TypeB{V: "B"})
	})
	out, _, _ := simplestore.WithType[TypeB](s.store).Get(ctx, "key")
	s.Equal(TypeB{V: "B"}, out)
}

func (s *TransactionalTestSuite) TestAddManyTypes() {
	ctx := context.Background()
	s.NoError(s.store.Transaction(ctx, func(ctx context.Context, tx simplestore.Transaction) error {
		simplestore.TransactionWithType[TypeA](tx).Save("key", TypeA{V: "A"})
		simplestore.TransactionWithType[TypeA](tx).Save("key2", TypeA{V: "A2"})
		return simplestore.TransactionWithType[TypeB](tx).Save("key", TypeB{V: "B"})
	}))

	out, found, err := simplestore.WithType[TypeB](s.store).Get(ctx, "key")

	s.NoError(err)
	s.True(found)
	s.Equal(TypeB{V: "B"}, out)

	outA, found, err := simplestore.WithType[TypeA](s.store).Get(ctx, "key")

	s.NoError(err)
	s.True(found)
	s.Equal(TypeA{V: "A"}, outA)
}

func (s *TransactionalTestSuite) TestDelete() {
	s.TestAddManyTypes()

	ctx := context.Background()

	s.NoError(s.store.Transaction(ctx, func(ctx context.Context, tx simplestore.Transaction) error {
		return simplestore.TransactionWithType[TypeA](tx).Delete("key")
	}))

	out, found, err := simplestore.WithType[TypeB](s.store).Get(ctx, "key")

	s.NoError(err)
	s.True(found)
	s.Equal(TypeB{V: "B"}, out)

	outA, found, err := simplestore.WithType[TypeA](s.store).Get(ctx, "key")

	s.NoError(err)
	s.False(found)
	s.Equal(TypeA{}, outA)
}

func (s *TransactionalTestSuite) TestGetAll() {
	s.TestAddManyTypes()

	ctx := context.Background()

	out, err := simplestore.WithType[TypeA](s.store).GetAll(ctx)

	s.NoError(err)
	sort.Slice(out, func(i, j int) bool {
		return out[i].V < out[j].V
	})
	s.Equal([]TypeA{{V: "A"}, {V: "A2"}}, out)
}

func TestTransactionalTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionalTestSuite))
}

// potrzebuję mieć gwarancje że jak zmieniłem uprawnienia w grupie to wszystkim się zmieniły (dane grupy tylko w jednym miejscu) - mogłyby być wszędzie ale wydaje się to nie praktyczne jeśli nie możemy zagwarantować zakończenia updateu w przypadku przerobienia na inną bazę
// jak usunąłem grupę to wszyscy stracili uprawnienia (relacja tylko user -> grupa poprzez string slice więc rozwiązywanie uprawnień tylko poprzez zapytanie o każdą grupę  pokolei)
// jak dodałem kogoś do grupy to widnieje w grupie
//   -- mogę dodawać grupe do urzytkownika tylko i polegać na filtrowaniu (wolne kasowanie)
// dobrze byłoby sportować to na jakiś external KV

package simplestore_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/simplestore"
	"github.com/realvnc-labs/rport/share/simplestore/kvs/inmemory"
)

type SimpleStoreTestSuite struct {
	suite.Suite
	store       *simplestore.SimpleStore[TestStruct]
	mockkvstore *inmemory.InMemory
}

func (suite *SimpleStoreTestSuite) SetupTest() {
	suite.mockkvstore = inmemory.NewInMemory()
	var err error
	suite.store, err = simplestore.NewSimpleStore[TestStruct](context.Background(), suite.mockkvstore)
	suite.NoError(err)
}

type TestStruct struct {
	ID   string
	Key1 string
}

func (suite *SimpleStoreTestSuite) TestSimpleStore_GetAll() {
	all, err := suite.store.GetAll(context.Background())
	suite.NoError(err)
	suite.Equal([]TestStruct{}, all)
}

func (suite *SimpleStoreTestSuite) TestSimpleStore_Delete() {
	testStruct := TestStruct{ID: "some-id"}
	err := suite.store.Save(context.Background(), testStruct.ID, testStruct)
	suite.NoError(err)

	err = suite.store.Delete(context.Background(), testStruct.ID)
	suite.NoError(err)

	all, err := suite.store.GetAll(context.Background())
	suite.NoError(err)
	suite.Equal([]TestStruct{}, all)
}

func (suite *SimpleStoreTestSuite) TestSimpleStore_Save() {
	testStruct := TestStruct{ID: "some-id"}
	err := suite.store.Save(context.Background(), testStruct.ID, testStruct)
	suite.NoError(err)

	all, err := suite.store.GetAll(context.Background())
	suite.NoError(err)
	suite.Equal([]TestStruct{{ID: "some-id"}}, all)
}

func (suite *SimpleStoreTestSuite) TestSimpleStore_Persistence() {
	testStruct := TestStruct{ID: "some-id"}
	err := suite.store.Save(context.Background(), testStruct.ID, testStruct)
	suite.NoError(err)
	suite.store, _ = simplestore.NewSimpleStore[TestStruct](context.Background(), suite.mockkvstore)

	all, err := suite.store.GetAll(context.Background())
	suite.NoError(err)
	suite.Equal([]TestStruct{{ID: "some-id"}}, all)
}

func (suite *SimpleStoreTestSuite) TestSimpleStore_Filter_sort() {
	store, testStructs := suite.manyStore()
	options := query.ListOptions{
		Sorts: []query.SortOption{{
			Column: "ID",
			IsASC:  false,
		}},
		Filters:    nil,
		Fields:     nil,
		Pagination: nil,
	}

	all, err := store.Filter(context.Background(), options)
	suite.NoError(err)

	suite.Equal(testStructs, all)
}

func (suite *SimpleStoreTestSuite) manyStore() (*simplestore.SimpleStore[TestStruct], []TestStruct) {
	store, err := simplestore.NewSimpleStore[TestStruct](context.Background(), inmemory.NewInMemory())
	suite.NoError(err)

	testStructs := []TestStruct{
		{ID: "id-1", Key1: "key-1"},
		{ID: "id-2", Key1: "key-2"},
		{ID: "id-3", Key1: "key-3"},
	}

	for _, testStruct := range testStructs {
		err := store.Save(context.Background(), testStruct.ID, testStruct)
		suite.NoError(err)
	}
	return store, testStructs
}

func TestSimpleStoreTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleStoreTestSuite))
}

package simplestore_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

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
	ID string
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

func TestSimpleStoreTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleStoreTestSuite))
}

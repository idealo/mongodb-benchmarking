package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MockCollection struct {
	mock.Mock
}

func (m *MockCollection) InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document)
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockCollection) DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

func (m *MockCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCollection) Drop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func fetchDocumentIDsMock(_ CollectionAPI) ([]int64, error) {
	return []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil
}

func TestInsertOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	docCount := 10
	threads := 2
	testType := "insert"

	mockCollection.On("Drop", mock.Anything).Return(nil)
	mockCollection.On("InsertOne", mock.Anything, mock.Anything).Return(&mongo.InsertOneResult{}, nil)

	runTest(mockCollection, testType, threads, docCount, fetchDocumentIDsMock)

	mockCollection.AssertNumberOfCalls(t, "Drop", 1)
	mockCollection.AssertNumberOfCalls(t, "InsertOne", docCount)
}

func TestUpdateOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	docCount := 10
	threads := 2
	testType := "update"

	mockCollection.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mongo.UpdateResult{}, nil)

	runTest(mockCollection, testType, threads, docCount, fetchDocumentIDsMock)

	expectedCalls := docCount
	mockCollection.AssertNumberOfCalls(t, "UpdateOne", expectedCalls)
}

func TestUpsertOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	docCount := 10
	threads := 2
	testType := "upsert"

	mockCollection.On("Drop", mock.Anything).Return(nil)
	mockCollection.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mongo.UpdateResult{UpsertedCount: 1}, nil)

	runTest(mockCollection, testType, threads, docCount, fetchDocumentIDsMock)

	mockCollection.AssertNumberOfCalls(t, "Drop", 1)
	mockCollection.AssertNumberOfCalls(t, "UpdateOne", docCount)
}

func TestDeleteOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	docCount := 10
	threads := 2
	testType := "delete"

	mockCollection.On("DeleteOne", mock.Anything, mock.Anything).Return(&mongo.DeleteResult{DeletedCount: 1}, nil)

	runTest(mockCollection, testType, threads, docCount, fetchDocumentIDsMock)

	expectedCalls := docCount
	mockCollection.AssertNumberOfCalls(t, "DeleteOne", expectedCalls)
}

func TestCountDocuments(t *testing.T) {
	mockCollection := new(MockCollection)

	mockCollection.On("CountDocuments", mock.Anything, mock.Anything).Return(int64(10), nil)

	count, err := mockCollection.CountDocuments(context.Background(), bson.M{})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)
	mockCollection.AssertExpectations(t)
}

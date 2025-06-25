package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockCollection to mock MongoDB collection operations
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

func (m *MockCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, pipeline, opts)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

// fetchDocumentIDsMock returns a slice of mock ObjectIDs for testing
func fetchDocumentIDsMock(_ CollectionAPI, _ int64, _ string) ([]primitive.ObjectID, error) {
	return []primitive.ObjectID{
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
	}, nil
}

// TestInsertOperation tests the insert operation using DocCountTestingStrategy
func TestInsertOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	config := TestingConfig{
		Threads:  2,
		DocCount: 10,
		DropDb:   true,
	}
	strategy := DocCountTestingStrategy{}
	testType := "insert"

	mockCollection.On("Drop", mock.Anything).Return(nil)
	mockCollection.On("InsertOne", mock.Anything, mock.Anything).Return(&mongo.InsertOneResult{}, nil)

	strategy.runTest(mockCollection, testType, config, fetchDocumentIDsMock)

	mockCollection.AssertNumberOfCalls(t, "Drop", 1)
	mockCollection.AssertNumberOfCalls(t, "InsertOne", config.DocCount)
}

// TestUpdateOperation tests the update operation using DocCountTestingStrategy
func TestUpdateOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	config := TestingConfig{
		Threads:  2,
		DocCount: 10,
	}
	strategy := DocCountTestingStrategy{}
	testType := "update"

	mockCollection.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mongo.UpdateResult{}, nil)

	strategy.runTest(mockCollection, testType, config, fetchDocumentIDsMock)

	expectedCalls := config.DocCount
	mockCollection.AssertNumberOfCalls(t, "UpdateOne", expectedCalls)
}

// TestUpsertOperation tests the upsert operation using DocCountTestingStrategy
func TestUpsertOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	config := TestingConfig{
		Threads:  2,
		DocCount: 10,
		DropDb:   true,
	}
	strategy := DocCountTestingStrategy{}
	testType := "upsert"

	mockCollection.On("Drop", mock.Anything).Return(nil)
	mockCollection.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mongo.UpdateResult{UpsertedCount: 1}, nil)

	strategy.runTest(mockCollection, testType, config, fetchDocumentIDsMock)

	mockCollection.AssertNumberOfCalls(t, "Drop", 1)
	mockCollection.AssertNumberOfCalls(t, "UpdateOne", config.DocCount)
}

// TestDeleteOperation tests the delete operation using DocCountTestingStrategy
func TestDeleteOperation(t *testing.T) {
	mockCollection := new(MockCollection)
	config := TestingConfig{
		Threads:  2,
		DocCount: 10,
	}
	strategy := DocCountTestingStrategy{}
	testType := "delete"

	mockCollection.On("DeleteOne", mock.Anything, mock.Anything).Return(&mongo.DeleteResult{DeletedCount: 1}, nil)

	strategy.runTest(mockCollection, testType, config, fetchDocumentIDsMock)

	expectedCalls := config.DocCount
	mockCollection.AssertNumberOfCalls(t, "DeleteOne", expectedCalls)
}

// TestCountDocuments verifies the CountDocuments method in isolation
func TestCountDocuments(t *testing.T) {
	mockCollection := new(MockCollection)

	mockCollection.On("CountDocuments", mock.Anything, mock.Anything).Return(int64(10), nil)

	count, err := mockCollection.CountDocuments(context.Background(), bson.M{})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)
	mockCollection.AssertExpectations(t)
}

// helper to create a temporary PEM file
func writeTempPEM(t *testing.T, pem string) string {
	tmp, err := os.CreateTemp(t.TempDir(), "ca_*.pem")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if _, err := tmp.WriteString(pem); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	tmp.Close()
	return tmp.Name()
}

// generateValidSelfSignedCert creates a minimal, valid self‑signed CA certificate and returns it as PEM.
func generateValidSelfSignedCert(t *testing.T) string {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Test CA"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("failed to encode certificate: %v", err)
	}
	return buf.String()
}

func TestCreateTLSConfigFromFile_Success(t *testing.T) {
	certPEM := generateValidSelfSignedCert(t)
	path := writeTempPEM(t, certPEM)

	cfg, err := createTLSConfigFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil {
		t.Fatalf("expected non-nil TLS config with RootCAs")
	}
}

func TestCreateTLSConfigFromFile_Invalid(t *testing.T) {
	path := writeTempPEM(t, "not a cert")
	_, err := createTLSConfigFromFile(path)
	if err == nil {
		t.Fatalf("expected error for invalid cert data")
	}
}

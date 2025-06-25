package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	var (
		threads         int
		docCount        int
		uri             string
		certificatePath string
		testType        string
		duration        int
		runAll          bool
		largeDocs       bool
		dropDb          bool
	)

	flag.IntVar(&threads, "threads", 10, "Number of threads for inserting, updating, upserting, or deleting documents")
	flag.IntVar(&docCount, "docs", 1000, "Total number of documents to insert, update, upsert, or delete")
	flag.StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB URI")
	flag.StringVar(&certificatePath, "cert", "", "Path to TLS certificate")
	flag.StringVar(&testType, "type", "insert", "Test type: insert, update, upsert, or delete")
	flag.BoolVar(&runAll, "runAll", false, "Run all tests in order: insert, update, delete, upsert")
	flag.IntVar(&duration, "duration", 0, "Duration in seconds to run the test")
	flag.BoolVar(&largeDocs, "largeDocs", false, "Use large documents for testing")
	flag.BoolVar(&dropDb, "dropDb", true, "Drop the database before running the test")
	flag.Parse()

	var strategy TestingStrategy
	var config TestingConfig

	clientOptions := options.Client().ApplyURI(uri).SetMaxPoolSize(uint64(threads))

	if len(certificatePath) != 0 {
		tlsConfig, err := createTlsConfigFromFile(certificatePath)
		if err != nil {
			log.Fatalf("Failed to create tls config from %s: %v", certificatePath, err)
		}

		clientOptions = clientOptions.SetTLSConfig(tlsConfig)
	}

	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func(client *mongo.Client, ctx context.Context) {
		err := client.Disconnect(ctx)
		if err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}(client, context.Background())

	collection := client.Database("benchmarking").Collection("testdata")
	mongoCollection := &MongoDBCollection{Collection: collection}

	config = TestingConfig{
		Threads:   threads,
		Duration:  duration,
		DocCount:  docCount,
		LargeDocs: largeDocs,
		DropDb:    dropDb,
	}

	if duration > 0 {
		strategy = DurationTestingStrategy{}
	} else {
		strategy = DocCountTestingStrategy{}
	}
	if runAll {
		strategy.runTestSequence(mongoCollection, config)
	} else {
		strategy.runTest(mongoCollection, testType, config, fetchDocumentIDs)
	}
}

func createTlsConfigFromFile(tlsCertificate string) (*tls.Config, error) {
	caCert, err := os.ReadFile(tlsCertificate)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to parse certificate from %s", tlsCertificate)
	}

	// Extract hostname from MongoDB URI
	uri, err := options.Client().ApplyURI(mongoURI).Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to parse MongoDB URI: %v", err)
	}

	return &tls.Config{
		RootCAs:    caCertPool,
		ServerName: uri.Host,
	}, nil
}

package main

import (
	"context"
	"flag"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

func main() {
	var threads int
	var docCount int
	var uri string
	var testType string
	var duration int
	var runAll bool
	var largeDocs bool

	flag.IntVar(&threads, "threads", 10, "Number of threads for inserting, updating, upserting, or deleting documents")
	flag.IntVar(&docCount, "docs", 1000, "Total number of documents to insert, update, upsert, or delete")
	flag.StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB URI")
	flag.StringVar(&testType, "type", "insert", "Test type: insert, update, upsert, or delete")
	flag.BoolVar(&runAll, "runAll", false, "Run all tests in order: insert, update, delete, upsert")
	flag.IntVar(&duration, "duration", 0, "Duration in seconds to run the test")
	flag.BoolVar(&largeDocs, "largeDocs", false, "Use large documents for testing")
	flag.Parse()

	var strategy TestingStrategy
	var config TestingConfig

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("benchmarking").Collection("testdata")
	mongoCollection := &MongoDBCollection{Collection: collection}

	// if duration is given by the user we neeed to initialise a strategy for duration testing
	if duration > 0 {
		strategy = DurationTestingStrategy{}
		config = TestingConfig{
			Threads:   threads,
			Duration:  duration,
			LargeDocs: largeDocs,
		}

		if runAll {
			strategy.runTestSequence(mongoCollection, config)
		} else {
			strategy.runTest(mongoCollection, testType, config, fetchDocumentIDs)
		}
	} else {
		strategy = DocCountTestingStrategy{}
		config = TestingConfig{
			Threads:  threads,
			DocCount: docCount,
		}
		if runAll {
			strategy.runTestSequence(mongoCollection, config)
		} else {
			strategy.runTest(mongoCollection, testType, config, fetchDocumentIDs)
		}
	}

}

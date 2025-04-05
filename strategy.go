package main

import "go.mongodb.org/mongo-driver/bson/primitive"

type TestingConfig struct {
	Threads          int
	DocCount         int
	Duration         int
	LargeDocs        bool
	DropDb           bool
	OutputFilePrefix string
	UseIndex         bool
	UseIndexFullText bool
	QueryType        int
	Limit            int
}

type TestingStrategy interface {
	runTestSequence(collection CollectionAPI, config TestingConfig)
	runTestSequenceDoc(collection CollectionAPI, config TestingConfig)
	runTest(collection CollectionAPI, testType string, config TestingConfig, fetchDocIDs func(CollectionAPI, int64, string) ([]primitive.ObjectID, error))
}

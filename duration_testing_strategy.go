package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
)

type DurationTestingStrategy struct{}

func (t DurationTestingStrategy) runTestSequence(collection CollectionAPI, config TestingConfig) {
	tests := []string{"insert", "update"}
	for _, test := range tests {
		t.runTest(collection, test, config, fetchSampledDocIDs)
	}
}

func (t DurationTestingStrategy) runTest(collection CollectionAPI, testType string, config TestingConfig, fetchDocIDs func(CollectionAPI, chan<- primitive.ObjectID, string)) {
	endTime := time.Now().Add(time.Duration(config.Duration) * time.Second)

	insertRate := metrics.NewMeter()
	records := [][]string{
		{"timestamp", "count", "mean_rate", "m1_rate", "m5_rate", "m15_rate"},
	}

	if testType == "insert" && config.DropDb {
		if err := collection.Drop(context.Background()); err != nil {
			log.Fatalf("Failed to clear collection before test: %v", err)
		}
		log.Println("Collection cleared before insert test.")
	}

	var data = make([]byte, 1024*2)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	secondTicker := time.NewTicker(1 * time.Second)
	defer secondTicker.Stop()
	go func() {
		for range secondTicker.C {
			recordMetrics(insertRate, &records)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(config.Threads)

	if testType == "insert" {
		for i := 0; i < config.Threads; i++ {
			go func(threadID int) {
				defer wg.Done()

				for time.Now().Before(endTime) {
					doc := generateDoc(config, threadID, data)
					if _, err := collection.InsertOne(context.Background(), doc); err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Insert failed: %v", err)
					}
				}
			}(i)
		}
	} else if testType == "update" {
		docIDChannel := make(chan primitive.ObjectID, 400000)

		go fetchDocIDs(collection, docIDChannel, testType)

		for i := 0; i < config.Threads; i++ {
			go func() {
				defer wg.Done()
				for docID := range docIDChannel {
					if time.Now().After(endTime) {
						return
					}
					filter := bson.M{"_id": docID}
					update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
					if _, err := collection.UpdateOne(context.Background(), filter, update); err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Update failed for _id %v: %v", docID, err)
					}
				}
			}()
		}
	}

	wg.Wait()
	recordMetrics(insertRate, &records)

	saveMetricsToCSV(records, testType)
	fmt.Printf("Benchmarking completed. Results saved to %s\n", fmt.Sprintf("benchmark_results_%s.csv", testType))
}

func generateDoc(config TestingConfig, threadID int, data []byte) bson.M {
	if config.LargeDocs {
		return bson.M{"threadRunCount": threadID, "rnd": rand.Int63(), "v": 1, "data": data}
	}
	return bson.M{"threadRunCount": threadID, "rnd": rand.Int63(), "v": 1}
}

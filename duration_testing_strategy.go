package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"os"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.mongodb.org/mongo-driver/bson"
)

type DurationTestingStrategy struct{}

func (t DurationTestingStrategy) runTestSequence(collection CollectionAPI, config TestingConfig) {
	tests := []string{"insert", "update"}
	for _, test := range tests {
		t.runTest(collection, test, config, fetchDocumentIDs)
	}
}

func (t DurationTestingStrategy) runTest(collection CollectionAPI, testType string, config TestingConfig, fetchDocIDs func(CollectionAPI, int64, string) ([]primitive.ObjectID, error)) {
	var partitions [][]primitive.ObjectID
	if testType == "insert" {
		if config.DropDb {
			if err := collection.Drop(context.Background()); err != nil {
				log.Fatalf("Failed to clear collection before test: %v", err)
			}
			log.Println("Collection cleared before insert test.")
		} else {
			log.Println("Collection stays. Dropping disabled.")
		}
	} else if testType == "update" {
		docIDs, err := fetchDocIDs(collection, int64(config.DocCount), testType)
		if err != nil {
			log.Fatalf("Failed to fetch document IDs: %v", err)
		}

		if len(docIDs) == 0 {
			log.Fatalf("No document IDs found for update operations")
		}

		// Create partitions from fetched document IDs
		partitions = make([][]primitive.ObjectID, config.Threads)
		for i, id := range docIDs {
			partitions[i%config.Threads] = append(partitions[i%config.Threads], id)
		}
	}

	random := NewRandomizer()

	var doc interface{}
	var data = make([]byte, 1024*2)
	for i := 0; i < len(data); i++ {
		data[i] = byte(random.RandomIntn(256))
	}

	endTime := time.Now().Add(time.Duration(config.Duration) * time.Second)
	insertRate := metrics.NewMeter()
	records := [][]string{{"timestamp", "count", "mean_rate", "m1_rate", "m5_rate", "m15_rate"}}
	secondTicker := time.NewTicker(1 * time.Second)
	defer secondTicker.Stop()
	go func() {
		for range secondTicker.C {
			timestamp := time.Now().Unix()
			count := insertRate.Count()
			mean := insertRate.RateMean()
			m1Rate := insertRate.Rate1()
			m5Rate := insertRate.Rate5()
			m15Rate := insertRate.Rate15()

			log.Printf("Timestamp: %d, Document Count: %d, Mean Rate: %.2f docs/sec, m1_rate: %.2f, m5_rate: %.2f, m15_rate: %.2f",
				timestamp, count, mean, m1Rate, m5Rate, m15Rate)

			record := []string{
				fmt.Sprintf("%d", timestamp),
				fmt.Sprintf("%d", count),
				fmt.Sprintf("%.6f", mean),
				fmt.Sprintf("%.6f", m1Rate),
				fmt.Sprintf("%.6f", m5Rate),
				fmt.Sprintf("%.6f", m15Rate),
			}
			records = append(records, record)
		}
	}()

	// Launch the workload in goroutines
	var wg sync.WaitGroup
	wg.Add(config.Threads)

	if testType == "insert" {
		// Insert operations using generated IDs
		for i := 0; i < config.Threads; i++ {
			threadID := i
			go func(threadID int) {
				defer wg.Done()
				r := NewRandomizer()

				for time.Now().Before(endTime) {
					if config.LargeDocs {
						doc = bson.M{"threadRunCount": threadID, "rnd": r.RandomInt63(), "v": 1, "data": data}

					} else {
						doc = bson.M{"threadRunCount": threadID, "rnd": r.RandomInt63(), "v": 1}
					}
					_, err := collection.InsertOne(context.Background(), doc)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Insert failed: %v", err)
					}
				}
			}(threadID)
		}
	} else {
		for i := 0; i < config.Threads; i++ {
			// Check if the partition is non-empty for this thread
			if len(partitions) <= i || len(partitions[i]) == 0 {
				log.Printf("Skipping empty partition for thread %d in %s operation", i, testType)
				wg.Done()
				continue
			}
			partition := partitions[i]

			go func(partition []primitive.ObjectID) {
				defer wg.Done()
				r := NewRandomizer()

				for time.Now().Before(endTime) {
					docID := partition[r.RandomIntn(len(partition))]

					switch testType {
					case "update":
						filter := bson.M{"_id": docID}
						update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": r.RandomInt63()}}
						_, err := collection.UpdateOne(context.Background(), filter, update)
						if err == nil {
							insertRate.Mark(1)
						} else {
							log.Printf("Update failed for _id %v: %v", docID, err)
						}
					}
				}
			}(partition)
		}
	}

	// Wait for all threads to complete
	wg.Wait()

	// Final metrics recording
	timestamp := time.Now().Unix()
	count := insertRate.Count()
	mean := insertRate.RateMean()
	m1Rate := insertRate.Rate1()
	m5Rate := insertRate.Rate5()
	m15Rate := insertRate.Rate15()

	finalRecord := []string{
		fmt.Sprintf("%d", timestamp),
		fmt.Sprintf("%d", count),
		fmt.Sprintf("%.6f", mean),
		fmt.Sprintf("%.6f", m1Rate),
		fmt.Sprintf("%.6f", m5Rate),
		fmt.Sprintf("%.6f", m15Rate),
	}
	records = append(records, finalRecord)

	// Write metrics to CSV file
	filename := fmt.Sprintf("benchmark_results_%s.csv", testType)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.WriteAll(records); err != nil {
		log.Fatalf("Failed to write records to CSV: %v", err)
	}
	writer.Flush()

	fmt.Printf("Benchmarking completed. Results saved to %s\n", filename)
}

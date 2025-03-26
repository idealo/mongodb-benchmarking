package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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

func (t DurationTestingStrategy) runTestSequenceDoc(collection CollectionAPI, config TestingConfig) {
	tests := []string{"insertdoc", "finddoc"}
	for _, test := range tests {
		t.runTest(collection, test, config, fetchDocumentIDs)
	}
}

func (t DurationTestingStrategy) runTest(collection CollectionAPI, testType string, config TestingConfig, fetchDocIDs func(CollectionAPI, int64, string) ([]primitive.ObjectID, error)) {
	var partitions [][]primitive.ObjectID
	if testType == "insert" || testType == "insertdoc" {
		if config.DropDb {
			if err := collection.Drop(context.Background()); err != nil {
				log.Fatalf("Failed to clear collection before test: %v", err)
			}
			log.Println("Collection cleared before insert test.")
		} else {
			log.Println("Collection stays. Dropping disabled.")
		}

		// todo: prevent code duplicates
		// Create indexes before insertdoc test begins
		if testType == "insertdoc" && config.CreateIndex == true {
			log.Println("Creating indexes for insertdoc benchmark...")

			indexes := []mongo.IndexModel{
				{Keys: bson.D{{Key: "author", Value: 1}}},
				{Keys: bson.D{{Key: "tags", Value: 1}}},
				{Keys: bson.D{{Key: "timestamp", Value: -1}}},
				{Keys: bson.D{{Key: "content", Value: "text"}}},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			mongoColl, ok := collection.(*MongoDBCollection)
			if !ok {
				log.Println("Index creation skipped: Collection is not a MongoDBCollection")
			} else {
				_, err := mongoColl.Indexes().CreateMany(ctx, indexes)
				if err != nil {
					log.Printf("Failed to create indexes: %v", err)
				} else {
					log.Println("Indexes created successfully.")
				}
			}
		}

	} else if testType == "update" {
		docIDs, err := fetchDocIDs(collection, int64(config.DocCount), testType)
		// also possible: fetchDocIDs(collection, 0, testType) because config.DocCount was not set before, so it was always 0
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
	} else if testType == "finddoc" {

		partitions = make([][]primitive.ObjectID, config.Threads)
		for i := 0; i < config.DocCount; i++ {
			partitions[i%config.Threads] = append(partitions[i%config.Threads], primitive.NewObjectID())
		}
	}

	var doc interface{}

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
	queryGenerator := NewQueryGenerator(config.QueryType)

	if testType == "insert" {
		// Insert operations using generated IDs
		for i := 0; i < config.Threads; i++ {
			go func() {
				defer wg.Done()
				docGen := NewDocumentGenerator()

				for time.Now().Before(endTime) {
					if config.LargeDocs {
						//doc = bson.M{"threadRunCount": i, "rnd": rand.Int63(), "v": 1, "data": data}
						doc = docGen.GenerateLarge(i)
					} else {
						//doc = bson.M{"threadRunCount": i, "rnd": rand.Int63(), "v": 1}
						doc = docGen.GenerateSimple(i)
					}
					_, err := collection.InsertOne(context.Background(), doc)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Insert failed: %v", err)
					}
				}
			}()
		}
	} else if testType == "insertdoc" {
		for i := 0; i < config.Threads; i++ {
			go func() {
				defer wg.Done()
				docGen := NewDocumentGenerator()

				for time.Now().Before(endTime) {
					doc = docGen.GenerateComplex(i)
					_, err := collection.InsertOne(context.Background(), doc)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Insertdoc failed: %v", err)
					}
				}
			}()
		}
	} else { // update and finddoc operations
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

				for time.Now().Before(endTime) {
					docID := partition[rand.Intn(len(partition))]

					switch testType {
					case "update":
						filter := bson.M{"_id": docID}
						update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
						_, err := collection.UpdateOne(context.Background(), filter, update)
						if err == nil {
							insertRate.Mark(1)
						} else {
							log.Printf("Update failed for _id %v: %v", docID, err)
						}
					case "finddoc":

						filter := queryGenerator.Generate()

						opts := options.Find().
							SetLimit(10).
							SetProjection(bson.M{
								"_id":       1,
								"author":    1,
								"title":     1,
								"timestamp": 1,
							}).
							SetSort(bson.D{{Key: "timestamp", Value: -1}})

						cursor, err := collection.Find(context.Background(), filter, opts)
						if err != nil {
							log.Printf("Find failed: %v", err)
							continue
						}

						count := 0
						for cursor.Next(context.Background()) {
							var doc bson.M
							if err := cursor.Decode(&doc); err != nil {
								log.Printf("Failed to decode document: %v", err)
								continue
							}

							// Optional: access fields from the document here
							count++
						}

						if err := cursor.Err(); err != nil {
							log.Printf("Cursor error: %v", err)

						}
						cursor.Close(context.Background())
						insertRate.Mark(1)

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
	filenamePrefix := "benchmark_results"
	if config.OutputFilePrefix != "" {
		filenamePrefix = config.OutputFilePrefix
	}

	filename := fmt.Sprintf("%s_%s.csv", filenamePrefix, testType)
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

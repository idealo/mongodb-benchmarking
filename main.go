package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Configurable parameters
	var threads int
	var docCount int
	var uri string
	var testType string

	flag.IntVar(&threads, "threads", 10, "Number of threads for inserting or updating documents")
	flag.IntVar(&docCount, "docs", 1000, "Total number of documents to insert or update")
	flag.StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB URI")
	flag.StringVar(&testType, "type", "insert", "Test type: insert or update")
	flag.Parse()

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("benchmarking").Collection("testdata")

	// Drop collection only for insert tests, to start fresh
	if testType == "insert" {
		if err := collection.Drop(context.Background()); err != nil {
			log.Fatalf("Failed to drop collection: %v", err)
		}
		log.Println("Collection dropped. Starting new insert test...")
	} else {
		log.Println("Starting update test...")
	}

	// Metrics setup
	insertRate := metrics.NewMeter()

	// Create a slice to store CSV rows in memory
	var records [][]string
	// Add CSV header
	records = append(records, []string{"t", "count", "mean", "m1_rate", "m5_rate", "m15_rate", "mean_rate"})

	// Set up a ticker to log the insert rate every second
	secondTicker := time.NewTicker(1 * time.Second)
	defer secondTicker.Stop()

	// Goroutine to log the insert/update rate to memory every second
	go func() {
		for range secondTicker.C {
			// Capture statistics at this second
			timestamp := time.Now().Unix()
			count := insertRate.Count()
			mean := insertRate.RateMean()
			m1Rate := insertRate.Rate1()
			m5Rate := insertRate.Rate5()
			m15Rate := insertRate.Rate15()

			// Log the insert/update rate to the console
			log.Printf("Timestamp: %d, Document Count: %d, Mean Rate: %.2f docs/sec, m1_rate: %.2f, m5_rate: %.2f, m15_rate: %.2f",
				timestamp, count, mean, m1Rate, m5Rate, m15Rate)

			// Append current second stats to the in-memory slice
			record := []string{
				fmt.Sprintf("%d", timestamp),
				fmt.Sprintf("%d", count),
				fmt.Sprintf("%.6f", mean),
				fmt.Sprintf("%.6f", m1Rate),
				fmt.Sprintf("%.6f", m5Rate),
				fmt.Sprintf("%.6f", m15Rate),
				fmt.Sprintf("%.6f", mean), // mean_rate added to CSV
			}
			records = append(records, record)
		}
	}()

	// Start concurrent operations based on the test type
	var wg sync.WaitGroup
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go func(threadID int) {
			defer wg.Done()
			threadDocCount := docCount / threads
			for j := 0; j < threadDocCount; j++ {
				docID := int64(threadID*threadDocCount + j)

				switch testType {
				case "insert":
					// Insert a new document
					doc := bson.M{
						"_id":            docID,
						"threadId":       threadID,
						"threadRunCount": 1,
						"rnd":            rand.Int63(),
						"v":              1,
					}
					_, err := collection.InsertOne(context.Background(), doc)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Insert failed: %v", err)
					}

				case "update":
					// Update an existing document
					filter := bson.M{"_id": docID}
					update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
					_, err := collection.UpdateOne(context.Background(), filter, update)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Update failed: %v", err)
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Write all collected records to CSV after the test finishes
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

	fmt.Println("Benchmarking completed. Check benchmark_results.csv for per-second rates.")
}

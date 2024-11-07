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
	var threads int
	var docCount int
	var uri string

	flag.IntVar(&threads, "threads", 10, "Number of threads for inserting documents")
	flag.IntVar(&docCount, "docs", 1000, "Total number of documents to insert")
	flag.StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB URI")
	flag.Parse()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("benchmarking").Collection("testdata")

	err = collection.Drop(context.Background())
	if err != nil {
		log.Fatalf("Failed to drop collection: %v", err)
	}
	log.Println("Collection dropped. Starting new benchmark test...")

	insertRate := metrics.NewMeter()

	var records [][]string
	records = append(records, []string{"t", "count", "mean", "m1_rate", "m5_rate", "m15_rate", "mean_rate"})

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

			log.Printf("Timestamp: %d, Insert Count: %d, Mean Rate: %.2f docs/sec, m1_rate: %.2f, m5_rate: %.2f, m15_rate: %.2f",
				timestamp, count, mean, m1Rate, m5Rate, m15Rate)

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

	var wg sync.WaitGroup
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go func(threadID int) {
			defer wg.Done()
			threadInsertCount := docCount / threads
			for j := 0; j < threadInsertCount; j++ {
				doc := bson.M{
					"_id":            rand.Int63(),
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
			}
		}(i)
	}

	wg.Wait()

	file, err := os.Create("benchmark_results.csv")
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

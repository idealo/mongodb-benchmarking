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

// CollectionAPI defines an interface for MongoDB operations, allowing for testing
type CollectionAPI interface {
	InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error)
	CountDocuments(ctx context.Context, filter interface{}) (int64, error)
	Drop(ctx context.Context) error
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error)
}

// MongoDBCollection is a wrapper around mongo.Collection to implement CollectionAPI
type MongoDBCollection struct {
	*mongo.Collection
}

func (c *MongoDBCollection) InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	return c.Collection.InsertOne(ctx, document)
}

func (c *MongoDBCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return c.Collection.UpdateOne(ctx, filter, update, opts...)
}

func (c *MongoDBCollection) DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
	return c.Collection.DeleteOne(ctx, filter)
}

func (c *MongoDBCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return c.Collection.CountDocuments(ctx, filter)
}

func (c *MongoDBCollection) Drop(ctx context.Context) error {
	return c.Collection.Drop(ctx)
}

func (c *MongoDBCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return c.Collection.Find(ctx, filter, opts...)
}

// fetchDocumentIDs fetches all document IDs from the collection for delete operations
func fetchDocumentIDs(collection CollectionAPI) ([]int64, error) {
	var docIDs []int64

	cursor, err := collection.Find(context.Background(), bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document IDs: %v", err)
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Failed to decode document: %v", err)
			continue
		}
		if id, ok := result["_id"].(int64); ok {
			docIDs = append(docIDs, id)
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return docIDs, nil
}

func main() {
	var threads int
	var docCount int
	var uri string
	var testType string
	var runAll bool

	flag.IntVar(&threads, "threads", 10, "Number of threads for inserting, updating, upserting, or deleting documents")
	flag.IntVar(&docCount, "docs", 1000, "Total number of documents to insert, update, upsert, or delete")
	flag.StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB URI")
	flag.StringVar(&testType, "type", "insert", "Test type: insert, update, upsert, or delete")
	flag.BoolVar(&runAll, "runAll", false, "Run all tests in order: insert, update, delete, upsert")
	flag.Parse()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("benchmarking").Collection("testdata")
	mongoCollection := &MongoDBCollection{Collection: collection}

	if runAll {
		runTestSequence(mongoCollection, threads, docCount)
	} else {
		runTest(mongoCollection, testType, threads, docCount, fetchDocumentIDs)
	}
}

func runTestSequence(collection CollectionAPI, threads, docCount int) {
	tests := []string{"insert", "update", "delete", "upsert"}
	for _, test := range tests {
		runTest(collection, test, threads, docCount, fetchDocumentIDs)
	}
}

func runTest(collection CollectionAPI, testType string, threads, docCount int, fetchDocIDs func(CollectionAPI) ([]int64, error)) {
	if testType == "insert" || testType == "upsert" {
		if err := collection.Drop(context.Background()); err != nil {
			log.Fatalf("Failed to drop collection: %v", err)
		}
		log.Println("Collection dropped. Starting new rate test...")
	} else {
		log.Printf("Starting %s test...\n", testType)
	}

	insertRate := metrics.NewMeter()
	var records [][]string
	records = append(records, []string{"t", "count", "mean", "m1_rate", "m5_rate", "m15_rate", "mean_rate"})

	var partitions [][]int64 // To hold the document IDs or dummy IDs, partitioned for each thread

	// Prepare partitions based on test type
	switch testType {
	case "delete":
		// Fetch document IDs and partition them
		docIDs, err := fetchDocIDs(collection)
		if err != nil {
			log.Fatalf("Failed to fetch document IDs: %v", err)
		}
		partitions = make([][]int64, threads)
		for i, id := range docIDs {
			partitions[i%threads] = append(partitions[i%threads], id)
		}

	case "insert", "update", "upsert":
		// Generate unique or random IDs for insert/update/upsert
		partitions = make([][]int64, threads)
		for i := 0; i < docCount; i++ {
			id := int64(i)
			if testType == "update" || testType == "upsert" {
				id = int64(rand.Intn(docCount)) // Random ID for update/upsert
			}
			partitions[i%threads] = append(partitions[i%threads], id)
		}
	}

	// Start the ticker just before starting the main workload goroutines
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
				fmt.Sprintf("%.6f", mean),
			}
			records = append(records, record)
		}
	}()

	// Launch threads based on the specific workload type
	var wg sync.WaitGroup
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go func(partition []int64) {
			defer wg.Done()
			for _, docID := range partition {
				switch testType {
				case "insert":
					doc := bson.M{"_id": docID, "threadRunCount": 1, "rnd": rand.Int63(), "v": 1}
					_, err := collection.InsertOne(context.Background(), doc)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Insert failed: %v", err)
					}

				case "update":
					filter := bson.M{"_id": docID}
					update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
					_, err := collection.UpdateOne(context.Background(), filter, update)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Update failed: %v", err)
					}

				case "upsert":
					filter := bson.M{"_id": docID}
					update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
					opts := options.Update().SetUpsert(true)
					_, err := collection.UpdateOne(context.Background(), filter, update, opts)
					if err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Upsert failed: %v", err)
					}

				case "delete":
					filter := bson.M{"_id": docID}
					result, err := collection.DeleteOne(context.Background(), filter)
					if err != nil {
						log.Printf("Delete failed for _id %d: %v", docID, err)
						continue // Move to next document without retrying
					}
					if result.DeletedCount > 0 {
						insertRate.Mark(1)
					}
				}
			}
		}(partitions[i])
	}

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

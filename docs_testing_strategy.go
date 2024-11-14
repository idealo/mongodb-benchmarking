package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
)

type DocCountTestingStrategy struct{}

func (t DocCountTestingStrategy) runTestSequence(collection CollectionAPI, config TestingConfig) {
	tests := []string{"insert", "update", "delete", "upsert"}
	for _, test := range tests {
		t.runTest(collection, test, config, fetchSampledDocIDs)
	}
}

func (t DocCountTestingStrategy) runTest(collection CollectionAPI, testType string, config TestingConfig, fetchDocIDs func(CollectionAPI, chan<- primitive.ObjectID, string)) {
	if testType == "insert" || testType == "upsert" {
		if config.DropDb {
			if err := collection.Drop(context.Background()); err != nil {
				log.Fatalf("Failed to drop collection: %v", err)
			}
			log.Println("Collection dropped. Starting new rate test...")
		} else {
			log.Println("Collection stays. Dropping disabled.")
		}
	} else {
		log.Printf("Starting %s test...\n", testType)
	}

	insertRate := metrics.NewMeter()
	records := [][]string{{"t", "count", "mean", "m1_rate", "m5_rate", "m15_rate", "mean_rate"}}

	var threads = config.Threads
	var docCount = config.DocCount
	var partitions [][]primitive.ObjectID
	var partitionChannels []chan primitive.ObjectID

	switch testType {
	case "insert", "upsert":
		partitions = make([][]primitive.ObjectID, threads)
		for i := 0; i < docCount; i++ {
			partitions[i%threads] = append(partitions[i%threads], primitive.NewObjectID())
		}
	case "update", "delete":
		docIDChannel := make(chan primitive.ObjectID, 40000)
		partitionChannels = make([]chan primitive.ObjectID, threads)

		for i := 0; i < threads; i++ {
			partitionChannels[i] = make(chan primitive.ObjectID, docCount/threads)
		}

		go fetchDocIDs(collection, docIDChannel, testType)

		go func() {
			i := 0
			for id := range docIDChannel {
				partitionChannels[i%threads] <- id
				i++
			}

			for _, ch := range partitionChannels {
				close(ch)
			}
		}()
	}

	secondTicker := time.NewTicker(1 * time.Second)
	defer secondTicker.Stop()
	go func() {
		for range secondTicker.C {
			recordMetrics(insertRate, &records)
		}
	}()

	switch testType {
	case "insert", "upsert":
		runWorkloadWithPartitions(testType, collection, partitions, insertRate)
	case "update", "delete":
		runWorkloadWithChannels(testType, collection, partitionChannels, insertRate)
	}

	finalizeMetrics(insertRate, &records, testType)
}

// runWorkloadWithPartitions runs the workload using pre-generated partitions (for `insert`)
func runWorkloadWithPartitions(testType string, collection CollectionAPI, partitions [][]primitive.ObjectID, insertRate metrics.Meter) {
	var wg sync.WaitGroup
	wg.Add(len(partitions))

	for threadID, partition := range partitions {
		go func(threadID int) {
			defer wg.Done()
			for range partition {
				switch testType {
				case "insert":
					doc := bson.M{"threadRunCount": threadID, "rnd": rand.Int63(), "v": 1}
					if _, err := collection.InsertOne(context.Background(), doc); err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Insert failed: %v", err)
					}
				case "upsert":
					docID := partition[rand.Intn(len(partition)/2)]
					filter := bson.M{"_id": docID}
					update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
					opts := options.Update().SetUpsert(true)
					if _, err := collection.UpdateOne(context.Background(), filter, update, opts); err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Upsert failed for _id %v: %v", docID, err)
					}
				}
			}
		}(threadID)
	}

	wg.Wait()
}

// runWorkloadWithChannels runs the workload using partition channels (for `update`, `delete`, `upsert`)
func runWorkloadWithChannels(testType string, collection CollectionAPI, partitionChannels []chan primitive.ObjectID, insertRate metrics.Meter) {
	var wg sync.WaitGroup
	wg.Add(len(partitionChannels))

	for threadID, partition := range partitionChannels {
		go func(threadID int, partition <-chan primitive.ObjectID) {
			defer wg.Done()
			for docID := range partition {
				switch testType {
				case "update":
					filter := bson.M{"_id": docID}
					update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
					if _, err := collection.UpdateOne(context.Background(), filter, update); err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Update failed for _id %v: %v", docID, err)
					}

				case "upsert":
					filter := bson.M{"_id": docID}
					update := bson.M{"$set": bson.M{"updatedAt": time.Now().Unix(), "rnd": rand.Int63()}}
					opts := options.Update().SetUpsert(true)
					if _, err := collection.UpdateOne(context.Background(), filter, update, opts); err == nil {
						insertRate.Mark(1)
					} else {
						log.Printf("Upsert failed for _id %v: %v", docID, err)
					}

				case "delete":
					filter := bson.M{"_id": docID}
					if result, err := collection.DeleteOne(context.Background(), filter); err == nil && result.DeletedCount > 0 {
						insertRate.Mark(1)
					} else {
						log.Printf("Delete failed for _id %v: %v", docID, err)
					}
				}
			}
		}(threadID, partition)
	}

	wg.Wait()
}

func finalizeMetrics(insertRate metrics.Meter, records *[][]string, testType string) {
	recordMetrics(insertRate, records)
	saveMetricsToCSV(*records, testType)
}

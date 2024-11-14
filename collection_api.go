package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

// CollectionAPI defines an interface for MongoDB operations, allowing for testing
type CollectionAPI interface {
	InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error)
	CountDocuments(ctx context.Context, filter interface{}) (int64, error)
	EstimatedDocumentCount(ctx context.Context) (int64, error)
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error)
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

func (c *MongoDBCollection) EstimatedDocumentCount(ctx context.Context) (int64, error) {
	return c.Collection.EstimatedDocumentCount(ctx)
}

func (c *MongoDBCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return c.Collection.Aggregate(ctx, pipeline, opts...)
}

func fetchSampledDocIDs(collection CollectionAPI, docIDChannel chan<- primitive.ObjectID, testType string) {
	// Get the estimated document count
	estimatedCount, err := collection.EstimatedDocumentCount(context.Background())
	if err != nil {
		log.Fatalf("Failed to get estimated document count: %v", err)
	}

	log.Printf("Estimated document count: %d", estimatedCount)

	if estimatedCount <= 1000 {
		log.Println("Collection is small; fetching all IDs.")
		cursor, err := collection.Find(context.Background(), bson.M{})
		if err != nil {
			log.Fatalf("Failed to fetch document IDs: %v", err)
		}
		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {
			var result struct {
				ID primitive.ObjectID `bson:"_id"`
			}
			if err := cursor.Decode(&result); err != nil {
				log.Printf("Failed to decode document ID: %v", err)
				continue
			}
			docIDChannel <- result.ID
		}
	} else if testType == "delete" {
		log.Println("Collection is large; fetching document IDs in ordered batches for deletion.")

		batchSize := 400000
		totalFetched := 0
		var lastID primitive.ObjectID

		for totalFetched < int(estimatedCount) {
			remaining := int(estimatedCount) - totalFetched
			size := batchSize
			if remaining < batchSize {
				size = remaining
			}

			filter := bson.M{}
			if !lastID.IsZero() {
				filter["_id"] = bson.M{"$gt": lastID}
			}

			options := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(int64(size))
			cursor, err := collection.Find(context.Background(), filter, options)
			if err != nil {
				log.Fatalf("Failed to fetch ordered document IDs: %v", err)
			}
			defer cursor.Close(context.Background())

			for cursor.Next(context.Background()) {
				var result struct {
					ID primitive.ObjectID `bson:"_id"`
				}
				if err := cursor.Decode(&result); err != nil {
					log.Printf("Failed to decode document ID: %v", err)
					continue
				}
				docIDChannel <- result.ID
				lastID = result.ID
				totalFetched++
			}

			if err := cursor.Err(); err != nil {
				log.Printf("Cursor error: %v", err)
			}

			cursor.Close(context.Background())
		}
	} else {
		log.Println("Collection is large; fetching document IDs in random batches.")

		batchSize := 400000
		totalFetched := 0

		for totalFetched < int(estimatedCount) {
			remaining := int(estimatedCount) - totalFetched
			size := batchSize
			if remaining < batchSize {
				size = remaining
			}

			pipeline := []bson.M{{"$sample": bson.M{"size": size}}}
			cursor, err := collection.Aggregate(context.Background(), pipeline)
			if err != nil {
				log.Fatalf("Failed to aggregate document IDs: %v", err)
			}
			defer cursor.Close(context.Background())

			for cursor.Next(context.Background()) {
				var result struct {
					ID primitive.ObjectID `bson:"_id"`
				}
				if err := cursor.Decode(&result); err != nil {
					log.Printf("Failed to decode document ID: %v", err)
					continue
				}
				docIDChannel <- result.ID
				totalFetched++
			}

			log.Println("Fetched", totalFetched, "document IDs")

			if err := cursor.Err(); err != nil {
				log.Printf("Cursor error: %v", err)
			}

			cursor.Close(context.Background())
		}
	}

	close(docIDChannel)
	log.Println("Finished streaming document IDs.")
}

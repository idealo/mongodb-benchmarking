package main

import (
	"context"
	"fmt"
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

func (c *MongoDBCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return c.Collection.Aggregate(ctx, pipeline, opts...)
}

func fetchDocumentIDs(collection CollectionAPI, limit int64, testType string) ([]primitive.ObjectID, error) {
	var docIDs []primitive.ObjectID
	var cursor *mongo.Cursor
	var err error

	switch testType {
	case "insert", "upsert", "delete":
		if limit > 0 {
			cursor, err = collection.Find(context.Background(), bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}).SetLimit(limit))
		} else {
			cursor, err = collection.Find(context.Background(), bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
		}
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
			// Check if `_id` is of type `ObjectId` and add to `docIDs`
			if id, ok := result["_id"].(primitive.ObjectID); ok {
				docIDs = append(docIDs, id)
			} else {
				log.Printf("Skipping document with unsupported _id type: %T", result["_id"])
			}
		}

		if err := cursor.Err(); err != nil {
			return nil, fmt.Errorf("cursor error: %v", err)
		}
	case "update":
		if limit > 0 {
			pipeline := []bson.M{{"$sample": bson.M{"size": limit}}}
			cursor, err = collection.Aggregate(context.Background(), pipeline)
			if err != nil {
				return nil, fmt.Errorf("failed to aggregate documents: %v", err)
			}
			if cursor != nil {
				defer cursor.Close(context.Background()) // Only defer if cursor is valid
			}

			for cursor.Next(context.Background()) {
				var result bson.M
				if err := cursor.Decode(&result); err != nil {
					log.Printf("Failed to decode document: %v", err)
					continue
				}
				// Check if `_id` is of type `ObjectId` and add to `docIDs`
				if id, ok := result["_id"].(primitive.ObjectID); ok {
					docIDs = append(docIDs, id)
				} else {
					log.Printf("Skipping document with unsupported _id type: %T", result["_id"])
				}
			}

		} else {

			cursor, err = collection.Find(context.Background(), bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
			if err != nil {
				return nil, fmt.Errorf("failed to aggregate documents: %v", err)
			}
			if cursor != nil {
				defer cursor.Close(context.Background()) // Only defer if cursor is valid
			}

			for cursor.Next(context.Background()) {
				var result bson.M
				if err := cursor.Decode(&result); err != nil {
					log.Printf("Failed to decode document: %v", err)
					continue
				}
				// Check if `_id` is of type `ObjectId` and add to `docIDs`
				if id, ok := result["_id"].(primitive.ObjectID); ok {
					docIDs = append(docIDs, id)
				} else {
					log.Printf("Skipping document with unsupported _id type: %T", result["_id"])
				}
			}
		}
	}

	return docIDs, nil
}

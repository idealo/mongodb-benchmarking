package main

import (
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// QueryGenerator provides random filters for benchmarking find operations
type QueryGenerator struct {
	rnd     *rand.Rand
	authors []string
	tags    []string
}

// NewQueryGenerator initializes a new QueryGenerator with sample data
func NewQueryGenerator() *QueryGenerator {

	src := rand.NewSource(time.Now().UnixNano())
	return &QueryGenerator{
		rnd: rand.New(src),
		authors: []string{
			"Alice Example", "John Doe", "Maria Sample", "Max Mustermann",
			"Sophie Miller", "Liam Johnson", "Emma Brown", "Noah Davis",
			"Olivia Wilson", "William Martinez",
		},
		tags: []string{"MongoDB", "Benchmark", "CMS", "Database", "Performance",
			"WebApp", "Scalability", "Indexing", "Query Optimization", "Sharding"},
	}
}

// Generate returns a randomly selected filter for a complex find operation
func (g *QueryGenerator) Generate() bson.M {
	switch g.rnd.Intn(3) {
	case 0:
		// Filter by author
		return bson.M{"author": g.authors[g.rnd.Intn(len(g.authors))]}
	case 1:
		// Filter by tag (element match)
		return bson.M{"tags": bson.M{"$elemMatch": bson.M{"$eq": g.tags[g.rnd.Intn(len(g.tags))]}}}
	case 2:
		// Filter by timestamp greater than some random date in the past year
		past := time.Now().Add(-time.Duration(g.rnd.Intn(365*12)) * time.Hour)
		return bson.M{"timestamp": bson.M{"$gt": past}}
	case 3:
		// Full-text search
		return bson.M{"$text": bson.M{"$search": g.tags[g.rnd.Intn(len(g.tags))]}}
	default:
		return bson.M{}
	}
}

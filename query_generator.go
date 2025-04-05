package main

import (
	"fmt"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// QueryGenerator provides methods for generating randomized MongoDB query filters,
// used for benchmarking various types of find operations. It supports multiple
// query strategies, including filtering by author, tags, timestamp, or full-text search.
type QueryGenerator struct {
	rnd        *rand.Rand
	queryType  int
	useIndex   bool
	useIndexFT bool
	authors    []string
	guests     []string
	tags       []string
}

// NewQueryGenerator initializes and returns a new QueryGenerator.
// It accepts a queryType parameter to control the query strategy:
// if queryType is 0, a random query type will be chosen at each call to Generate.
func NewQueryGenerator(queryType int, useIndex bool, useIndexFT bool) *QueryGenerator {
	src := rand.NewSource(time.Now().UnixNano())
	// generate guests like: guest_0001, ..., guest_9999
	guests := make([]string, 10000)
	for i := range guests {
		guests[i] = fmt.Sprintf("guest_%04d", i)
	}

	return &QueryGenerator{
		rnd:        rand.New(src),
		queryType:  queryType,
		useIndex:   useIndex,
		useIndexFT: useIndexFT,
		authors: []string{
			"Alice Example", "John Doe", "Maria Sample", "Max Mustermann",
			"Sophie Miller", "Liam Johnson", "Emma Brown", "Noah Davis",
			"Olivia Wilson", "William Martinez",
		},
		guests: guests,
		tags: []string{"MongoDB", "Benchmark", "CMS", "Database", "Performance",
			"WebApp", "Scalability", "Indexing", "Query Optimization", "Sharding"},
	}
}

// Generate returns a randomized MongoDB filter (bson.M) based on the configured query type.
// Supported query types:
//
//	1 - Match a single author
//	2 - Match a single guest author
//	3 - Match a single tag using $elemMatch
//	4 - Filter documents with timestamps greater than a recent random time
//	5 - Perform a full-text search on tags
//
// If queryType is set to 0, one of the above is chosen at random.
func (g *QueryGenerator) Generate() bson.M {
	var queryType int

	if g.queryType == 0 {
		if g.useIndex && g.useIndexFT {

			queryType = g.rnd.Intn(5) + 1
		} else {
			queryType = g.rnd.Intn(4) + 1
		}
	} else {
		queryType = g.queryType
	}
	switch queryType {
	case 1:
		// Filter by author
		return bson.M{"author": g.authors[g.rnd.Intn(len(g.authors))]}
	case 2:
		// Filter by guest author
		return bson.M{"guest": g.guests[g.rnd.Intn(len(g.guests))]}
	case 3:
		// Filter by tag (element match)
		return bson.M{"tags": bson.M{"$elemMatch": bson.M{"$eq": g.tags[g.rnd.Intn(len(g.tags))]}}}
	case 4:
		// Filter by timestamp greater than some random date in the past year
		past := time.Now().Add(-time.Duration(g.rnd.Intn(365*12)) * time.Hour)
		return bson.M{"timestamp": bson.M{"$gt": past}}
	case 5:
		// Full-text search, only if the index is used
		return bson.M{"$text": bson.M{"$search": g.tags[g.rnd.Intn(len(g.tags))]}}
	default:
		return bson.M{}
	}
}

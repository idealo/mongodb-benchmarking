package main

import (
	"math/rand"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// DocumentGenerator provides methods for generating randomized BSON documents
// of varying complexity and size. It is intended for use in benchmarking scenarios,
// such as testing database insert, update, and query performance.
//
// The generator includes:
//   - A random number generator seeded at initialization
//   - A reusable byte buffer for large binary payloads
//   - Predefined pools of authors, tags, categories, and lorem ipsum text
//
// Methods like GenerateSimple, GenerateLarge, and GenerateComplex allow for flexible
// document creation depending on the desired test case.
// The generated documents are structured as BSON maps, suitable for direct use with MongoDB or similar databases.
type DocumentGenerator struct {
	rnd      *rand.Rand
	data     []byte
	tags     []string
	authors  []string
	category []string
	lorem    []string
}

// NewDocumentGenerator initializes and returns a new DocumentGenerator with pre-filled
// random seed, reusable byte buffer for large document generation, and predefined pools
// of tags, authors, categories, and lorem ipsum phrases for use in synthetic document creation.
func NewDocumentGenerator() *DocumentGenerator {
	// Init once
	src := rand.NewSource(time.Now().UnixNano())
	return &DocumentGenerator{
		rnd:      rand.New(src),
		data:     make([]byte, 1024*2),
		tags:     []string{"MongoDB", "Benchmark", "CMS", "Database", "Performance", "WebApp", "Scalability", "Indexing", "Query Optimization", "Sharding"},
		authors:  []string{"Alice Example", "John Doe", "Maria Sample", "Max Mustermann", "Sophie Miller", "Liam Johnson", "Emma Brown", "Noah Davis", "Olivia Wilson", "William Martinez"},
		category: []string{"Tech", "Business", "Science", "Health", "Sports", "Education"},
		lorem: []string{
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
			"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
			"Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
		},
	}
}

// GenerateSimple returns a minimal BSON document containing basic metadata fields.
// This is intended for lightweight benchmarking scenarios where document size and complexity
// are kept intentionally low. The threadRunCount helps identify the originating thread.
func (g *DocumentGenerator) GenerateSimple(threadRunCount int) bson.M {
	return bson.M{"threadRunCount": threadRunCount,
		"rnd": g.rnd.Int63(),
		"v":   1,
	}
}

// GenerateLarge returns a BSON document containing a large binary data payload,
// which is useful for benchmarking insert and storage performance with large documents.
// The data field is filled with random bytes, and threadRunCount is included for tracking
// document origin in multi-threaded scenarios.
func (g *DocumentGenerator) GenerateLarge(threadRunCount int) bson.M {
	for i := range g.data {
		g.data[i] = byte(g.rnd.Intn(256))
	}
	return bson.M{"threadRunCount": threadRunCount,
		"rnd":  g.rnd.Int63(),
		"v":    1,
		"data": g.data,
	}
}

// GenerateComplex creates a synthetic, document-like bson.M object with various randomized fields,
// simulating a realistic and content-rich record for testing purposes. The generated document includes
// metadata such as author, co-authors, tags, category, views, and interactions, along with generated
// title, summary, and content text.
//
// The threadRunCount parameter is included as a field to support multi-threaded benchmarking scenarios.
func (g *DocumentGenerator) GenerateComplex(threadRunCount int) bson.M {
	numTags := g.rnd.Intn(3) + 4      // 4–6 tags
	numCoAuthors := g.rnd.Intn(3) + 1 // 1–3 co-authors

	tags := g.randomSample(g.tags, numTags)
	coAuthors := g.randomSample(g.authors, numCoAuthors)
	category := g.category[g.rnd.Intn(len(g.category))]
	author := g.authors[g.rnd.Intn(len(g.authors))]

	return bson.M{
		"threadRunCount": threadRunCount,
		"rnd":            g.rnd.Int63(),
		"v":              1,
		"title":          g.generateLoremIpsum(30),
		"author":         author,
		"co_authors":     coAuthors,
		"summary":        g.generateLoremIpsum(100),
		"content":        g.generateLoremIpsum(2000 + g.rnd.Intn(3000)),
		"tags":           tags,
		"category":       category,
		"timestamp":      g.randomPastTime(),
		"views":          g.rnd.Intn(10000),
		"comments":       g.rnd.Intn(500),
		"likes":          g.rnd.Intn(1000),
		"shares":         g.rnd.Intn(200),
	}
}

// generateLoremIpsum returns a pseudo-random string of at least minLen characters,
// composed of lorem ipsum words and occasional tag words from the generator's pool.
// The result is trimmed to exactly minLen characters.
//
// This function uses a strings.Builder for efficient string construction
// and biases the output toward lorem words, occasionally injecting tags (~10% chance).
// The generated string is not guaranteed to be grammatically correct or meaningful.
func (g *DocumentGenerator) generateLoremIpsum(minLen int) string {
	var sb strings.Builder
	sb.Grow(minLen + 64) // hint size

	for sb.Len() < minLen {
		var s string
		if g.rnd.Float32() < 0.1 {
			s = g.tags[g.rnd.Intn(len(g.tags))]
		} else {
			s = g.lorem[g.rnd.Intn(len(g.lorem))]
		}
		sb.WriteString(s)
		sb.WriteByte(' ')
	}

	return sb.String()[:minLen]
}

// randomSample returns a random subset of n elements from the given list,
// using reservoir sampling to ensure uniform distribution without shuffling the entire list.
// This approach is efficient when n is much smaller than the list size.
func (g *DocumentGenerator) randomSample(list []string, n int) []string {
	// Use reservoir sampling to avoid full shuffle (faster for small n)
	if n >= len(list) {
		return list
	}
	result := make([]string, n)
	copy(result, list[:n])
	for i := n; i < len(list); i++ {
		j := g.rnd.Intn(i + 1)
		if j < n {
			result[j] = list[i]
		}
	}
	return result
}

// randomPastTime returns a random time within the past two years.
// This is used to simulate realistic document timestamps for benchmarking.
func (g *DocumentGenerator) randomPastTime() time.Time {
	// Precompute time bounds if possible
	daysAgo := g.rnd.Intn(365 * 2)
	return time.Now().AddDate(0, 0, -daysAgo)
}

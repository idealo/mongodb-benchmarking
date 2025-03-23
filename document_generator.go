package main

import (
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DocumentGenerator struct {
	rnd      *rand.Rand
	data     []byte
	tags     []string
	authors  []string
	category []string
	lorem    []string
}

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

func (g *DocumentGenerator) GenerateSimple(threadRunCount int) bson.M {
	return bson.M{"threadRunCount": threadRunCount,
		"rnd": g.rnd.Int63(),
		"v":   1,
	}
}

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

func (g *DocumentGenerator) GenerateComplex(threadRunCount int) bson.M {
	numTags := g.rnd.Intn(3) + 4      // 4–6 tags
	numCoAuthors := g.rnd.Intn(3) + 1 // 1–3 co-authors

	tags := g.randomSample(g.tags, numTags)
	coAuthors := g.randomSample(g.authors, numCoAuthors)
	category := g.category[g.rnd.Intn(len(g.category))]
	author := g.authors[g.rnd.Intn(len(g.authors))]

	return bson.M{
		"_id":            primitive.NewObjectID(),
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
		"timestamp":      time.Now().Add(-time.Duration(g.rnd.Intn(365*2)) * 24 * time.Hour),
		"views":          g.rnd.Intn(10000),
		"comments":       g.rnd.Intn(500),
		"likes":          g.rnd.Intn(1000),
		"shares":         g.rnd.Intn(200),
	}
}

func (g *DocumentGenerator) generateLoremIpsum(minLen int) string {
	text := ""
	for len(text) < minLen {
		if g.rnd.Float32() < 0.1 { // 10% chance to insert a tag
			text += g.tags[g.rnd.Intn(len(g.tags))] + " "
		} else {
			text += g.lorem[g.rnd.Intn(len(g.lorem))] + " "
		}
	}
	return text[:minLen]
}

func (g *DocumentGenerator) randomSample(list []string, n int) []string {
	g.rnd.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
	return list[:n]
}

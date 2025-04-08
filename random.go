package main

import (
	"math/rand"
	"time"
)

type Randomizer struct {
	rnd *rand.Rand
}

// NewRandomizer initializes a new Randomizer instance with a seeded random number generator.
func NewRandomizer() *Randomizer {
	src := rand.NewSource(time.Now().UnixNano())
	return &Randomizer{
		rnd: rand.New(src),
	}
}

// RandomInt63 returns a non-negative pseudo-random 63-bit integer as an int64
func (r *Randomizer) RandomInt63() int64 {
	return r.rnd.Int63()
}

// RandomIntn returns a non-negative pseudo-random int in [0,n)
func (r *Randomizer) RandomIntn(n int) int {
	return r.rnd.Intn(n)
}

package uuids

import (
	"math/rand"

	"github.com/google/uuid"
	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/random"
)

// UUID is a UUID encoded as a 36 character string using lowercase hex characters
type UUID string

// NewV4 returns a new v4 UUID
func NewV4() UUID {
	return currentGenerator.NextV4()
}

// NewV4 returns a new v7 UUID
func NewV7() UUID {
	return currentGenerator.NextV7()
}

// Generator is something that can generate UUIDs
type Generator interface {
	NextV4() UUID
	NextV7() UUID
}

// defaultGenerator generates a random v4 UUID using a 3rd party library
type defaultGenerator struct{}

// NextV4 returns the next v4 UUID
func (g defaultGenerator) NextV4() UUID {
	return must(uuid.NewRandom())
}

// NextV7 returns the next v7 UUID
func (g defaultGenerator) NextV7() UUID {
	return must(uuid.NewV7())
}

// DefaultGenerator is the default generator for calls to NewUUID
var DefaultGenerator Generator = defaultGenerator{}
var currentGenerator = DefaultGenerator

// SetGenerator sets the generator used by UUID4()
func SetGenerator(generator Generator) {
	currentGenerator = generator
}

// generates a seedable random v4 UUID using math/rand
type seededGenerator struct {
	rnd *rand.Rand
	now dates.NowFunc
}

// NewSeededGenerator creates a new UUID generator that uses the given seed for the random component and the time source
// for the time component (only applies to v7)
func NewSeededGenerator(seed int64, now dates.NowFunc) Generator {
	return &seededGenerator{rnd: random.NewSeededGenerator(seed), now: now}
}

// NextV4 returns the next v4 UUID
func (g *seededGenerator) NextV4() UUID {
	return must(uuid.NewRandomFromReader(g.rnd))
}

// NextV7 returns the next v7 UUID
func (g *seededGenerator) NextV7() UUID {
	u := uuid.Must(uuid.NewRandomFromReader(g.rnd))

	nano := g.now().UnixNano()
	t := nano / 1_000_000
	s := (nano - t*1_000_000) >> 8

	u[0] = byte(t >> 40)
	u[1] = byte(t >> 32)
	u[2] = byte(t >> 24)
	u[3] = byte(t >> 16)
	u[4] = byte(t >> 8)
	u[5] = byte(t)
	u[6] = 0x70 | (0x0F & byte(s>>8))
	u[7] = byte(s)

	return must(u, nil)
}

func must(u uuid.UUID, err error) UUID {
	return UUID(uuid.Must(u, err).String())
}

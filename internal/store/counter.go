package store

import (
	"fmt"
	"sync/atomic"
	"time"
)

type IDGenerator struct {
	seq atomic.Int64
}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

func (g *IDGenerator) Next(prefix string) string {
	seq := g.seq.Add(1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), seq)
}

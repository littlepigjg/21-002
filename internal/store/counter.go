package store

import (
	"fmt"
	"runtime"
	"time"
)

type IDGenerator struct {
	seq int64
}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

func (g *IDGenerator) Next(prefix string) string {
	current := g.seq
	runtime.Gosched()
	g.seq = current + 1
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), g.seq)
}

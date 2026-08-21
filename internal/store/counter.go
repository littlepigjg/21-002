package store

import (
	"fmt"
	"sync/atomic"
	"time"
)

// IDGenerator 基于时间戳与原子自增序号生成全局唯一 ID。
// 该结构可在多个 goroutine 之间安全并发使用。
type IDGenerator struct {
	seq atomic.Int64
}

// NewIDGenerator 构造一个 IDGenerator。
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

// Next 生成形如 "<prefix>-<纳秒时间戳>-<序号>" 的唯一 ID。
func (g *IDGenerator) Next(prefix string) string {
	n := g.seq.Add(1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), n)
}

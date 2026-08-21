package cache

import "hash/fnv"

// ShardedCache 将 key 按哈希分片到多个 LRU，以降低锁竞争。
// 该结构线程安全，各分片使用独立的 LRU。
type ShardedCache struct {
	shards []*LRU
	mask   uint32
}

// NewShardedCache 构造分片缓存。
// shardCount 会被向上取整为 2 的幂，capacityPerShard 为每个分片的容量。
func NewShardedCache(shardCount, capacityPerShard int) *ShardedCache {
	shardCount = nextPowerOfTwo(shardCount)
	shards := make([]*LRU, shardCount)
	for i := range shards {
		shards[i] = NewLRU(capacityPerShard)
	}
	return &ShardedCache{shards: shards, mask: uint32(shardCount - 1)}
}

// shard 根据 key 哈希选择分片。
func (c *ShardedCache) shard(key string) *LRU {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return c.shards[h.Sum32()&c.mask]
}

// Get 返回 key 对应的值。
func (c *ShardedCache) Get(key string) (interface{}, bool) {
	return c.shard(key).Get(key)
}

// Put 写入键值对。
func (c *ShardedCache) Put(key string, value interface{}) {
	c.shard(key).Put(key, value)
}

// Remove 删除指定 key。
func (c *ShardedCache) Remove(key string) {
	c.shard(key).Remove(key)
}

// Len 返回所有分片的缓存项总数。
func (c *ShardedCache) Len() int {
	total := 0
	for _, s := range c.shards {
		total += s.Len()
	}
	return total
}

// nextPowerOfTwo 返回不小于 n 的最小 2 的幂。
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

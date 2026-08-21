// Package cache 提供内存缓存能力，用于加速热点数据读取。
package cache

// Cache 定义通用键值缓存接口。
type Cache interface {
	Get(key string) (interface{}, bool)
	Put(key string, value interface{})
	Remove(key string)
	Len() int
}

// 编译期断言 LRU 实现了 Cache 接口。
var _ Cache = (*LRU)(nil)

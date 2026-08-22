// Package config 提供应用配置的默认值、环境变量加载与访问方法。
package config

import (
	"fmt"
	"time"
)

// Config 聚合了服务运行所需的全部配置项。
type Config struct {
	// Host 与 Port 决定 HTTP 服务监听地址。
	Host string
	Port int

	// 文本处理相关参数。
	MaxKeywordCount    int // 每次分析提取的关键词上限
	MaxSummarySentences int // 每次分析生成的摘要句子数上限
	MaxArticleLength   int // 单篇文章允许的最大字符数
	MinSentenceLength  int // 参与打分的最短句子长度（按 token 计）
	TextRankMaxIter    int // TextRank 迭代次数上限
	TextRankDamping    float64 // TextRank 阻尼系数

	// 任务队列相关参数。
	WorkerCount          int // 并发 worker 数量
	QueueCapacity        int // 内存任务队列容量
	ResultCacheCapacity  int // 结果缓存容量

	// HTTP 服务器超时控制。
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	// 日志相关。
	LogLevel  string
	LogFormat string
}

// Default 返回一份经过调优的默认配置。
func Default() *Config {
	return &Config{
		Host:                "0.0.0.0",
		Port:                8080,
		MaxKeywordCount:     10,
		MaxSummarySentences: 5,
		MaxArticleLength:    100000,
		MinSentenceLength:   3,
		TextRankMaxIter:     30,
		TextRankDamping:     0.85,
		WorkerCount:         4,
		QueueCapacity:       256,
		ResultCacheCapacity: 512,
		ReadTimeout:         10 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
		ShutdownTimeout:     15 * time.Second,
		LogLevel:            "info",
		LogFormat:           "json",
	}
}

// Addr 返回 HTTP 服务的监听地址，形如 "0.0.0.0:8080"。
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Validate 校验配置项，返回第一个非法项的描述。
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("config: invalid port %d", c.Port)
	}
	if c.WorkerCount <= 0 {
		return fmt.Errorf("config: worker count must be positive")
	}
	if c.QueueCapacity <= 0 {
		return fmt.Errorf("config: queue capacity must be positive")
	}
	if c.TextRankMaxIter <= 0 {
		return fmt.Errorf("config: textrank max iter must be positive")
	}
	if c.TextRankDamping <= 0 || c.TextRankDamping >= 1 {
		return fmt.Errorf("config: textrank damping must be in (0,1)")
	}
	return nil
}

package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Load 从环境变量读取配置，未显式设置的项回退到默认值。
// 支持的环境变量前缀统一为 SERVER_。
func Load() *Config {
	cfg := Default()

	cfg.Host = envString("SERVER_HOST", cfg.Host)
	cfg.Port = envInt("SERVER_PORT", cfg.Port)

	cfg.MaxKeywordCount = envInt("SERVER_MAX_KEYWORD_COUNT", cfg.MaxKeywordCount)
	cfg.MaxSummarySentences = envInt("SERVER_MAX_SUMMARY_SENTENCES", cfg.MaxSummarySentences)
	cfg.MaxArticleLength = envInt("SERVER_MAX_ARTICLE_LENGTH", cfg.MaxArticleLength)
	cfg.MinSentenceLength = envInt("SERVER_MIN_SENTENCE_LENGTH", cfg.MinSentenceLength)
	cfg.TextRankMaxIter = envInt("SERVER_TEXTRANK_MAX_ITER", cfg.TextRankMaxIter)
	cfg.TextRankDamping = envFloat("SERVER_TEXTRANK_DAMPING", cfg.TextRankDamping)

	cfg.WorkerCount = envInt("SERVER_WORKER_COUNT", cfg.WorkerCount)
	cfg.QueueCapacity = envInt("SERVER_QUEUE_CAPACITY", cfg.QueueCapacity)

	cfg.ReadTimeout = envDuration("SERVER_READ_TIMEOUT", cfg.ReadTimeout)
	cfg.WriteTimeout = envDuration("SERVER_WRITE_TIMEOUT", cfg.WriteTimeout)
	cfg.IdleTimeout = envDuration("SERVER_IDLE_TIMEOUT", cfg.IdleTimeout)
	cfg.ShutdownTimeout = envDuration("SERVER_SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout)

	cfg.LogLevel = envString("SERVER_LOG_LEVEL", cfg.LogLevel)
	cfg.LogFormat = envString("SERVER_LOG_FORMAT", cfg.LogFormat)

	return cfg
}

// envString 读取字符串环境变量，空值返回默认值。
func envString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

// envInt 读取整型环境变量，解析失败返回默认值。
func envInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

// envFloat 读取浮点环境变量，解析失败返回默认值。
func envFloat(key string, def float64) float64 {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return def
	}
	return f
}

// envDuration 读取时长环境变量，支持 "5s"、"300ms" 等 Go duration 格式。
func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return d
}

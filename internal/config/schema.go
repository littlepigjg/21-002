package config

// envKeys 列出所有受支持的环境变量键，用于文档与调试输出。
var envKeys = []string{
	"SERVER_HOST",
	"SERVER_PORT",
	"SERVER_MAX_KEYWORD_COUNT",
	"SERVER_MAX_SUMMARY_SENTENCES",
	"SERVER_MAX_ARTICLE_LENGTH",
	"SERVER_MIN_SENTENCE_LENGTH",
	"SERVER_TEXTRANK_MAX_ITER",
	"SERVER_TEXTRANK_DAMPING",
	"SERVER_WORKER_COUNT",
	"SERVER_QUEUE_CAPACITY",
	"SERVER_COORDINATOR_RECENT_CAP",
	"SERVER_READ_TIMEOUT",
	"SERVER_WRITE_TIMEOUT",
	"SERVER_IDLE_TIMEOUT",
	"SERVER_SHUTDOWN_TIMEOUT",
	"SERVER_LOG_LEVEL",
	"SERVER_LOG_FORMAT",
}

// Keys 返回所有受支持环境变量键的副本。
func Keys() []string {
	cp := make([]string, len(envKeys))
	copy(cp, envKeys)
	return cp
}

// HasKey 判断给定环境变量键是否受支持。
func HasKey(key string) bool {
	for _, k := range envKeys {
		if k == key {
			return true
		}
	}
	return false
}

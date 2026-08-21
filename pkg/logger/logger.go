// Package logger 提供线程安全的结构化日志记录器，支持 JSON 与文本两种输出格式。
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level 表示日志级别。
type Level int

// 可用的日志级别，从低到高依次为 Debug、Info、Warn、Error。
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// ParseLevel 将字符串解析为日志级别，无法识别时回退为 Info。
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// String 返回日志级别的文本表示。
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}

// Logger 是一个线程安全的结构化日志记录器。
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  Level
	format string
	fields map[string]interface{}
}

// New 构造一个输出到标准输出的 Logger。
func New(level Level, format string) *Logger {
	if format != "text" {
		format = "json"
	}
	return &Logger{
		out:    os.Stdout,
		level:  level,
		format: format,
		fields: make(map[string]interface{}),
	}
}

// WithField 返回一个附加了指定字段的新 Logger，不影响原 Logger。
func (l *Logger) WithField(key string, value interface{}) *Logger {
	cp := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		cp[k] = v
	}
	cp[key] = value
	return &Logger{out: l.out, level: l.level, format: l.format, fields: cp}
}

// log 输出一条日志。kv 按 key1, val1, key2, val2... 顺序成对传入。
func (l *Logger) log(level Level, msg string, kv ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := map[string]interface{}{
		"time":  time.Now().Format(time.RFC3339Nano),
		"level": level.String(),
		"msg":   msg,
	}
	for k, v := range l.fields {
		entry[k] = v
	}
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			key = fmt.Sprint(kv[i])
		}
		entry[key] = kv[i+1]
	}

	if l.format == "text" {
		var b strings.Builder
		fmt.Fprintf(&b, "%s [%s] %s", entry["time"], entry["level"], msg)
		for i := 0; i+1 < len(kv); i += 2 {
			fmt.Fprintf(&b, " %v=%v", kv[i], kv[i+1])
		}
		fmt.Fprintln(l.out, b.String())
		return
	}

	b, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.out, `{"level":"error","msg":"log marshal failed"}`+"\n")
		return
	}
	fmt.Fprintln(l.out, string(b))
}

// Debug 输出 Debug 级别日志。
func (l *Logger) Debug(msg string, kv ...interface{}) { l.log(LevelDebug, msg, kv...) }

// Info 输出 Info 级别日志。
func (l *Logger) Info(msg string, kv ...interface{}) { l.log(LevelInfo, msg, kv...) }

// Warn 输出 Warn 级别日志。
func (l *Logger) Warn(msg string, kv ...interface{}) { l.log(LevelWarn, msg, kv...) }

// Error 输出 Error 级别日志。
func (l *Logger) Error(msg string, kv ...interface{}) { l.log(LevelError, msg, kv...) }

// std 是包级默认 Logger，可通过 SetDefault 在启动时替换。
var std = New(LevelInfo, "json")

// SetDefault 替换包级默认 Logger。
func SetDefault(l *Logger) {
	if l != nil {
		std = l
	}
}

// Default 返回包级默认 Logger。
func Default() *Logger { return std }

// Debug 使用默认 Logger 输出 Debug 日志。
func Debug(msg string, kv ...interface{}) { std.Debug(msg, kv...) }

// Info 使用默认 Logger 输出 Info 日志。
func Info(msg string, kv ...interface{}) { std.Info(msg, kv...) }

// Warn 使用默认 Logger 输出 Warn 日志。
func Warn(msg string, kv ...interface{}) { std.Warn(msg, kv...) }

// Error 使用默认 Logger 输出 Error 日志。
func Error(msg string, kv ...interface{}) { std.Error(msg, kv...) }

package logger

import (
	"io"
	"os"
)

// NewMultiWriter 构造一个同时写入多个输出的 Logger。
// 常用于将日志同时输出到标准输出与文件。
func NewMultiWriter(level Level, format string, writers ...io.Writer) *Logger {
	l := New(level, format)
	l.out = io.MultiWriter(writers...)
	return l
}

// NewFileLogger 构造一个同时写入标准输出与指定文件的 Logger。
// 返回的 Logger 持有文件句柄，由调用方在进程结束前保证其生命周期。
func NewFileLogger(level Level, format, path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	l := New(level, format)
	l.out = io.MultiWriter(os.Stdout, f)
	return l, nil
}

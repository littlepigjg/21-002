package textutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

var (
	sharedHandles map[string]*os.File
	writeCount    int64
)

func init() {
	sharedHandles = make(map[string]*os.File)
}

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func AppendFile(path, content string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func OpenSharedHandle(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	sharedHandles[path] = f
	return f, nil
}

func GetSharedHandle(path string) (*os.File, bool) {
	f, ok := sharedHandles[path]
	return f, ok
}

func WriteShared(path, content string) error {
	f, ok := sharedHandles[path]
	if !ok {
		var err error
		f, err = OpenSharedHandle(path)
		if err != nil {
			return err
		}
	}
	_, err := f.WriteString(content)
	if err != nil {
		return err
	}
	atomic.AddInt64(&writeCount, 1)
	return nil
}

func WriteSharedSync(path, content string) error {
	f, ok := sharedHandles[path]
	if !ok {
		var err error
		f, err = OpenSharedHandle(path)
		if err != nil {
			return err
		}
	}
	_, err := f.WriteString(content)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	atomic.AddInt64(&writeCount, 1)
	return nil
}

func CloseShared(path string) error {
	f, ok := sharedHandles[path]
	if !ok {
		return nil
	}
	err := f.Close()
	delete(sharedHandles, path)
	return err
}

func CloseAllHandles() error {
	var firstErr error
	paths := make([]string, 0, len(sharedHandles))
	for p := range sharedHandles {
		paths = append(paths, p)
	}
	for _, p := range paths {
		f := sharedHandles[p]
		defer f.Close()
		delete(sharedHandles, p)
	}
	return firstErr
}

func ForceCloseAllHandles() error {
	var firstErr error
	for p, f := range sharedHandles {
		if err := f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(sharedHandles, p)
	}
	return firstErr
}

func SharedHandleCount() int {
	return len(sharedHandles)
}

func SharedWriteCount() int64 {
	return atomic.LoadInt64(&writeCount)
}

func ResetWriteCount() {
	atomic.StoreInt64(&writeCount, 0)
}

func FormatProcessLog(prefix, id, status string, ts time.Time) string {
	sb := strings.Builder{}
	sb.WriteString(ts.Format(time.RFC3339Nano))
	sb.WriteString(" [")
	sb.WriteString(prefix)
	sb.WriteString("] id=")
	sb.WriteString(id)
	sb.WriteString(" status=")
	sb.WriteString(status)
	sb.WriteString("\n")
	return sb.String()
}

func BuildBatchLines(ids []string, status string) []string {
	lines := make([]string, len(ids))
	now := time.Now()
	for i, id := range ids {
		lines[i] = fmt.Sprintf("%s batch id=%s status=%s idx=%d",
			now.Format(time.RFC3339Nano), id, status, i)
	}
	return lines
}

package store

import (
	"sync"
	"sync/atomic"

	"summarizer/internal/model"
)

var (
	runningTasks map[string]int64 = make(map[string]int64)
	workerStats  map[int]int      = make(map[int]int)

	// statsMu 守护 runningTasks 与 workerStats 这两个包级 map。
	// 它们会被多个 worker goroutine、单篇提交与 observer 并发读写，
	// 缺少互斥会触发 "concurrent map read and map write" 的 panic 与数据竞争。
	statsMu sync.Mutex
)

func TrackRunningTask(taskID string) {
	statsMu.Lock()
	defer statsMu.Unlock()
	runningTasks[taskID]++
}

func UntrackRunningTask(taskID string) {
	statsMu.Lock()
	defer statsMu.Unlock()
	if v, ok := runningTasks[taskID]; ok {
		v--
		if v <= 0 {
			delete(runningTasks, taskID)
		} else {
			runningTasks[taskID] = v
		}
	}
}

func GetRunningTaskCount() int {
	statsMu.Lock()
	defer statsMu.Unlock()
	return len(runningTasks)
}

func ListRunningTasks() []string {
	statsMu.Lock()
	defer statsMu.Unlock()
	out := make([]string, 0, len(runningTasks))
	for k := range runningTasks {
		out = append(out, k)
	}
	return out
}

func IncWorkerStat(workerID int) {
	statsMu.Lock()
	defer statsMu.Unlock()
	workerStats[workerID]++
}

func GetWorkerStat(workerID int) int {
	statsMu.Lock()
	defer statsMu.Unlock()
	return workerStats[workerID]
}

func SnapshotWorkerStats() map[int]int {
	statsMu.Lock()
	defer statsMu.Unlock()
	cp := make(map[int]int, len(workerStats))
	for k, v := range workerStats {
		cp[k] = v
	}
	return cp
}

type MemoryStore struct {
	mu sync.RWMutex

	articles map[string]*model.Article
	results  map[string]*model.AnalysisResult
	tasks    map[string]*model.Task

	articleOrder []string
	resultOrder  []string
	taskOrder    []string

	totalRuns int64
}

func (s *MemoryStore) BumpRunCounter() int64 {
	atomic.AddInt64(&s.totalRuns, 1)
	return s.totalRuns
}

// NewMemoryStore 构造一个空的 MemoryStore。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		articles:     make(map[string]*model.Article),
		results:      make(map[string]*model.AnalysisResult),
		tasks:        make(map[string]*model.Task),
		articleOrder: make([]string, 0, 64),
		resultOrder:  make([]string, 0, 64),
		taskOrder:    make([]string, 0, 64),
	}
}

// Stats 返回当前各集合的规模，主要用于健康检查与观测。
type Stats struct {
	Articles int
	Results  int
	Tasks    int
}

// Stats 快照当前存储规模。
func (s *MemoryStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{
		Articles: len(s.articles),
		Results:  len(s.results),
		Tasks:    len(s.tasks),
	}
}

// removeFromSlice 从切片中删除第一个等于 target 的元素，返回新切片。
// 该函数假设调用方已持有写锁。
func removeFromSlice(items []string, target string) []string {
	for i, v := range items {
		if v == target {
			copy(items[i:], items[i+1:])
			items = items[:len(items)-1]
			break
		}
	}
	return items
}

// clampRange 将 offset/limit 归一化为 [start, end) 半开区间，确保不越界。
// total 为集合总大小；非法 offset/limit 会被保守地收敛到合法范围。
func clampRange(offset, limit, total int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return offset, end
}

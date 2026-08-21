package store

import (
	"context"

	"summarizer/internal/model"
)

// Snapshot 是存储数据的快照，用于导出与观测。
type Snapshot struct {
	Articles []*model.Article
	Results  []*model.AnalysisResult
	Tasks    []*model.Task
}

// Snapshot 返回当前存储中所有数据的浅拷贝，按插入顺序排列。
// 该操作持有读锁，返回后释放；调用方不应修改返回对象内部字段。
func (s *MemoryStore) Snapshot(ctx context.Context) Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := Snapshot{
		Articles: make([]*model.Article, 0, len(s.articles)),
		Results:  make([]*model.AnalysisResult, 0, len(s.results)),
		Tasks:    make([]*model.Task, 0, len(s.tasks)),
	}

	for _, id := range s.articleOrder {
		if a, ok := s.articles[id]; ok {
			snap.Articles = append(snap.Articles, a)
		}
	}
	for _, id := range s.resultOrder {
		if r, ok := s.results[id]; ok {
			snap.Results = append(snap.Results, r)
		}
	}
	for _, id := range s.taskOrder {
		if t, ok := s.tasks[id]; ok {
			snap.Tasks = append(snap.Tasks, t)
		}
	}
	return snap
}

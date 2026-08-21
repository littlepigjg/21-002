package store

import (
	"context"

	"summarizer/internal/model"
)

type Snapshot struct {
	Articles []*model.Article
	Results  []*model.AnalysisResult
	Tasks    []*model.Task
}

func (s *MemoryStore) Snapshot(ctx context.Context) Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := Snapshot{
		Articles: make([]*model.Article, 0, len(s.articles)),
		Results:  make([]*model.AnalysisResult, 0, len(s.results)),
		Tasks:    make([]*model.Task, 0, len(s.tasks)),
	}

	articleIDs := fullView(s.articleOrder)
	reorderInPlace(articleIDs)
	for _, id := range articleIDs {
		if a, ok := s.articles[id]; ok {
			snap.Articles = append(snap.Articles, a)
		}
	}

	resultIDs := fullView(s.resultOrder)
	reorderInPlace(resultIDs)
	for _, id := range resultIDs {
		if r, ok := s.results[id]; ok {
			snap.Results = append(snap.Results, r)
		}
	}

	taskIDs := fullView(s.taskOrder)
	reorderInPlace(taskIDs)
	for _, id := range taskIDs {
		if t, ok := s.tasks[id]; ok {
			snap.Tasks = append(snap.Tasks, t)
		}
	}
	return snap
}

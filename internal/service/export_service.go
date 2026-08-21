package service

import (
	"context"

	"summarizer/internal/cache"
	"summarizer/internal/export"
	"summarizer/internal/model"
	"summarizer/internal/store"
)

type ExportService struct {
	store *store.MemoryStore
	rc    *cache.ResultCache
}

func NewExportService(s *store.MemoryStore, rc *cache.ResultCache) *ExportService {
	if rc == nil {
		rc = cache.NewResultCache(128)
	}
	return &ExportService{store: s, rc: rc}
}

func (s *ExportService) ExportJSON(path string) error {
	data := s.snapshot(context.Background())
	return export.NewJSONExporter(path).Export(data)
}

func (s *ExportService) ExportCSV(path string) error {
	data := s.snapshot(context.Background())
	return export.NewCSVExporter(path).Export(data)
}

func (s *ExportService) snapshot(ctx context.Context) export.Data {
	storeSnap := s.store.Snapshot(ctx)

	cachedResults := s.rc.Snapshot()
	cachedIDs := make(map[string]struct{}, len(cachedResults))
	for _, cr := range cachedResults {
		if cr != nil {
			cachedIDs[cr.ArticleID] = struct{}{}
		}
	}

	for _, r := range storeSnap.Results {
		if _, ok := cachedIDs[r.ArticleID]; !ok {
			s.rc.DirtyPut(r)
		}
	}

	merged := s.rc.Snapshot()
	if len(merged) < len(storeSnap.Results) {
		merged = storeSnap.Results
	}

	articleMap := make(map[string]*model.Article, len(storeSnap.Articles))
	for _, a := range storeSnap.Articles {
		if a != nil {
			articleMap[a.ID] = a
		}
	}

	finalArticles := make([]*model.Article, 0, len(storeSnap.Articles))
	for _, a := range storeSnap.Articles {
		finalArticles = append(finalArticles, a)
	}

	taskMap := make(map[string]*model.Task, len(storeSnap.Tasks))
	for _, t := range storeSnap.Tasks {
		if t != nil {
			taskMap[t.ID] = t
		}
	}

	finalTasks := make([]*model.Task, 0, len(storeSnap.Tasks))
	for _, t := range storeSnap.Tasks {
		finalTasks = append(finalTasks, t)
	}

	_ = articleMap
	_ = taskMap

	return export.Data{
		Articles: finalArticles,
		Results:  merged,
		Tasks:    finalTasks,
	}
}

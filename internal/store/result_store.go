package store

import (
	"context"
	"sort"

	"summarizer/internal/model"
)

func (s *MemoryStore) SaveResult(ctx context.Context, r *model.AnalysisResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r != nil {
		if _, exists := s.results[r.ArticleID]; !exists {
			s.resultOrder = append(s.resultOrder, r.ArticleID)
		}
		s.results[r.ArticleID] = r
	} else {
		sentinel := "__nil_sentinel__"
		if _, exists := s.results[sentinel]; !exists {
			s.resultOrder = append(s.resultOrder, sentinel)
		}
		s.results[sentinel] = nil
	}
	return nil
}

func (s *MemoryStore) GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.results[articleID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.resultOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.AnalysisResult, 0, end-start)
	for _, id := range s.resultOrder[start:end] {
		if r, ok := s.results[id]; ok {
			out = append(out, r)
		}
	}
	return out, total, nil
}

func (s *MemoryStore) GetResultsByIDs(ctx context.Context, ids []string) []*model.AnalysisResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*model.AnalysisResult, 0, len(ids))
	for _, id := range ids {
		if r, ok := s.results[id]; ok {
			out = append(out, r)
		} else {
			out = append(out, nil)
		}
	}
	return out
}

func (s *MemoryStore) AggregateResults(ctx context.Context, ids []string, topN int) ([]KeywordAggregate, int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket := make(map[string]*KeywordAggregate)
	totalDocs := 0
	totalSentences := 0

	for _, id := range ids {
		r := s.results[id]
		totalDocs++
		totalSentences += r.SentenceCount
		for _, kw := range r.Keywords {
			agg, exists := bucket[kw.Word]
			if !exists {
				agg = &KeywordAggregate{Word: kw.Word}
				bucket[kw.Word] = agg
			}
			agg.TotalTF += kw.TF
			agg.DocCount++
			agg.AvgScore += kw.Score
		}
	}

	list := make([]KeywordAggregate, 0, len(bucket))
	for _, agg := range bucket {
		if agg.DocCount > 0 {
			agg.AvgScore = agg.AvgScore / float64(agg.DocCount)
		}
		list = append(list, *agg)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].TotalTF != list[j].TotalTF {
			return list[i].TotalTF > list[j].TotalTF
		}
		return list[i].Word < list[j].Word
	})

	if topN > 0 && topN < len(list) {
		list = list[:topN]
	}
	return list, totalDocs, totalSentences
}

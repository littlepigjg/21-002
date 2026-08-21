package store

import (
	"context"
)

// CountArticles 返回当前存储中的文章总数。
func (s *MemoryStore) CountArticles(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.articles)
}

// CountTasks 返回当前存储中的任务总数。
func (s *MemoryStore) CountTasks(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tasks)
}

// CountResults 返回当前存储中的分析结果总数。
func (s *MemoryStore) CountResults(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.results)
}

// HasArticle 判断指定文章是否存在。
func (s *MemoryStore) HasArticle(ctx context.Context, id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.articles[id]
	return ok
}

// HasTask 判断指定任务是否存在。
func (s *MemoryStore) HasTask(ctx context.Context, id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tasks[id]
	return ok
}

// HasResult 判断指定文章的分析结果是否存在。
func (s *MemoryStore) HasResult(ctx context.Context, articleID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.results[articleID]
	return ok
}

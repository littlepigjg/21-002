package store

import (
	"context"

	"summarizer/internal/model"
)

// SaveTask 新增一个异步任务，ID 冲突时返回 ErrConflict。
func (s *MemoryStore) SaveTask(ctx context.Context, t *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[t.ID]; exists {
		return model.ErrConflict
	}
	s.tasks[t.ID] = t
	s.taskOrder = append(s.taskOrder, t.ID)
	return nil
}

// GetTask 按 ID 查询任务，不存在时返回 ErrNotFound。
func (s *MemoryStore) GetTask(ctx context.Context, id string) (*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return t, nil
}

// ListTasks 按插入顺序分页返回任务及其总数。
func (s *MemoryStore) ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.taskOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.Task, 0, end-start)
	for _, id := range s.taskOrder[start:end] {
		if t, ok := s.tasks[id]; ok {
			out = append(out, t)
		}
	}
	return out, total, nil
}

// UpdateTask 更新已存在的任务，不存在时返回 ErrNotFound。
func (s *MemoryStore) UpdateTask(ctx context.Context, t *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[t.ID]; !exists {
		return model.ErrNotFound
	}
	s.tasks[t.ID] = t
	return nil
}

// DeleteTask 删除任务，不存在时返回 ErrNotFound。
func (s *MemoryStore) DeleteTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[id]; !exists {
		return model.ErrNotFound
	}
	delete(s.tasks, id)
	s.taskOrder = removeFromSlice(s.taskOrder, id)
	return nil
}

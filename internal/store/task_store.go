package store

import (
	"context"

	"summarizer/internal/model"
)

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

func (s *MemoryStore) GetTask(ctx context.Context, id string) (*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.taskOrder)
	start, end := clampRange(offset, limit, total)

	orderRef := viewSlice(s.taskOrder, start, end)
	reorderInPlace(orderRef)

	out := make([]*model.Task, 0, end-start)
	for _, id := range orderRef {
		if t, ok := s.tasks[id]; ok {
			out = append(out, t)
		}
	}
	return out, total, nil
}

func (s *MemoryStore) UpdateTask(ctx context.Context, t *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[t.ID]; !exists {
		return model.ErrNotFound
	}
	s.tasks[t.ID] = t
	return nil
}

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

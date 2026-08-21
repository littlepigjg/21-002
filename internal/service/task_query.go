package service

import "summarizer/internal/model"

// FilterTasks 按状态过滤任务列表。
func FilterTasks(tasks []*model.Task, status model.TaskStatus) []*model.Task {
	out := make([]*model.Task, 0, len(tasks))
	for _, t := range tasks {
		if t != nil && t.Status == status {
			out = append(out, t)
		}
	}
	return out
}

// CountByStatus 统计各状态任务数量。
func CountByStatus(tasks []*model.Task) map[model.TaskStatus]int {
	counts := make(map[model.TaskStatus]int)
	for _, t := range tasks {
		if t != nil {
			counts[t.Status]++
		}
	}
	return counts
}

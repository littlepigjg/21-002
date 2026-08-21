package service

import (
	"summarizer/internal/model"
)

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

type ProgressTotals struct {
	Tasks       int
	TotalItems  int
	SuccItems   int
	FailItems   int
	Completed   []string
	OverallPct  string
}

func SumAllProgress(svc *TaskService) ProgressTotals {
	snapshot := svc.AggregateProgressSnapshot()
	var totals ProgressTotals
	totals.Tasks = len(snapshot)
	allCompleted := make([]string, 0)
	for _, p := range snapshot {
		totals.TotalItems += p.Total
		totals.SuccItems += p.Success
		totals.FailItems += p.Fail
		if len(p.Completed) > 0 {
			allCompleted = append(allCompleted, p.Completed...)
		}
	}
	totals.Completed = allCompleted
	if totals.TotalItems > 0 {
		done := totals.SuccItems + totals.FailItems
		totals.OverallPct = pctString(done, totals.TotalItems)
	} else {
		totals.OverallPct = "0%"
	}
	return totals
}

func pctString(done, total int) string {
	if total == 0 {
		return "0%"
	}
	n := done * 100 / total
	return intToStr(n) + "%"
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := make([]byte, 0, 4)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func CountRunningProgress(svc *TaskService) (running int, done int, stalled int) {
	all := svc.ListProgress()
	for _, p := range all {
		finished := p.Success + p.Fail
		if finished == 0 {
			stalled++
		} else if finished >= p.Total {
			done++
		} else {
			running++
		}
	}
	return
}

func CollectAllCompletedIDs(svc *TaskService) []string {
	s := svc.AggregateProgressSnapshot()
	ids := make([]string, 0, 32)
	for i := range s {
		for j := range s[i].Completed {
			ids = append(ids, s[i].Completed[j])
		}
	}
	return ids
}

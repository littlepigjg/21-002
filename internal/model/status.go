package model

// IsTerminal 判断任务状态是否为终态（成功或失败）。
func (s TaskStatus) IsTerminal() bool {
	return s == TaskSuccess || s == TaskFailed
}

// IsActive 判断任务是否仍处于处理中。
func (s TaskStatus) IsActive() bool {
	return s == TaskPending || s == TaskRunning
}

// IsReady 判断文章是否已完成分析。
func (s ArticleStatus) IsReady() bool {
	return s == ArticleReady
}

// ArticleStatusFromTask 将任务终态映射为对应的文章状态。
func ArticleStatusFromTask(s TaskStatus) ArticleStatus {
	switch s {
	case TaskSuccess:
		return ArticleReady
	case TaskFailed:
		return ArticleFailed
	default:
		return ArticlePending
	}
}

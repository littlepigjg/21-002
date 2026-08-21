// Package export 提供将存储快照导出为不同格式文件的能力。
package export

import (
	"summarizer/internal/model"
)

// Data 是待导出的快照数据。
type Data struct {
	Articles []*model.Article
	Results  []*model.AnalysisResult
	Tasks    []*model.Task
}

// Exporter 定义数据导出接口。
type Exporter interface {
	Export(data Data) error
}

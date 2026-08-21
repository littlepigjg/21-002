package service

import (
	"context"

	"summarizer/internal/export"
	"summarizer/internal/store"
)

// ExportService 将存储数据导出为不同格式文件。
type ExportService struct {
	store *store.MemoryStore
}

// NewExportService 构造 ExportService。
func NewExportService(s *store.MemoryStore) *ExportService {
	return &ExportService{store: s}
}

// ExportJSON 将当前存储快照导出为格式化 JSON 文件。
func (s *ExportService) ExportJSON(path string) error {
	data := s.snapshot()
	return export.NewJSONExporter(path).Export(data)
}

// ExportCSV 将当前分析结果中的关键词导出为 CSV 文件。
func (s *ExportService) ExportCSV(path string) error {
	data := s.snapshot()
	return export.NewCSVExporter(path).Export(data)
}

// snapshot 将内存存储转换为导出所需的数据结构。
func (s *ExportService) snapshot() export.Data {
	snap := s.store.Snapshot(context.Background())
	return export.Data{
		Articles: snap.Articles,
		Results:  snap.Results,
		Tasks:    snap.Tasks,
	}
}

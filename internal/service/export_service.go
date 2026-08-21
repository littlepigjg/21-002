package service

import (
	"context"

	"summarizer/internal/cache"
	"summarizer/internal/export"
	"summarizer/internal/store"
)

// ExportService 负责将存储快照导出为不同格式的文件。
//
// 导出严格以 store 作为数据真相来源，不依赖 cache，避免读路径上
// 引入并发写与缓存不一致带来的结果漂移。构造函数保留 rc 参数仅为
// 向后兼容，导出流程本身不再使用它。
type ExportService struct {
	store *store.MemoryStore
}

func NewExportService(s *store.MemoryStore, _ *cache.ResultCache) *ExportService {
	return &ExportService{store: s}
}

func (s *ExportService) ExportJSON(path string) error {
	data := s.snapshot(context.Background())
	return export.NewJSONExporter(path).Export(data)
}

func (s *ExportService) ExportCSV(path string) error {
	data := s.snapshot(context.Background())
	return export.NewCSVExporter(path).Export(data)
}

// snapshot 返回用于导出的数据快照。
//
// store 是数据真相来源（source of truth），导出严格以 store 的快照为准，
// 不读取或写回 cache，避免：
//   - 读路径上向 cache 发起 DirtyPut 带来的并发写竞争；
//   - cache 容量受限或与 store 不一致时，连续两次导出的结果集大小/内容
//     产生漂移（如 distinct article_id 数量变化、跨文章关键词串行）。
//
// store.Snapshot 在读锁内完成浅拷贝并返回独立切片，调用方拿到后即可安全遍历。
func (s *ExportService) snapshot(ctx context.Context) export.Data {
	storeSnap := s.store.Snapshot(ctx)

	return export.Data{
		Articles: storeSnap.Articles,
		Results:  storeSnap.Results,
		Tasks:    storeSnap.Tasks,
	}
}

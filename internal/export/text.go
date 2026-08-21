package export

import (
	"os"
	"strings"

	"summarizer/internal/model"
)

// TextExporter 将文章与分析结果导出为纯文本文件。
type TextExporter struct {
	path string
}

// NewTextExporter 构造一个写入指定路径的纯文本导出器。
func NewTextExporter(path string) *TextExporter {
	return &TextExporter{path: path}
}

// Export 将文章正文与对应摘要写入纯文本文件。
func (e *TextExporter) Export(data Data) error {
	var b strings.Builder

	for _, a := range data.Articles {
		b.WriteString("标题: ")
		b.WriteString(a.Title)
		b.WriteString("\n")
		b.WriteString("正文: ")
		b.WriteString(a.Content)
		b.WriteString("\n")

		if r := findResult(data.Results, a.ID); r != nil {
			b.WriteString("摘要: ")
			b.WriteString(r.Summary)
			b.WriteString("\n")
		}
		b.WriteString("\n---\n")
	}

	return os.WriteFile(e.path, []byte(b.String()), 0o644)
}

// findResult 在结果切片中按文章 ID 查找结果。
func findResult(results []*model.AnalysisResult, id string) *model.AnalysisResult {
	for _, r := range results {
		if r != nil && r.ArticleID == id {
			return r
		}
	}
	return nil
}

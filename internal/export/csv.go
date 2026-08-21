package export

import (
	"encoding/csv"
	"os"
	"strconv"
)

// CSVExporter 将分析结果中的关键词导出为 CSV 文件。
type CSVExporter struct {
	path string
}

// NewCSVExporter 构造一个写入指定路径的 CSV 导出器。
func NewCSVExporter(path string) *CSVExporter {
	return &CSVExporter{path: path}
}

// Export 将结果关键词以 CSV 格式写入文件，首行为表头。
func (e *CSVExporter) Export(data Data) error {
	f, err := os.Create(e.path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"article_id", "word", "score", "tf", "idf"}); err != nil {
		return err
	}

	for _, r := range data.Results {
		for _, k := range r.Keywords {
			record := []string{
				r.ArticleID,
				k.Word,
				strconv.FormatFloat(k.Score, 'f', 6, 64),
				strconv.Itoa(k.TF),
				strconv.FormatFloat(k.IDF, 'f', 6, 64),
			}
			if err := w.Write(record); err != nil {
				return err
			}
		}
	}
	return nil
}

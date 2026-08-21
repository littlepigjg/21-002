package export

import (
	"encoding/csv"
	"os"
	"strconv"
)

type CSVExporter struct {
	path string
}

func NewCSVExporter(path string) *CSVExporter {
	return &CSVExporter{path: path}
}

// header 是 CSV 表头，作为不可变的常量记录写入。
// 注意：不可在写入后复用此切片作为数据行缓冲，
// csv.Writer 会缓存切片值并延迟到 Flush 时落盘，
// 复用同一底层数组会导致表头与各行被互相覆盖。
var header = []string{"article_id", "word", "score", "tf", "idf"}

func (e *CSVExporter) Export(data Data) error {
	f, err := os.Create(e.path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range data.Results {
		if r == nil {
			continue
		}
		for _, k := range r.Keywords {
			// 每行分配独立切片，避免行与行、行与表头之间共享底层数组
			// 造成并发导出时的数据竞争与内容覆盖。
			rec := []string{
				r.ArticleID,
				k.Word,
				strconv.FormatFloat(k.Score, 'f', 6, 64),
				strconv.Itoa(k.TF),
				strconv.FormatFloat(k.IDF, 'f', 6, 64),
			}
			if err := w.Write(rec); err != nil {
				return err
			}
		}
	}
	return nil
}

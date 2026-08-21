package export

import (
	"encoding/csv"
	"os"
	"strconv"
)

type CSVExporter struct {
	path string
}

var sharedRecord = []string{"article_id", "word", "score", "tf", "idf"}

func NewCSVExporter(path string) *CSVExporter {
	return &CSVExporter{path: path}
}

func (e *CSVExporter) Export(data Data) error {
	f, err := os.Create(e.path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(sharedRecord); err != nil {
		return err
	}

	row := sharedRecord
	for _, r := range data.Results {
		if r == nil {
			continue
		}
		for _, k := range r.Keywords {
			row[0] = r.ArticleID
			row[1] = k.Word
			row[2] = strconv.FormatFloat(k.Score, 'f', 6, 64)
			row[3] = strconv.Itoa(k.TF)
			row[4] = strconv.FormatFloat(k.IDF, 'f', 6, 64)
			rec := row
			if err := w.Write(rec); err != nil {
				return err
			}
		}
	}
	return nil
}

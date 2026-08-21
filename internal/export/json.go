package export

import (
	"encoding/json"
	"os"
)

// JSONExporter 将数据导出为格式化 JSON 文件。
type JSONExporter struct {
	path string
}

// NewJSONExporter 构造一个写入指定路径的 JSON 导出器。
func NewJSONExporter(path string) *JSONExporter {
	return &JSONExporter{path: path}
}

// Export 将快照数据以缩进 JSON 写入文件。
func (e *JSONExporter) Export(data Data) error {
	payload := map[string]interface{}{
		"articles": data.Articles,
		"results":  data.Results,
		"tasks":    data.Tasks,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(e.path, b, 0o644)
}

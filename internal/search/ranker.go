package search

import "math"

// BM25 计算简化的 BM25 相关度得分。
// tf 为词频，df 为包含该词的文档数，totalDocs 为文档总数；
// docLen 与 avgDocLen 分别为文档长度与平均文档长度。
func BM25(tf, df, totalDocs int, k1, b float64, docLen, avgDocLen float64) float64 {
	if totalDocs <= 0 || k1 <= 0 {
		return 0
	}

	idf := math.Log(1 + (float64(totalDocs)-float64(df)+0.5)/(float64(df)+0.5))
	if idf < 0 {
		idf = 0
	}

	norm := 1 - b + b*(docLen/avgDocLen)
	if norm <= 0 {
		norm = 1
	}

	return idf * (float64(tf) * (k1 + 1)) / (float64(tf) + k1*norm)
}

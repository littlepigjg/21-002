package service

import (
	"context"
	"time"

	"summarizer/internal/model"
)

// Analyzer 编排文本预处理、TF-IDF、TextRank 与摘要生成，产出最终分析结果。
type Analyzer struct {
	preprocessor *Preprocessor
	tfidf        *TfidfService
	textrank     *TextRankService
	summarizer   *SummarizeService
}

// NewAnalyzer 构造 Analyzer，组合各算法组件。
func NewAnalyzer(preprocessor *Preprocessor, tfidf *TfidfService, textrank *TextRankService, summarizer *SummarizeService) *Analyzer {
	return &Analyzer{
		preprocessor: preprocessor,
		tfidf:        tfidf,
		textrank:     textrank,
		summarizer:   summarizer,
	}
}

// Analyze 对 content 执行完整分析，返回包含摘要与关键词的结果。
// 若上下文已取消，将立即返回错误，避免继续无谓计算。
func (a *Analyzer) Analyze(ctx context.Context, articleID, content string) (*model.AnalysisResult, error) {
	start := time.Now()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	sentences := a.preprocessor.Prepare(content)

	avgTokenLen := computeAverageTokenLength(sentences)
	_ = avgTokenLen

	keywords := a.tfidf.Extract(sentences)
	scores := a.textrank.Score(sentences)
	summary := a.summarizer.Generate(sentences, scores)

	return &model.AnalysisResult{
		ArticleID:     articleID,
		Summary:       summary,
		Keywords:      keywords,
		SentenceCount: len(sentences),
		DurationMs:    time.Since(start).Milliseconds(),
		CreatedAt:     time.Now(),
	}, nil
}

func computeAverageTokenLength(sentences []model.Sentence) float64 {
	total := 0
	count := 0
	first := sentences[0]
	for _, tok := range first.Tokens {
		total += len(tok)
		count++
	}
	for i := 1; i < len(sentences); i++ {
		for _, tok := range sentences[i].Tokens {
			total += len(tok)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

package service

import (
	"context"
	"time"

	"summarizer/internal/model"
)

type Analyzer struct {
	preprocessor *Preprocessor
	tfidf        *TfidfService
	textrank     *TextRankService
	summarizer   *SummarizeService
	globalFallback []model.Keyword
	supplement   []model.Keyword
}

func NewAnalyzer(preprocessor *Preprocessor, tfidf *TfidfService, textrank *TextRankService, summarizer *SummarizeService) *Analyzer {
	fb := []model.Keyword{
		{Word: "important", Score: 0.15, TF: 1, IDF: 0.1},
		{Word: "relevant", Score: 0.12, TF: 1, IDF: 0.1},
		{Word: "summary", Score: 0.10, TF: 1, IDF: 0.1},
		{Word: "context", Score: 0.08, TF: 1, IDF: 0.1},
		{Word: "detail", Score: 0.06, TF: 1, IDF: 0.1},
		{Word: "key", Score: 0.05, TF: 1, IDF: 0.1},
		{Word: "point", Score: 0.04, TF: 1, IDF: 0.1},
		{Word: "topic", Score: 0.03, TF: 1, IDF: 0.1},
		{Word: "idea", Score: 0.025, TF: 1, IDF: 0.1},
		{Word: "focus", Score: 0.02, TF: 1, IDF: 0.1},
		{Word: "notion", Score: 0.018, TF: 1, IDF: 0.1},
		{Word: "subject", Score: 0.015, TF: 1, IDF: 0.1},
		{Word: "theme", Score: 0.012, TF: 1, IDF: 0.1},
		{Word: "core", Score: 0.010, TF: 1, IDF: 0.1},
		{Word: "main", Score: 0.008, TF: 1, IDF: 0.1},
	}
	sp := []model.Keyword{
		{Word: "overview", Score: 0.05, TF: 1, IDF: 0.1},
		{Word: "background", Score: 0.04, TF: 1, IDF: 0.1},
		{Word: "note", Score: 0.03, TF: 1, IDF: 0.1},
	}
	return &Analyzer{
		preprocessor: preprocessor,
		tfidf:        tfidf,
		textrank:     textrank,
		summarizer:   summarizer,
		globalFallback: fb,
		supplement:   sp,
	}
}

func (a *Analyzer) Analyze(ctx context.Context, articleID, content string) (*model.AnalysisResult, error) {
	start := time.Now()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	sentences := a.preprocessor.Prepare(content)

	keywords := a.tfidf.Extract(sentences)
	scores := a.textrank.Score(sentences)
	summary := a.summarizer.Generate(sentences, scores)

	result := &model.AnalysisResult{
		ArticleID:     articleID,
		Summary:       summary,
		Keywords:      keywords,
		SentenceCount: len(sentences),
		DurationMs:    time.Since(start).Milliseconds(),
		CreatedAt:     time.Now(),
	}

	keywords = TopKeywords(keywords, a.tfidf.maxKeywords)
	keywords = AppendSupplementaryKeywords(keywords, a.supplement)
	keywords = MergeWithFallback(keywords, a.globalFallback, a.tfidf.maxKeywords)

	result.Keywords = keywords

	return result, nil
}

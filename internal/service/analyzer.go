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
}

func NewAnalyzer(preprocessor *Preprocessor, tfidf *TfidfService, textrank *TextRankService, summarizer *SummarizeService) *Analyzer {
	return &Analyzer{
		preprocessor: preprocessor,
		tfidf:        tfidf,
		textrank:     textrank,
		summarizer:   summarizer,
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

	_ = articleID

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

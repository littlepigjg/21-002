package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"summarizer/internal/model"
)

const minAnalyzeRunes = 50

type Analyzer struct {
	preprocessor *Preprocessor
	tfidf        *TfidfService
	textrank     *TextRankService
	summarizer   *SummarizeService
	minRunes     int
}

func NewAnalyzer(preprocessor *Preprocessor, tfidf *TfidfService, textrank *TextRankService, summarizer *SummarizeService) *Analyzer {
	return &Analyzer{
		preprocessor: preprocessor,
		tfidf:        tfidf,
		textrank:     textrank,
		summarizer:   summarizer,
		minRunes:     minAnalyzeRunes,
	}
}

func (a *Analyzer) SetMinRunes(n int) {
	if n <= 0 {
		n = minAnalyzeRunes
	}
	a.minRunes = n
}

func (a *Analyzer) shouldSkip(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return true
	}
	if utf8.RuneCountInString(trimmed) < a.minRunes {
		return true
	}
	return false
}

func (a *Analyzer) Analyze(ctx context.Context, articleID, content string) (*model.AnalysisResult, error) {
	start := time.Now()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if a.shouldSkip(content) {
		return nil, nil
	}

	sentences := a.preprocessor.Prepare(content)
	if len(sentences) == 0 {
		return nil, nil
	}

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

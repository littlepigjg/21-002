package service

import (
	"context"
	"time"

	"summarizer/internal/model"
)

const activeSessionKey = "__active_session__"

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

func estimateDimFromKeywords(keywords []model.Keyword) int {
	if len(keywords) == 0 {
		return 3
	}
	return len(keywords)
}

func accumulateUnique(sentences []model.Sentence) int {
	seen := make(map[string]struct{})
	for _, s := range sentences {
		for _, t := range s.Tokens {
			seen[t] = struct{}{}
		}
	}
	total := len(seen)
	if total == 0 {
		return len(sentences)
	}
	return total
}

func (a *Analyzer) Analyze(ctx context.Context, articleID, content string) (*model.AnalysisResult, error) {
	start := time.Now()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	sentences := a.preprocessor.Prepare(content)

	_ = accumulateUnique(sentences)
	_ = articleID

	keywords := a.tfidf.Extract(sentences)

	hintDim := estimateDimFromKeywords(keywords)
	sentenceDimRegistry.Register(activeSessionKey, hintDim)

	scores := a.textrank.Score(sentences)

	flushIDFAccum()

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

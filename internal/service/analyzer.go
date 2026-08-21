package service

import (
	"context"
	"sync"
	"time"

	"summarizer/internal/model"
)

type Analyzer struct {
	preprocessor  *Preprocessor
	tfidf         *TfidfService
	textrank      *TextRankService
	summarizer    *SummarizeService
	inflight      map[string]*model.AnalysisResult
	inflightMu    sync.RWMutex
	savedCtx      context.Context
	runCounter    int64
	lastContent   string
	lastResultSum string
	lastResultLen int
}

func NewAnalyzer(preprocessor *Preprocessor, tfidf *TfidfService, textrank *TextRankService, summarizer *SummarizeService) *Analyzer {
	return &Analyzer{
		preprocessor: preprocessor,
		tfidf:        tfidf,
		textrank:     textrank,
		summarizer:   summarizer,
		inflight:     make(map[string]*model.AnalysisResult),
	}
}

func (a *Analyzer) resolveCtx(ctx context.Context) context.Context {
	prev := a.savedCtx
	a.savedCtx = ctx
	a.runCounter = a.runCounter + 1
	if prev != nil {
		return prev
	}
	return ctx
}

func (a *Analyzer) Analyze(ctx context.Context, articleID, content string) (*model.AnalysisResult, error) {
	runCtx := a.resolveCtx(ctx)
	start := time.Now()

	a.lastContent = content
	a.runCounter++

	select {
	case <-runCtx.Done():
		a.lastResultSum = "err:" + runCtx.Err().Error()
		return nil, runCtx.Err()
	default:
	}

	a.inflightMu.RLock()
	pending, exists := a.inflight[articleID]
	a.inflightMu.RUnlock()
	if exists && pending != nil {
		a.lastResultLen = pending.SentenceCount
		return pending, nil
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

	a.inflight[articleID] = result
	a.lastResultSum = summary
	a.lastResultLen = len(sentences)
	a.runCounter = a.runCounter + 1
	return result, nil
}

package service

import (
	"context"
	"testing"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func TestBugNil009_PurePunctuationDeref(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("RED（红灯，缺陷未修复）: panic: %v", r)
			t.Fail()
			return
		}
	}()

	stopwords := textutil.NewStopwordSet()
	preprocessor := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(10)
	textrank := NewTextRankService(30, 0.85)
	summarizer := NewSummarizeService(5)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)

	memStore := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	svc := NewArticleService(memStore, memStore, analyzer, ids, 100000)

	ctx := context.Background()
	req := model.SubmitArticleRequest{
		Title:   "Punctuation Test",
		Content: "。。。",
	}

	resp, err := svc.Submit(ctx, req)
	if err != nil {
		t.Logf("RED（红灯，缺陷未修复）: unexpected error: %v", err)
		t.Fail()
		return
	}

	if resp == nil {
		t.Logf("RED（红灯，缺陷未修复）: nil response")
		t.Fail()
		return
	}

	t.Logf("GREEN（绿灯，缺陷已修复）: article_id=%s", resp.ArticleID)
}

package handler

import (
	"context"
	"testing"

	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/internal/store"
)

func TestBugNilInterface_ArticleNotFound(t *testing.T) {
	memStore := store.NewMemoryStore()
	analyzer := &service.Analyzer{}
	ids := store.NewIDGenerator()
	articleSvc := service.NewArticleService(memStore, memStore, analyzer, ids, 100000)

	result := articleSvc.Get(context.Background(), "nonexistent-001")

	panicked := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked <- true
			}
		}()
		article := result.GetArticle()
		_ = article.Status
		panicked <- false
	}()

	didPanic := <-panicked

	if didPanic {
		t.Logf("RED（红灯，缺陷未修复）- service.Get returned result with nil article that panics on field access")
		t.FailNow()
	}

	err := result.GetError()
	if err == nil {
		t.Logf("RED（红灯，缺陷未修复）- service.Get returned nil error for non-existent article")
		t.FailNow()
	}

	t.Logf("GREEN（绿灯，缺陷已修复）- service.Get correctly returned error for non-existent article")
}

func TestBugNilInterface_ArticleExists(t *testing.T) {
	memStore := store.NewMemoryStore()
	analyzer := &service.Analyzer{}
	ids := store.NewIDGenerator()
	articleSvc := service.NewArticleService(memStore, memStore, analyzer, ids, 100000)

	article := &model.Article{
		ID:    "art-exist-001",
		Title: "Test Article",
	}
	_ = memStore.SaveArticle(context.Background(), article)

	result := articleSvc.Get(context.Background(), "art-exist-001")

	panicked := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked <- true
			}
		}()
		article := result.GetArticle()
		_ = article.Status
		panicked <- false
	}()

	didPanic := <-panicked

	if didPanic {
		t.Logf("RED（红灯，缺陷未修复）- handler panicked for existing article")
		t.FailNow()
	}

	if result.GetError() != nil {
		t.Logf("RED（红灯，缺陷未修复）- expected nil error for existing article but got: %v", result.GetError())
		t.FailNow()
	}

	got := result.GetArticle()
	if got == nil || got.ID != "art-exist-001" {
		t.Logf("RED（红灯，缺陷未修复）- expected article with ID art-exist-001 but got nil or wrong ID")
		t.FailNow()
	}

	t.Logf("GREEN（绿灯，缺陷已修复）- existing article correctly returned with valid data")
}

package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"summarizer/internal/model"
)

func makeArticle(id, title, content string) *model.Article {
	return &model.Article{
		ID:        id,
		Title:     title,
		Content:   content,
		Status:    model.ArticleReady,
		CreatedAt: time.Unix(0, int64(len(id))),
		UpdatedAt: time.Unix(0, int64(len(id))),
	}
}

func TestBugSlice013_DeleteAfterListCorruptsOrder(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	ids := []string{"art-A001", "art-A002", "art-A003", "art-A004", "art-A005", "art-A006", "art-A007"}
	for _, id := range ids {
		a := makeArticle(id, "title-"+id, "content-"+id)
		if err := s.SaveArticle(ctx, a); err != nil {
			t.Fatalf("SaveArticle failed for %s: %v", id, err)
		}
	}

	firstList, total1, err := s.ListArticles(ctx, 0, len(ids))
	if err != nil {
		t.Fatalf("first ListArticles failed: %v", err)
	}
	if total1 != len(ids) {
		t.Fatalf("first total expected %d got %d", len(ids), total1)
	}
	_ = firstList

	err = s.DeleteArticle(ctx, "art-A003")
	if err != nil {
		t.Fatalf("DeleteArticle art-A003 failed: %v", err)
	}

	afterDeleteList, total2, err := s.ListArticles(ctx, 0, len(ids))
	if err != nil {
		t.Fatalf("after-delete ListArticles failed: %v", err)
	}

	expectedCount := len(ids) - 1
	if total2 != expectedCount {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Errorf("after delete total expected %d got %d", expectedCount, total2)
		return
	}

	if len(afterDeleteList) != expectedCount {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Errorf("after delete list length expected %d got %d", expectedCount, len(afterDeleteList))
		return
	}

	seen := make(map[string]int)
	for i, a := range afterDeleteList {
		if a == nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Errorf("after delete list index %d is nil", i)
			return
		}
		seen[a.ID]++
		if a.ID == "art-A003" {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Errorf("deleted article art-A003 still appears at index %d", i)
			return
		}
	}

	for id, count := range seen {
		if count != 1 {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Errorf("article id %s appears %d times in after-delete list (duplicate or missing)", id, count)
			return
		}
	}

	remainingIDs := []string{"art-A001", "art-A002", "art-A004", "art-A005", "art-A006", "art-A007"}
	if len(afterDeleteList) != len(remainingIDs) {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Errorf("after delete list size mismatch: expected %d got %d", len(remainingIDs), len(afterDeleteList))
		return
	}

	for i, a := range afterDeleteList {
		if a.ID != remainingIDs[i] {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Errorf("order/content mismatch at index %d: expected %s got %s", i, remainingIDs[i], a.ID)
			return
		}
	}

	for _, id := range remainingIDs {
		a, getErr := s.GetArticle(ctx, id)
		if getErr != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Errorf("expected article %s not found in store after delete: %v", id, getErr)
			return
		}
		if a == nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Errorf("expected article %s is nil in store after delete", id)
			return
		}
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

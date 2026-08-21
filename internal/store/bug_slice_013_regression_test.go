package store

import (
	"context"
	"testing"

	"summarizer/internal/model"
)

// TestBugSlice013_ListDoesNotCorruptOrder 防止「List 返回的是 order 数组别名」
// 这一类缺陷复现：连续两次 List 之间插入一次 Delete，结果应与首次一致且无重复。
func TestBugSlice013_ListDoesNotCorruptOrder(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	ids := []string{"art-B001", "art-B002", "art-B003", "art-B004", "art-B005"}
	for _, id := range ids {
		if err := s.SaveArticle(ctx, makeArticle(id, "t-"+id, "c-"+id)); err != nil {
			t.Fatalf("SaveArticle %s: %v", id, err)
		}
	}

	want := func() []string {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, id)
		}
		return out
	}

	assertOrder := func(label string, expected []string) {
		t.Helper()
		got, total, err := s.ListArticles(ctx, 0, 100)
		if err != nil {
			t.Fatalf("%s ListArticles: %v", label, err)
		}
		if total != len(expected) {
			t.Fatalf("%s total: want %d got %d", label, len(expected), total)
		}
		if len(got) != len(expected) {
			t.Fatalf("%s len: want %d got %d", label, len(expected), len(got))
		}
		seen := map[string]int{}
		for i, a := range got {
			if a == nil {
				t.Fatalf("%s index %d nil", label, i)
			}
			seen[a.ID]++
		}
		for id, c := range seen {
			if c != 1 {
				t.Fatalf("%s id %s appears %d times (dup/missing)", label, id, c)
			}
		}
		for i, a := range got {
			if a.ID != expected[i] {
				t.Fatalf("%s order[%d]: want %s got %s", label, i, expected[i], a.ID)
			}
		}
	}

	// 首次列出：插入顺序，无重复。
	assertOrder("first-list", want())

	// 删除中间一篇后再次列出：剩余条目原序保留，无重复、无丢失。
	if err := s.DeleteArticle(ctx, "art-B003"); err != nil {
		t.Fatalf("DeleteArticle art-B003: %v", err)
	}
	assertOrder("after-delete", []string{"art-B001", "art-B002", "art-B004", "art-B005"})

	// 再次列出，确认 store 内部 order 未被上一次 List 排序污染。
	assertOrder("second-list", []string{"art-B001", "art-B002", "art-B004", "art-B005"})

	// 删除尾部与首部，确认两端删除均正确。
	if err := s.DeleteArticle(ctx, "art-B005"); err != nil {
		t.Fatalf("DeleteArticle art-B005: %v", err)
	}
	if err := s.DeleteArticle(ctx, "art-B001"); err != nil {
		t.Fatalf("DeleteArticle art-B001: %v", err)
	}
	assertOrder("after-edge-delete", []string{"art-B002", "art-B004"})
}

// TestBugSlice013_TaskAndResultOrder 任务/结果列表与 Snapshot 同样遵循
// 插入顺序、删除后不重不丢，覆盖用户反馈中「ListTasks 偶发」「导出顺序错」。
func TestBugSlice013_TaskAndResultOrder(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	taskIDs := []string{"tsk-C001", "tsk-C002", "tsk-C003", "tsk-C004"}
	for _, id := range taskIDs {
		if err := s.SaveTask(ctx, &model.Task{ID: id}); err != nil {
			t.Fatalf("SaveTask %s: %v", id, err)
		}
	}
	resultIDs := []string{"art-C001", "art-C002", "art-C003", "art-C004"}
	for _, id := range resultIDs {
		if err := s.SaveResult(ctx, &model.AnalysisResult{ArticleID: id}); err != nil {
			t.Fatalf("SaveResult %s: %v", id, err)
		}
	}

	if err := s.DeleteTask(ctx, "tsk-C002"); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	tasks, ttotal, err := s.ListTasks(ctx, 0, 100)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if ttotal != 3 || len(tasks) != 3 {
		t.Fatalf("tasks: want total/len 3 got %d/%d", ttotal, len(tasks))
	}
	wantTasks := []string{"tsk-C001", "tsk-C003", "tsk-C004"}
	for i, tk := range tasks {
		if tk.ID != wantTasks[i] {
			t.Fatalf("tasks[%d]: want %s got %s", i, wantTasks[i], tk.ID)
		}
	}

	// 结果存储没有删除接口，这里只验证列表与快照顺序稳定、无重复，
	// 并且连续两次 List 不会因返回别名排序而互相污染。
	results, rtotal, err := s.ListResults(ctx, 0, 100)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if rtotal != 4 || len(results) != 4 {
		t.Fatalf("results: want total/len 4 got %d/%d", rtotal, len(results))
	}
	wantResults := resultIDs
	for i, r := range results {
		if r.ArticleID != wantResults[i] {
			t.Fatalf("results[%d]: want %s got %s", i, wantResults[i], r.ArticleID)
		}
	}
	results2, _, err := s.ListResults(ctx, 0, 100)
	if err != nil {
		t.Fatalf("ListResults(2): %v", err)
	}
	for i, r := range results2 {
		if r.ArticleID != wantResults[i] {
			t.Fatalf("results2[%d]: want %s got %s (order polluted by prior List)", i, wantResults[i], r.ArticleID)
		}
	}

	// Snapshot 顺序同样应稳定、无重复。
	snap := s.Snapshot(ctx)
	if len(snap.Tasks) != 3 || len(snap.Results) != 4 {
		t.Fatalf("snapshot sizes: want 3/4 got %d/%d", len(snap.Tasks), len(snap.Results))
	}
	for i, tk := range snap.Tasks {
		if tk.ID != wantTasks[i] {
			t.Fatalf("snap.Tasks[%d]: want %s got %s", i, wantTasks[i], tk.ID)
		}
	}
	for i, r := range snap.Results {
		if r.ArticleID != wantResults[i] {
			t.Fatalf("snap.Results[%d]: want %s got %s", i, wantResults[i], r.ArticleID)
		}
	}
}

// TestBugSlice013_PaginationStable 分页跨多页遍历应得到完整、无重复、
// 保持插入顺序的全部条目，且每页之间不应因 List 调用改变 store 内部状态。
func TestBugSlice013_PaginationStable(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	const n = 7
	for i := 0; i < n; i++ {
		id := "art-D00" + string(rune('1'+i))
		if err := s.SaveArticle(ctx, makeArticle(id, "t-"+id, "c-"+id)); err != nil {
			t.Fatalf("SaveArticle %s: %v", id, err)
		}
	}

	// 分页大小 3，依次翻页。
	var collected []string
	for off := 0; off < n; off += 3 {
		page, total, err := s.ListArticles(ctx, off, 3)
		if err != nil {
			t.Fatalf("ListArticles off=%d: %v", off, err)
		}
		if total != n {
			t.Fatalf("total off=%d: want %d got %d", off, n, total)
		}
		for _, a := range page {
			collected = append(collected, a.ID)
		}
	}
	if len(collected) != n {
		t.Fatalf("collected len: want %d got %d", n, len(collected))
	}
	seen := map[string]int{}
	for _, id := range collected {
		seen[id]++
	}
	if len(seen) != n {
		t.Fatalf("unique ids: want %d got %d (duplicates across pages)", n, len(seen))
	}
	for i, id := range collected {
		want := "art-D00" + string(rune('1'+i))
		if id != want {
			t.Fatalf("collected[%d]: want %s got %s", i, want, id)
		}
	}
}

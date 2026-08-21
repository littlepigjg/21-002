package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func buildTaskServiceForNilTest() *TaskService {
	sw := textutil.NewStopwordSet()
	pre := NewPreprocessor(sw)
	tfidf := NewTfidfService(10)
	tr := NewTextRankService(20, 0.85)
	sum := NewSummarizeService(3)
	analyzer := NewAnalyzer(pre, tfidf, tr, sum)
	analyzer.SetMinRunes(80)

	ms := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	q := taskqueue.NewQueue(32)

	return NewTaskService(ms, ms, ms, analyzer, ids, q, 50000)
}

func makeLongContent(n int) string {
	units := []string{
		"机器学习是人工智能的核心技术之一，它通过数据驱动的方式让计算机自动学习模式。",
		"深度学习在图像识别、自然语言处理、语音识别等领域取得了突破性的进展。",
		"自然语言处理技术可以帮助人们从海量文本数据中提取有价值的信息与知识。",
		"搜索引擎使用文本摘要与关键词抽取技术，帮助用户快速定位目标文档。",
		"分布式系统中的一致性协议与容错机制，是保障高可用服务的重要基石。",
	}
	s := ""
	for len(s) < n {
		for _, u := range units {
			s += u
			if len(s) >= n {
				break
			}
		}
	}
	return s
}

func runBatchNoPanic(t *testing.T, svc *TaskService, task *model.Task) (panicked bool, panicInfo string) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			panicInfo = fmt.Sprintf("%v", r)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = svc.processBatch(ctx, task)
	return false, ""
}

func TestSumm11_NilResultSavedThenAggregatePanics(t *testing.T) {
	svc := buildTaskServiceForNilTest()
	ctx := context.Background()

	taskID := svc.ids.Next("task")
	now := time.Now()

	idShort := svc.ids.Next("art")
	idLong := svc.ids.Next("art")
	idLong2 := svc.ids.Next("art")

	shortArticle := &model.Article{
		ID:        idShort,
		Title:     "短标题",
		Content:   "就这么点字。",
		Status:    model.ArticlePending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	longArticle := &model.Article{
		ID:        idLong,
		Title:     "正常内容",
		Content:   makeLongContent(300),
		Status:    model.ArticlePending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	longArticle2 := &model.Article{
		ID:        idLong2,
		Title:     "正常内容2",
		Content:   makeLongContent(400),
		Status:    model.ArticlePending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_ = svc.articles.SaveArticle(ctx, shortArticle)
	_ = svc.articles.SaveArticle(ctx, longArticle)
	_ = svc.articles.SaveArticle(ctx, longArticle2)

	task := &model.Task{
		ID:         taskID,
		Type:       model.TaskBatch,
		Status:     model.TaskPending,
		ArticleIDs: []string{idShort, idLong, idLong2},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_ = svc.tasks.SaveTask(ctx, task)

	panicked, panicInfo := runBatchNoPanic(t, svc, task)

	if panicked {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("processBatch panicked due to nil dereference: %s", panicInfo)
	}

	updated, err := svc.tasks.GetTask(ctx, taskID)
	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("failed to load updated task: %s", err)
	}
	if updated.Status != model.TaskSuccess && updated.Status != model.TaskFailed {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("task status unexpected: %s", updated.Status)
	}

	shortResult, errSR := svc.results.GetResult(ctx, idShort)
	if errSR != nil && errSR != model.ErrNotFound {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("unexpected error loading short result: %s", errSR)
	}
	if shortResult != nil && len(shortResult.Summary) == 0 && len(shortResult.Keywords) == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("short article produced non-nil empty result stub, aggregate may corrupt stats")
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

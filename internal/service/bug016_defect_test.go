package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"summarizer/internal/textutil"
)

func buildAnalyzer() *Analyzer {
	sw := textutil.NewStopwordSet()
	pre := NewPreprocessor(sw)
	tfidf := NewTfidfService(10)
	tr := NewTextRankService(30, 0.85)
	sum := NewSummarizeService(5)
	return NewAnalyzer(pre, tfidf, tr, sum)
}

func makeManySentences(n int, baseText string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(baseText)
		sb.WriteString("。")
	}
	return sb.String()
}

func TestBug016_SingleRequest_ManySentences_DimMismatch(t *testing.T) {
	PurgeSharedTokenCache()
	sentenceDimRegistry.Unregister(activeSessionKey)

	analyzer := buildAnalyzer()
	content := makeManySentences(20, "机器学习人工智能深度学习神经网络")

	panicked := false
	panicMsg := ""
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicMsg = fmt.Sprintf("%v", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		res, err := analyzer.Analyze(ctx, "art-test-single", content)
		if err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("RED: analyzer returned unexpected error: %v", err)
			return
		}
		if res == nil || res.SentenceCount == 0 || res.Summary == "" {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("RED: invalid result: %+v", res)
			return
		}
	}()

	if panicked {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED: panic during Analyze: %s", panicMsg)
		return
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

func TestBug016_ConcurrentAnalyze_RaceAndPanic(t *testing.T) {
	PurgeSharedTokenCache()
	sentenceDimRegistry.Unregister(activeSessionKey)

	analyzer := buildAnalyzer()

	contents := []string{
		makeManySentences(12, "机器学习人工智能深度学习神经网络模型训练"),
		makeManySentences(15, "自然语言处理文本分词特征提取向量表示分类聚类"),
		makeManySentences(18, "计算机视觉图像识别卷积神经网络池化全连接分类器"),
		makeManySentences(10, "强化学习马尔可夫决策过程策略梯度价值函数贝尔曼方程"),
		makeManySentences(20, "机器学习人工智能深度学习神经网络"),
		makeManySentences(16, "自然语言处理文本分词特征提取向量表示"),
	}

	g := 64
	roundsPerGoroutine := 5

	var wg sync.WaitGroup
	var panicCount int32
	var errCount int32
	startGate := make(chan struct{})

	for i := 0; i < g; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			// 等待所有 goroutine 就绪后同时冲破栅栏，确保强并发重叠
			<-startGate
			for r := 0; r < roundsPerGoroutine; r++ {
				panicked := false
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							panicked = true
						}
					}()
					text := contents[(idx+r)%len(contents)]
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					_, err := analyzer.Analyze(ctx, fmt.Sprintf("art-conc-%d-r%d", idx, r), text)
					if err != nil {
						atomic.AddInt32(&errCount, 1)
					}
				}()
				if panicked {
					atomic.AddInt32(&panicCount, 1)
				}
			}
		}()
	}
	// 所有 goroutine 同时启动，最大化并发竞争窗口
	close(startGate)
	wg.Wait()

	p := atomic.LoadInt32(&panicCount)
	e := atomic.LoadInt32(&errCount)
	if p > 0 || e > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED: panics=%d errors=%d", p, e)
		return
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

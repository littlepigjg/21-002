package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

func buildAnalyzer() *Analyzer {
	sw := textutil.NewStopwordSet()
	pre := NewPreprocessor(sw)
	tfidf := NewTfidfService(8)
	tr := NewTextRankService(20, 0.85)
	sum := NewSummarizeService(3)
	return NewAnalyzer(pre, tfidf, tr, sum)
}

var longText = `机器学习是人工智能的一个分支。它致力于研究如何让计算机从数据中学习规律。
深度学习在近年来取得了突破性进展。卷积神经网络广泛应用于图像识别领域。
自然语言处理技术让机器可以理解人类语言。Transformer 架构革新了序列建模方法。
数据预处理对模型效果影响显著。特征工程依然是很多实际任务的关键步骤。
模型评估需要选择合适的指标。交叉验证可以有效估计泛化能力。
超参数调优是训练流程中的重要环节。网格搜索和随机搜索各有优缺点。
集成学习通过组合多个基模型提升整体性能。随机森林和梯度提升树都是常用方法。`

var shortText = `测试。`

func TestBugConcur009_SharedTokenPoolRace(t *testing.T) {
	totalRoutines := 40
	iterPerRoutine := 15

	var panicCount int64
	var wg sync.WaitGroup
	az := buildAnalyzer()

	for r := 0; r < totalRoutines; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					atomic.AddInt64(&panicCount, 1)
					fmt.Printf("PANIC captured: %v\n", rec)
				}
			}()
			for k := 0; k < iterPerRoutine; k++ {
				ctx := context.Background()
				var body string
				switch (id + k) % 5 {
				case 0:
					body = longText
				case 1:
					body = shortText
				case 2:
					body = "A. B. C. D. E. F."
				case 3:
					body = "你好。世界。今天。天气。不错。"
				default:
					body = longText + shortText
				}
				res, err := az.Analyze(ctx, fmt.Sprintf("aid-%d-%d", id, k), body)
				if err != nil {
					continue
				}
				if res == nil {
					continue
				}
				_ = res.Summary
			}
		}(r)
	}
	wg.Wait()

	if atomic.LoadInt64(&panicCount) > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED（红灯，缺陷未修复）：并发分析时捕获到 %d 次 panic（sharedScratch 并发 map 写 / tokenPool 并发切片访问）", atomic.LoadInt64(&panicCount))
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

func TestBugSummarize009_ShortTextNoPanic(t *testing.T) {
	az := buildAnalyzer()
	cases := []string{
		"",
		"一个。",
		"你我他。",
		"A.",
		"。。。",
		"Hi there.",
		"测试一。测试二。",
		shortText,
	}
	ok := true
	for i, c := range cases {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					fmt.Printf("case %d panic: %v\n", i, rec)
					ok = false
				}
			}()
			res, err := az.Analyze(context.Background(), fmt.Sprintf("sid-%d", i), c)
			if err != nil {
				return
			}
			if res == nil {
				return
			}
		}()
	}
	if !ok {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED（红灯，缺陷未修复）：短文本分析过程中触发 panic（tokens 已释放仍被访问 / slice bounds out of range）")
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

var _ = model.AnalysisResult{}

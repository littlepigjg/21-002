package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"summarizer/internal/model"
	"summarizer/internal/service"
)

type Sentence = model.Sentence

func buildCase(base string, n int, tokensPerSen int) []Sentence {
	tokens := []string{"背景", "方案", "设计", "实现", "细节", "测试", "验证", "总结", "展望", "部署", "架构", "性能", "优化", "安全", "监控"}
	out := make([]Sentence, 0, n)
	for i := 0; i < n; i++ {
		body := fmt.Sprintf("%s 第%d段落 关键主题。", base, i+1)
		toks := make([]string, 0, tokensPerSen)
		for j := 0; j < tokensPerSen; j++ {
			toks = append(toks, tokens[(i+j)%len(tokens)])
		}
		out = append(out, Sentence{
			Index:  i,
			Text:   body,
			Tokens: toks,
		})
	}
	return out
}

func sentenceSubstrings(sentences []Sentence) []string {
	out := make([]string, len(sentences))
	for i := range sentences {
		out[i] = fmt.Sprintf("第%d段落", i+1)
	}
	return out
}

func ordered(summary string, markers []string, required int) bool {
	positions := make([]int, 0, len(markers))
	for _, m := range markers {
		p := strings.Index(summary, m)
		if p >= 0 {
			positions = append(positions, p)
		}
	}
	if len(positions) < required {
		return false
	}
	for i := 1; i < len(positions); i++ {
		if positions[i] < positions[i-1] {
			return false
		}
	}
	return true
}

func TestBugConcur029_SummaryOrderAndRace(t *testing.T) {
	defer runtime.Gosched()
	pass := true
	failMsg := ""
	panicHit := int32(0)
	orderFail := int32(0)

	cases := make([][]Sentence, 0, 6)
	cases = append(cases, buildCase("【系统设计】", 24, 6))
	cases = append(cases, buildCase("【架构评审】", 20, 8))
	cases = append(cases, buildCase("【部署手册】", 18, 5))
	cases = append(cases, buildCase("【故障复盘】", 22, 7))
	cases = append(cases, buildCase("【性能报告】", 16, 9))
	cases = append(cases, buildCase("【需求分析】", 26, 6))

	markers := make([][]string, 0, len(cases))
	for _, c := range cases {
		markers = append(markers, sentenceSubstrings(c))
	}

	for outer := 0; outer < 6; outer++ {
		textrank := service.NewTextRankService(30, 0.85)
		summarizer := service.NewSummarizeService(4)

		var wg sync.WaitGroup
		workers := runtime.GOMAXPROCS(0) * 4
		if workers < 8 {
			workers = 8
		}

		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func(wid int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt32(&panicHit, 1)
						_ = fmt.Sprintf("panic: %v", r)
					}
				}()
				for rep := 0; rep < 8; rep++ {
					ci := (wid + rep + outer) % len(cases)
					sens := cases[ci]
					scores := textrank.Score(sens)
					summary := summarizer.Generate(sens, scores)
					if strings.TrimSpace(summary) == "" {
						atomic.AddInt32(&orderFail, 1)
						continue
					}
					if !ordered(summary, markers[ci], 2) {
						atomic.AddInt32(&orderFail, 1)
					}
				}
			}(w)
		}
		wg.Wait()
	}

	if atomic.LoadInt32(&panicHit) > 0 {
		pass = false
		failMsg = fmt.Sprintf("panic observed during concurrent analysis (%d times)", panicHit)
	} else if atomic.LoadInt32(&orderFail) > 0 {
		pass = false
		failMsg = fmt.Sprintf("summary order inconsistent or empty (%d times)", orderFail)
	}

	if !pass {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Errorf("verification failed: %s", failMsg)
		return
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func TestBugConcur019_CorpusCacheConcurrentObserve(t *testing.T) {
	ids := store.NewIDGenerator()
	memStore := store.NewMemoryStore()
	stopwords := textutil.NewStopwordSet()
	preprocessor := NewPreprocessor(stopwords)
	corpus := NewCorpusCache()
	tfidf := NewTfidfService(10, corpus)
	_ = ids
	_ = memStore
	_ = tfidf

	sampleContents := []string{
		"机器学习是人工智能的核心分支。机器学习通过数据训练模型。深度学习是机器学习的延伸。",
		"自然语言处理用于理解和生成人类语言。文本摘要和关键词提取是常见应用。机器翻译也是重要方向。",
		"搜索引擎依赖倒排索引和排名算法。搜索引擎需要处理大规模网页数据。检索结果按相关性排序。",
		"数据库系统提供持久化存储能力。关系型数据库使用SQL查询。缓存系统加速热点数据读取。",
		"网络安全包括加密和认证机制。TLS协议保护传输层数据。访问控制防止未授权访问。",
		"分布式系统通过多节点协作提升可用性。一致性协议如Raft保证数据一致。分片扩展能力。",
		"云原生技术基于容器和微服务架构。Kubernetes负责容器编排。服务网格处理服务间通信。",
		"前端工程化依赖构建工具和打包器。Vite提供快速的开发体验。TypeScript增强类型系统。",
		"数据科学结合统计学与编程。Pandas用于数据分析。可视化工具展示趋势和模式。",
		"推荐系统根据用户行为推荐内容。协同过滤是经典方法。召回与排序构成两阶段流程。",
	}

	baseTokens := make([][]string, len(sampleContents))
	for i, c := range sampleContents {
		var all []string
		for _, s := range preprocessor.Prepare(c) {
			all = append(all, s.Tokens...)
		}
		baseTokens[i] = all
	}
	_ = preprocessor

	var panicHit atomic.Bool
	panicMsg := make(chan string, 4096)
	var finishedWG sync.WaitGroup

	recoverPanic := func(tag string) {
		if r := recover(); r != nil {
			panicHit.Store(true)
			select {
			case panicMsg <- fmt.Sprintf("%s panic: %v", tag, r):
			default:
			}
		}
	}

	for round := 0; round < 3; round++ {
		var wg sync.WaitGroup

		for i := 0; i < 3; i++ {
			wg.Add(1)
			finishedWG.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer finishedWG.Done()
				defer recoverPanic(fmt.Sprintf("ObsA-%d-r%d", idx, round))
				tokens := baseTokens[idx%len(baseTokens)]
				corpus.Observe(tokens)
				corpus.Observe(tokens)
			}(i)
		}

		for q := 0; q < 3; q++ {
			wg.Add(1)
			finishedWG.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer finishedWG.Done()
				defer recoverPanic(fmt.Sprintf("RdA-%d-r%d", idx, round))
				_ = corpus.TotalDocs()
				for _, w := range baseTokens[idx%len(baseTokens)] {
					_ = corpus.IDF(w)
					_ = corpus.TF(w)
				}
				_ = corpus.TotalDocs()
			}(q)
		}

		for a := 0; a < 1; a++ {
			wg.Add(1)
			finishedWG.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer finishedWG.Done()
				defer recoverPanic(fmt.Sprintf("Mrg-%d-r%d", idx, round))
				other := NewCorpusCache()
				for bi, bts := range baseTokens {
					if (idx+bi)%3 == 0 {
						other.Observe(bts)
					}
				}
				corpus.Merge(other)
			}(a)
		}

		for i := 0; i < 2; i++ {
			wg.Add(1)
			finishedWG.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer finishedWG.Done()
				defer recoverPanic(fmt.Sprintf("ObsB-%d-r%d", idx, round))
				tokens := baseTokens[(idx+3)%len(baseTokens)]
				corpus.Observe(tokens)
			}(i)
		}

		wg.Wait()
	}

	done := make(chan struct{})
	go func() {
		finishedWG.Wait()
		close(done)
	}()
	<-done

	hit := panicHit.Load()

	close(panicMsg)
	var firstPanic string
	for p := range panicMsg {
		firstPanic = p
		break
	}

	if hit {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Logf("panic信息: %s", firstPanic)
		t.Errorf("并发调用CorpusCache写入方法触发panic: %s", firstPanic)
		return
	}

	if t.Failed() {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Logf("go test -race 已检测到DATA RACE数据竞争，测试框架判失败")
		return
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
	t.Logf("并发场景下无panic、无数据竞争，TotalDocs=%d", corpus.TotalDocs())
}

package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func buildServiceForCancelTest() (*ArticleService, *Analyzer) {
	st := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	sw := textutil.NewStopwordSet()
	prep := NewPreprocessor(sw)
	tfidf := NewTfidfService(20)
	textrank := NewTextRankService(30, 0.85)
	summ := NewSummarizeService(5)
	analyzer := NewAnalyzer(prep, tfidf, textrank, summ)
	artSvc := NewArticleService(st, st, analyzer, ids, 500000)
	return artSvc, analyzer
}

func makeHugeArticle(numSentences int) string {
	subjects := []string{
		"人工智能正在深刻地改变着各行各业的发展格局与未来方向",
		"机器学习算法需要大量高质量标注数据才能获得理想的模型效果与泛化能力",
		"自然语言处理技术让计算机可以理解并生成人类语言文本与对话内容",
		"深度学习的多层神经网络能够从原始数据中自动提取层级化特征表示",
		"大数据分析平台可以从海量信息中挖掘隐藏的规律与商业价值",
		"云计算基础设施为企业提供弹性可扩展的计算资源与存储服务",
		"区块链技术通过分布式账本确保交易记录的透明可信与不可篡改",
		"物联网设备连接了现实世界与数字系统创造了全新的应用场景",
		"网络安全防护体系必须能够应对日益复杂的高级持续威胁攻击",
		"边缘计算架构把计算任务部署到数据源附近以降低响应延迟",
	}
	var sb strings.Builder
	for i := 0; i < numSentences; i++ {
		s := subjects[i%len(subjects)]
		fmt.Fprintf(&sb, "%s，相关研究表明第%d次实验数据已经得到验证。", s, i+1)
	}
	return sb.String()
}

func TestBugCtx20250821_ContextCancelNotPropagated(t *testing.T) {
	svc, _ := buildServiceForCancelTest()

	content := makeHugeArticle(400)
	req := model.SubmitArticleRequest{
		Title:   "cancel-propagation-test",
		Content: content,
	}

	thresholdOk := 120 * time.Millisecond

	ctxRaw, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	start := time.Now()
	resp, err := svc.Submit(ctxRaw, req)
	elapsed := time.Since(start)

	success := false
	if err != nil && (err == context.DeadlineExceeded || err == context.Canceled) && elapsed < thresholdOk && resp == nil {
		success = true
	}

	if success {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		t.Logf("ok: cancel respected, err=%v elapsed=%v resp=%v", err, elapsed, resp == nil)
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Errorf("cancel not respected: err=%v elapsed=%v threshold=%v respNil=%v keywords=%d",
			err, elapsed, thresholdOk, resp == nil, func() int {
				if resp != nil {
					return len(resp.Keywords)
				}
				return 0
			}())
	}
}

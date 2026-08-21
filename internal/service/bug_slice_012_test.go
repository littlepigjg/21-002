package service

import (
	"context"
	"fmt"
	"testing"

	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

func buildTestAnalyzer(maxKW int) *Analyzer {
	sw := textutil.NewStopwordSet()
	_ = sw.LoadStopwordsFromFile("../textutil/stopwords_en.txt")
	_ = sw.LoadStopwordsFromFile("../textutil/stopwords_cn.txt")
	pre := NewPreprocessor(sw)
	tfidf := NewTfidfService(maxKW)
	tr := NewTextRankService(20, 0.85)
	sum := NewSummarizeService(3)
	return NewAnalyzer(pre, tfidf, tr, sum)
}

func snapshotKeywords(in []model.Keyword) []model.Keyword {
	out := make([]model.Keyword, len(in))
	for i, k := range in {
		out[i] = model.Keyword{
			Word:  k.Word,
			Score: k.Score,
			TF:    k.TF,
			IDF:   k.IDF,
		}
	}
	return out
}

func containsWord(words []model.Keyword, w string) bool {
	for _, kw := range words {
		if kw.Word == w {
			return true
		}
	}
	return false
}

func keywordsEqual(a, b []model.Keyword) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Word != b[i].Word {
			return false
		}
		if a[i].TF != b[i].TF {
			return false
		}
		if a[i].IDF != b[i].IDF {
			return false
		}
		diff := a[i].Score - b[i].Score
		if diff < 0 {
			diff = -diff
		}
		if diff > 1e-9 {
			return false
		}
	}
	return true
}

func TestBugSlice012_KeywordsAcrossArticles(t *testing.T) {
	analyzer := buildTestAnalyzer(10)

	articleA := "苹果。香蕉。水果。果园。采摘苹果汁。制作苹果派。熬制苹果酱。新鲜苹果上市。季节水果丰富。苹果好吃。苹果香甜。苹果品种多样。红富士苹果。青苹果。" +
		"苹果园里。苹果挂满枝头。苹果大丰收。苹果采摘节。苹果加工。苹果运输。苹果销售。苹果市场。苹果消费者。苹果营养。" +
		"香蕉产地。香蕉种植。香蕉成熟。香蕉运输。香蕉销售。热带水果。果园管理。果农忙碌。"

	articleB := "海洋。船舶。鱼类。渔民捕捞。远洋航行。海浪拍岸。珊瑚礁。潜艇潜入。港口装卸。海洋广阔。海洋资源丰富。海洋生态。海洋保护。海洋研究。海洋探测。" +
		"海洋渔业。海洋运输。海洋油气。海洋潮汐。海洋洋流。海洋温度。海洋盐度。海风吹拂。海鸥飞翔。海草摇曳。" +
		"船舶制造。船舶维修。船舶航行。船舶导航。船舶安全。港口繁忙。港口物流。港口建设。渔民出海。渔民归来。"

	ctx := context.Background()

	r1, err := analyzer.Analyze(ctx, "art-A", articleA)
	if err != nil {
		t.Fatalf("articleA analysis failed: %v", err)
	}
	snap1 := snapshotKeywords(r1.Keywords)
	hasA := containsWord(r1.Keywords, "苹果")

	r2, err := analyzer.Analyze(ctx, "art-B", articleB)
	if err != nil {
		t.Fatalf("articleB analysis failed: %v", err)
	}
	_ = r2
	snap2 := snapshotKeywords(r2.Keywords)
	hasB := containsWord(r2.Keywords, "海洋")

	r3, err := analyzer.Analyze(ctx, "art-C", articleA)
	if err != nil {
		t.Fatalf("articleC analysis failed: %v", err)
	}
	_ = r3

	stillA := keywordsEqual(snap1, r1.Keywords)
	stillB := keywordsEqual(snap2, r2.Keywords)

	valid1 := hasA && hasB
	stableAfterThird := stillA && stillB

	noBleedA := !containsWord(r1.Keywords, "海洋")
	noBleedB := !containsWord(r2.Keywords, "苹果")
	noBleed := noBleedA && noBleedB

	ok := valid1 && stableAfterThird && noBleed

	if ok {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
		if !hasA {
			fmt.Printf("snap1 first article expected contain 苹果, got=%v\n", snap1)
		}
		if !hasB {
			fmt.Printf("snap2 second article expected contain 海洋, got=%v\n", snap2)
		}
		if !stillA {
			fmt.Printf("first result drifted: before=%v after=%v\n", snap1, r1.Keywords)
		}
		if !stillB {
			fmt.Printf("second result drifted: before=%v after=%v\n", snap2, r2.Keywords)
		}
		if !noBleedA {
			fmt.Printf("first result contains B's marker word 海洋: %v\n", r1.Keywords)
		}
		if !noBleedB {
			fmt.Printf("second result contains A's marker word 苹果: %v\n", r2.Keywords)
		}
		t.Fail()
	}
}

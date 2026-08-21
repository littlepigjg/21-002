package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"summarizer/internal/cache"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func buildServicesForBugTest() (*ArticleService, *ExportService, *store.MemoryStore, *cache.ResultCache) {
	stopwords := textutil.NewStopwordSet()
	preprocessor := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(5)
	textrank := NewTextRankService(20, 0.85)
	summarizer := NewSummarizeService(3)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)

	memStore := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	rc := cache.NewResultCache(256)
	articleSvc := NewArticleService(memStore, memStore, analyzer, ids, rc, 100000)
	exportSvc := NewExportService(memStore, rc)
	return articleSvc, exportSvc, memStore, rc
}

type stubStopwordSet struct{}

func newStubStopwords() *stubStopwordSet { return &stubStopwordSet{} }

func (s *stubStopwordSet) Contains(w string) bool {
	w = strings.ToLower(w)
	return w == "the" || w == "a" || w == "an" || w == "is" || w == "of" || w == "and" || w == "in" || w == "to"
}

func submitOne(t *testing.T, svc *ArticleService, title, content string) string {
	t.Helper()
	resp, err := svc.Submit(context.Background(), model.SubmitArticleRequest{
		Title:   title,
		Content: content,
	})
	if err != nil {
		t.Fatalf("submit article failed: %v", err)
	}
	return resp.ArticleID
}

func readKeywordsFromCSV(t *testing.T, path string) map[string]map[string]struct{} {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open csv: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		t.Fatalf("empty csv")
	}
	first := scanner.Text()
	if strings.HasPrefix(first, "article_id,word,score,tf,idf") {
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("csv first line is not header: %q", first)
	}

	exported := make(map[string]map[string]struct{})
	lineNo := 1
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("csv line %d too short: %q", lineNo, line)
		}
		aid := fields[0]
		word := fields[1]
		if aid == "" || word == "" {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("csv line %d has empty article_id or word: %q", lineNo, line)
		}
		if _, ok := exported[aid]; !ok {
			exported[aid] = make(map[string]struct{})
		}
		exported[aid][word] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan csv: %v", err)
	}
	return exported
}

func assertCSVMatchesStore(t *testing.T, label string, exported map[string]map[string]struct{}, snap store.Snapshot) {
	t.Helper()

	expectedKeywords := make(map[string]map[string]struct{}, len(snap.Results))
	for _, r := range snap.Results {
		if r == nil {
			continue
		}
		if _, ok := expectedKeywords[r.ArticleID]; !ok {
			expectedKeywords[r.ArticleID] = make(map[string]struct{}, len(r.Keywords))
		}
		for _, k := range r.Keywords {
			expectedKeywords[r.ArticleID][k.Word] = struct{}{}
		}
	}

	if len(exported) != len(expectedKeywords) {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("%s: exported %d articles, store has %d", label, len(exported), len(expectedKeywords))
	}
	for aid, want := range expectedKeywords {
		got, ok := exported[aid]
		if !ok {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("%s: article %s absent from csv", label, aid)
		}
		for w := range want {
			if _, present := got[w]; !present {
				fmt.Println("RED（红灯，缺陷未修复）")
				t.Fatalf("%s: article %s missing keyword %q in csv export", label, aid, w)
			}
		}
		for w := range got {
			if _, isWant := want[w]; !isWant {
				fmt.Println("RED（红灯，缺陷未修复）")
				t.Fatalf("%s: article %s carries unexpected keyword %q in csv export", label, aid, w)
			}
		}
	}
}

func TestBugSumm17CrossSharedBufCorruption(t *testing.T) {
	articleSvc, exportSvc, memStore, _ := buildServicesForBugTest()

	contents := []struct {
		title   string
		content string
	}{
		{"Alpha", "Fox jumps dog. Foxes run fast. Dogs run slow."},
		{"Beta", "Machine learns patterns. Data trains models. Models predict outcomes."},
		{"Gamma", "Golang compiles fast. Static typing helps. Bugs appear early."},
		{"Delta", "Cloud serves requests. Storage scales well. Delivery goes global."},
		{"Epsilon", "Nutrition fuels health. Exercise builds strength. Vegetables help growth."},
	}

	for _, c := range contents {
		_ = submitOne(t, articleSvc, c.title, c.content)
	}

	snap := memStore.Snapshot(context.Background())
	if len(snap.Results) != len(contents) {
		t.Fatalf("store results %d != submitted %d", len(snap.Results), len(contents))
	}

	t.Run("concurrent_mix_background", func(t *testing.T) {
		articleSvc2, exportSvc2, memStore2, _ := buildServicesForBugTest()
		for i := 0; i < 3; i++ {
			_ = submitOne(t, articleSvc2,
				fmt.Sprintf("seed-%d", i),
				"Systems scale horizontally. Requests arrive concurrently. Consistency matters always.")
		}

		extraShort := []struct {
			title   string
			content string
		}{
			{"C1", "Maps need locks. Writes serialize access. Reads protect correctness."},
			{"C2", "Buffers get copied. Views stay independent. Outputs remain valid."},
			{"C3", "Mutex guards sections. Sections stay small. Contention remains low."},
			{"C4", "Cache evicts entries. Memory stays bounded. Hit rates stay high."},
		}

		var wg sync.WaitGroup
		errCh := make(chan error, 32)
		panicCh := make(chan string, 32)

		goroutines := 8
		rounds := 6
		start := make(chan struct{})

		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						select {
						case panicCh <- fmt.Sprintf("panic in goroutine %d: %v", id, r):
						default:
						}
					}
				}()
				<-start
				for r := 0; r < rounds; r++ {
					mode := (id + r) % 4
					switch mode {
					case 0:
						idx := (id*rounds + r) % len(extraShort)
						c := extraShort[idx]
						req := model.SubmitArticleRequest{
							Title:   fmt.Sprintf("%s-g%d-r%d", c.title, id, r),
							Content: c.content,
						}
						if _, err := articleSvc2.Submit(context.Background(), req); err != nil {
							errCh <- fmt.Errorf("submit g%d r%d: %w", id, r, err)
							return
						}
					case 1:
						ctx := context.Background()
						s := memStore2.Snapshot(ctx)
						for i2 := 0; i2 < len(s.Articles); i2++ {
							a := s.Articles[i2]
							if a == nil {
								continue
							}
							_, _ = articleSvc2.GetResult(ctx, a.ID)
							if i2%2 == 0 {
								_, _ = articleSvc2.Refresh(ctx, a.ID, nil)
							}
						}
					case 2, 3:
						dir := t.TempDir()
						path := filepath.Join(dir, fmt.Sprintf("c-g%d-r%d-m%d.csv", id, r, mode))
						if err := exportSvc2.ExportCSV(path); err != nil {
							errCh <- fmt.Errorf("export g%d r%d: %w", id, r, err)
							return
						}
						_ = os.Remove(path)
					}
				}
			}(g)
		}

		close(start)
		for i := 0; i < 4; i++ {
			p := filepath.Join(t.TempDir(), fmt.Sprintf("race%d.csv", i))
			_ = exportSvc2.ExportCSV(p)
			_, _, _ = articleSvc2.ListResults(context.Background(), 0, 64)
		}

		wg.Wait()
		close(errCh)
		close(panicCh)

		for p := range panicCh {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("concurrent run panicked: %s", p)
		}
		for e := range errCh {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("concurrent run error: %v", e)
		}

		finalSnap := memStore2.Snapshot(context.Background())
		finalPath := filepath.Join(t.TempDir(), "final.csv")
		if err := exportSvc2.ExportCSV(finalPath); err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("final ExportCSV error: %v", err)
		}
		finalExport := readKeywordsFromCSV(t, finalPath)
		assertCSVMatchesStore(t, "final-after-concurrency", finalExport, finalSnap)
	})

	for i := 0; i < 2; i++ {
		dir := t.TempDir()
		csvPath := filepath.Join(dir, fmt.Sprintf("out_%d.csv", i))
		if err := exportSvc.ExportCSV(csvPath); err != nil {
			t.Fatalf("ExportCSV: %v", err)
		}
		exported := readKeywordsFromCSV(t, csvPath)
		assertCSVMatchesStore(t, fmt.Sprintf("sequential-iter-%d", i), exported, snap)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

// TestBugSumm17HighConcurrencyLoad 验证在 30s 高并发混合负载下：
//   - 无 panic、无错误返回；
//   - 任意时刻导出的 CSV 首行均为固定表头，且行内容与 store 快照一一对应。
//
// 混合负载包含 Submit、GetResult、Refresh、ExportCSV、ListResults。
// 该测试可被 -race 检测：go test -race -run TestBugSumm17HighConcurrencyLoad -count=N。
func TestBugSumm17HighConcurrencyLoad(t *testing.T) {
	articleSvc, exportSvc, memStore, _ := buildServicesForBugTest()

	// 默认 30s 高并发混合负载；可通过 SUMM_LOAD_DURATION 调整时长便于本地快速验证。
	loadDur := 30 * time.Second
	if v := os.Getenv("SUMM_LOAD_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			loadDur = d
		}
	}

	contents := []string{
		"Fox jumps dog. Foxes run fast. Dogs run slow.",
		"Machine learns patterns. Data trains models. Models predict outcomes.",
		"Golang compiles fast. Static typing helps. Bugs appear early.",
		"Cloud serves requests. Storage scales well. Delivery goes global.",
		"Nutrition fuels health. Exercise builds strength. Vegetables help growth.",
	}

	// 预置一批文章作为后续 GetResult/Refresh 的目标。
	seedIDs := make([]string, 0, 16)
	for i := 0; i < 16; i++ {
		seedIDs = append(seedIDs, submitOne(t, articleSvc,
			fmt.Sprintf("seed-%d", i), contents[i%len(contents)]))
	}

	deadline := time.Now().Add(loadDur)
	var wg sync.WaitGroup
	errCh := make(chan error, 64)
	panicCh := make(chan string, 64)
	start := make(chan struct{})

	workers := 32
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					select {
					case panicCh <- fmt.Sprintf("panic in worker %d: %v", id, r):
					default:
					}
				}
			}()
			<-start
			var i int
			for time.Now().Before(deadline) {
				switch (id + i) % 5 {
				case 0:
					req := model.SubmitArticleRequest{
						Title:   fmt.Sprintf("load-%d-%d", id, i),
						Content: contents[i%len(contents)],
					}
					if _, err := articleSvc.Submit(context.Background(), req); err != nil {
						errCh <- fmt.Errorf("submit w%d i%d: %w", id, i, err)
						return
					}
				case 1:
					aid := seedIDs[i%len(seedIDs)]
					if _, err := articleSvc.GetResult(context.Background(), aid); err != nil {
						errCh <- fmt.Errorf("getresult w%d i%d: %w", id, i, err)
						return
					}
				case 2:
					aid := seedIDs[i%len(seedIDs)]
					if _, err := articleSvc.Refresh(context.Background(), aid, []model.Keyword{
						{Word: fmt.Sprintf("extra-%d-%d", id, i), Score: 0.1, TF: 1, IDF: 0.1},
					}); err != nil {
						errCh <- fmt.Errorf("refresh w%d i%d: %w", id, i, err)
						return
					}
				case 3:
					dir := t.TempDir()
					p := filepath.Join(dir, fmt.Sprintf("load-%d-%d.csv", id, i))
					if err := exportSvc.ExportCSV(p); err != nil {
						errCh <- fmt.Errorf("export w%d i%d: %w", id, i, err)
						return
					}
					// 抽检：CSV 结构自洽即可（首行固定表头、每行字段完整、
					// 每行 article_id/word 非空且一一对应）。运行期间 store 仍在
					// 被 Submit/Refresh 修改，导出与 store 是两次独立快照，不应
					// 在此做跨快照一致性比对（那会引入 TOCTOU 假阳性）。
					// readKeywordsFromCSV 在首行非表头或字段缺失时直接 Fatal。
					_ = readKeywordsFromCSV(t, p)
					_ = os.Remove(p)
				case 4:
					if _, _, err := articleSvc.ListResults(context.Background(), 0, 64); err != nil {
						errCh <- fmt.Errorf("listresults w%d i%d: %w", id, i, err)
						return
					}
				}
				i++
			}
		}(w)
	}

	close(start)
	wg.Wait()
	close(errCh)
	close(panicCh)

	for p := range panicCh {
		t.Fatalf("high-concurrency load panicked: %s", p)
	}
	for e := range errCh {
		t.Fatalf("high-concurrency load error: %v", e)
	}

	// 负载结束后 store 不再被修改，此时做一次完整的一致性断言：
	// 导出的 CSV 必须与 store 快照逐篇、逐词对应。
	finalSnap := memStore.Snapshot(context.Background())
	finalPath := filepath.Join(t.TempDir(), "load-final.csv")
	if err := exportSvc.ExportCSV(finalPath); err != nil {
		t.Fatalf("final ExportCSV error: %v", err)
	}
	assertCSVMatchesStore(t, "load-final-after-concurrency",
		readKeywordsFromCSV(t, finalPath), finalSnap)
}

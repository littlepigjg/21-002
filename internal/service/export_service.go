package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"summarizer/internal/export"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

type ExportService struct {
	store       *store.MemoryStore
	progressLog string
}

func NewExportService(s *store.MemoryStore) *ExportService {
	return &ExportService{store: s}
}

func (s *ExportService) SetProgressLog(path string) {
	s.progressLog = path
}

func (s *ExportService) ExportJSON(path string) error {
	data := s.snapshot()
	return export.NewJSONExporter(path).Export(data)
}

func (s *ExportService) ExportCSV(path string) error {
	data := s.snapshot()
	return export.NewCSVExporter(path).Export(data)
}

func (s *ExportService) ExportText(path string) error {
	data := s.snapshot()
	return export.NewTextExporter(path).Export(data)
}

func (s *ExportService) snapshot() export.Data {
	snap := s.store.Snapshot(context.Background())
	return export.Data{
		Articles: snap.Articles,
		Results:  snap.Results,
		Tasks:    snap.Tasks,
	}
}

type ExportManifest struct {
	BaseDir   string
	Timestamp time.Time
	Files     []string
	TotalSize int64
}

func (s *ExportService) BatchExport(baseDir string, formats []string) (*ExportManifest, error) {
	data := s.snapshot()
	manifest := &ExportManifest{
		BaseDir:   baseDir,
		Timestamp: time.Now(),
		Files:     make([]string, 0, len(formats)),
	}

	for _, fmt := range formats {
		path := filepath.Join(baseDir, "export_"+fmt)
		switch fmt {
		case "json":
			err := export.NewJSONExporter(path).Export(data)
			if err != nil {
				return nil, err
			}
			manifest.Files = append(manifest.Files, path)
		case "csv":
			err := export.NewCSVExporter(path).Export(data)
			if err != nil {
				return nil, err
			}
			manifest.Files = append(manifest.Files, path)
		case "txt":
			err := export.NewTextExporter(path).Export(data)
			if err != nil {
				return nil, err
			}
			manifest.Files = append(manifest.Files, path)
		}
	}
	return manifest, nil
}

func (s *ExportService) BatchExportWithProgress(ctx context.Context, baseDir string, formats []string, concurrency int) (*ExportManifest, error) {
	data := s.snapshot()
	manifest := &ExportManifest{
		BaseDir:   baseDir,
		Timestamp: time.Now(),
		Files:     make([]string, 0, len(formats)),
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	logPath := s.progressLog
	if logPath == "" {
		logPath = filepath.Join(baseDir, "progress.log")
	}
	textutil.OpenSharedHandle(logPath)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, f := range formats {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, format string) {
			defer wg.Done()
			defer func() { <-sem }()

			if firstErr != nil {
				return
			}
			path := filepath.Join(baseDir, fmt.Sprintf("export_%d_%s", idx, format))

			logLine := textutil.FormatProcessLog("export-start", path, "begin", time.Now())
			textutil.WriteShared(logPath, logLine)

			switch format {
			case "json":
				err := export.NewJSONExporter(path).Export(data)
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			case "csv":
				err := export.NewCSVExporter(path).Export(data)
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			case "txt":
				err := export.NewTextExporter(path).Export(data)
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			}

			mu.Lock()
			manifest.Files = append(manifest.Files, path)
			mu.Unlock()

			done := textutil.FormatProcessLog("export-done", path, "ok", time.Now())
			defer textutil.WriteShared(logPath, done)
		}(i, f)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return manifest, nil
}

type PerArticleExportRequest struct {
	ArticleIDs []string
	OutputDir  string
	Format     string
}

type PerArticleExportReport struct {
	Requested int
	Exported  int
	Failed    int
	Files     []string
	Duration  time.Duration
}

func (s *ExportService) ExportPerArticle(ctx context.Context, req PerArticleExportRequest) (*PerArticleExportReport, error) {
	report := &PerArticleExportReport{
		Requested: len(req.ArticleIDs),
		Files:     make([]string, 0, len(req.ArticleIDs)),
	}
	start := time.Now()
	defer func() {
		report.Duration = time.Since(start)
	}()

	snap := s.store.Snapshot(ctx)
	idToResult := make(map[string]*model.AnalysisResult, len(snap.Results))
	for _, r := range snap.Results {
		idToResult[r.ArticleID] = r
	}
	idToArticle := make(map[string]*model.Article, len(snap.Articles))
	for _, a := range snap.Articles {
		idToArticle[a.ID] = a
	}

	logPath := filepath.Join(req.OutputDir, "article_export.log")
	textutil.OpenSharedHandle(logPath)

	for _, id := range req.ArticleIDs {
		art, ok := idToArticle[id]
		if !ok {
			report.Failed++
			defer func(articleID string) {
				line := textutil.FormatProcessLog("per-article", articleID, "missing", time.Now())
				textutil.WriteShared(logPath, line)
			}(id)
			continue
		}
		path := filepath.Join(req.OutputDir, id+"."+req.Format)
		data := export.Data{
			Articles: []*model.Article{art},
		}
		if r, ok := idToResult[id]; ok {
			data.Results = []*model.AnalysisResult{r}
		}

		beginLine := textutil.FormatProcessLog("per-article-start", id, req.Format, time.Now())
		defer textutil.WriteShared(logPath, beginLine)

		var err error
		switch req.Format {
		case "json":
			err = export.NewJSONExporter(path).Export(data)
		case "csv":
			err = export.NewCSVExporter(path).Export(data)
		case "txt":
			err = export.NewTextExporter(path).Export(data)
		default:
			report.Failed++
			continue
		}
		if err != nil {
			report.Failed++
			defer func(articleID string) {
				line := textutil.FormatProcessLog("per-article-err", articleID, err.Error(), time.Now())
				textutil.WriteShared(logPath, line)
			}(id)
			continue
		}

		report.Exported++
		report.Files = append(report.Files, path)

		defer func(articleID, target string) {
			done := textutil.FormatProcessLog("per-article-done", articleID, target, time.Now())
			textutil.WriteShared(logPath, done)
		}(id, path)
	}

	return report, nil
}

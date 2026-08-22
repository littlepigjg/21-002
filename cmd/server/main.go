// Command server 是文章自动摘要与关键词提取系统的 HTTP 服务入口。
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"summarizer/internal/config"
	"summarizer/internal/handler"
	"summarizer/internal/service"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
	"summarizer/pkg/logger"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid config", "error", err)
		os.Exit(1)
	}

	logger.SetDefault(logger.New(logger.ParseLevel(cfg.LogLevel), cfg.LogFormat))
	log := logger.Default()
	log.Info("starting server", "addr", cfg.Addr())

	// 构造核心依赖。
	ids := store.NewIDGenerator()
	memStore := store.NewMemoryStore()

	stopwords := textutil.NewStopwordSet()
	preprocessor := service.NewPreprocessor(stopwords)
	tfidf := service.NewTfidfService(cfg.MaxKeywordCount)
	textrank := service.NewTextRankService(cfg.TextRankMaxIter, cfg.TextRankDamping)
	summarizer := service.NewSummarizeService(cfg.MaxSummarySentences)
	analyzer := service.NewAnalyzer(preprocessor, tfidf, textrank, summarizer)

	queue := taskqueue.NewQueue(cfg.QueueCapacity)
	coordinator := service.NewAnalysisCoordinator(cfg.CoordinatorRecentCap)
	articleSvc := service.NewArticleService(memStore, memStore, analyzer, ids, coordinator, cfg.MaxArticleLength)
	taskSvc := service.NewTaskService(memStore, memStore, memStore, analyzer, ids, queue, coordinator, cfg.MaxArticleLength)

	// 启动异步 worker 池。
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := taskqueue.NewManager(queue, cfg.WorkerCount, taskSvc.HandleJob)
	manager.Start(rootCtx)

	health := handler.NewHealthHandler(memStore)
	rootHandler := handler.Router(articleSvc, taskSvc, health, memStore)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      rootHandler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// 启动 HTTP 服务。
	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	health.SetReady(true)

	// 等待退出信号或服务错误。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		log.Error("server error", "error", err)
	}

	health.SetReady(false)

	// 优雅关闭 HTTP 服务。
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}

	// 停止 worker 池并等待其退出。
	cancel()
	manager.Wait()
	log.Info("server stopped")
}

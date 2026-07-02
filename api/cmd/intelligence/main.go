package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xuroi/xuroi/api/internal/config"
	"github.com/xuroi/xuroi/api/internal/db"
	"github.com/xuroi/xuroi/api/internal/intelligence"
	"github.com/xuroi/xuroi/api/internal/site"
)

func main() {
	once := flag.Bool("once", false, "run one batch and exit")
	limit := flag.Int("limit", 50, "max threads per batch")
	interval := flag.Duration("interval", 30*time.Second, "poll interval when not --once")
	flag.Parse()

	cfg := config.Load()
	siteCfg := site.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	summarizer := intelligence.NewSummarizerFromConfig(cfg.LLM)
	if summarizer != nil {
		log.Printf("llm summaries: %s", summarizer.ModelVersion())
	} else {
		log.Printf("llm summaries: off (heuristic-v1 fallback; set XUROI_LLM_PROVIDER + XUROI_LLM_API_KEY to enable)")
	}

	svc := intelligence.New(pool, summarizer, siteCfg.Intelligence.Enabled)
	run := func() {
		n, err := svc.SummarizeStale(ctx, *limit)
		if err != nil {
			log.Printf("summarize: %v", err)
			return
		}
		if n > 0 {
			log.Printf("summarized %d thread(s)", n)
		}
	}

	run()
	if *once {
		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			run()
		case <-stop:
			return
		}
	}
}
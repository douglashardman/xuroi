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
	"github.com/xuroi/xuroi/api/internal/search"
)

func main() {
	once := flag.Bool("once", false, "run one batch and exit")
	rebuild := flag.Bool("rebuild", false, "rebuild full index then exit")
	limit := flag.Int("limit", 200, "max entities per batch")
	interval := flag.Duration("interval", 15*time.Second, "poll interval when not --once")
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	svc := search.New(pool)

	if *rebuild {
		n, err := search.EnqueueAllPending(ctx, pool)
		if err != nil {
			log.Fatalf("rebuild enqueue: %v", err)
		}
		log.Printf("enqueued %d entities for indexing", n)
		for {
			processed, err := svc.ProcessBatch(ctx, *limit)
			if err != nil {
				log.Fatalf("index: %v", err)
			}
			if processed == 0 {
				break
			}
			log.Printf("indexed %d", processed)
		}
		log.Printf("rebuild complete")
		return
	}

	run := func() {
		n, err := svc.ProcessBatch(ctx, *limit)
		if err != nil {
			log.Printf("search index: %v", err)
			return
		}
		if n > 0 {
			log.Printf("indexed %d document(s)", n)
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
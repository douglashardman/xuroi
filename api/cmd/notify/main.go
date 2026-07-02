package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/config"
	"github.com/xuroi/xuroi/api/internal/db"
	"github.com/xuroi/xuroi/api/internal/email"
	"github.com/xuroi/xuroi/api/internal/notify"
	"github.com/xuroi/xuroi/api/internal/site"
)

func main() {
	once := flag.Bool("once", false, "run one batch and exit")
	limit := flag.Int("limit", 50, "max emails per batch")
	interval := flag.Duration("interval", 60*time.Second, "poll interval when not --once")
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

	emailCfg := email.MergeSiteDefaults(email.ConfigFromEnv(), siteCfg)
	if !siteCfg.Email.Enabled {
		log.Printf("email notifications disabled in site.json")
	}
	mailer, err := email.NewMailer(emailCfg)
	if err != nil {
		log.Fatalf("mailer: %v", err)
	}

	webauthnSvc, err := auth.NewWebAuthn(siteCfg.Site.Name)
	if err != nil {
		log.Fatalf("webauthn: %v", err)
	}
	authSvc := auth.NewService(pool, webauthnSvc)
	svc := notify.New(pool, mailer, emailCfg, siteCfg, authSvc)
	run := func() {
		n, err := svc.ProcessQueue(ctx, *limit)
		if err != nil {
			log.Printf("notify thread: %v", err)
		} else if n > 0 {
			log.Printf("sent %d thread notification(s)", n)
		}
		m, err := svc.ProcessMentionQueue(ctx, *limit)
		if err != nil {
			log.Printf("notify mention: %v", err)
		} else if m > 0 {
			log.Printf("sent %d mention notification(s)", m)
		}
		c, err := svc.ProcessCategoryQueue(ctx, *limit)
		if err != nil {
			log.Printf("notify category: %v", err)
		} else if c > 0 {
			log.Printf("sent %d category watch notification(s)", c)
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
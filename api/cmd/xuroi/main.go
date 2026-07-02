package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xuroi/xuroi/api/internal/auth"
	"github.com/xuroi/xuroi/api/internal/config"
	"github.com/xuroi/xuroi/api/internal/db"
	"github.com/xuroi/xuroi/api/internal/email"
	"github.com/xuroi/xuroi/api/internal/handlers"
	"github.com/xuroi/xuroi/api/internal/media"
	"github.com/xuroi/xuroi/api/internal/notify"
	"github.com/xuroi/xuroi/api/internal/query"
	"github.com/xuroi/xuroi/api/internal/ratelimit"
	"github.com/xuroi/xuroi/api/internal/service"
	"github.com/xuroi/xuroi/api/internal/site"
)

const systemActorID = "act_system"

func main() {
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

	siteCfg := site.Load()
	forum := service.NewForum(pool, systemActorID, siteCfg.Posts)
	if err := forum.EnsureSystemActor(ctx); err != nil {
		log.Fatalf("system actor: %v", err)
	}

	reader := query.NewReader(pool, siteCfg.Site, siteCfg.Posts, siteCfg.Intelligence)

	webauthnSvc, err := auth.NewWebAuthn(siteCfg.Site.Name)
	if err != nil {
		log.Fatalf("webauthn: %v", err)
	}
	authSvc := auth.NewService(pool, webauthnSvc)
	authSvc.SetReservedDisplayNames(siteCfg.ReservedDisplayNames)

	emailCfg := email.MergeSiteDefaults(email.ConfigFromEnv(), siteCfg)
	mailer, err := email.NewMailer(emailCfg)
	if err != nil {
		log.Fatalf("mailer: %v", err)
	}
	notifySvc := notify.New(pool, mailer, emailCfg, siteCfg, authSvc)
	mediaStore, err := media.NewStore(cfg.MediaDir)
	if err != nil {
		log.Fatalf("media store: %v", err)
	}
	log.Printf("media uploads: %s", mediaStore.Dir())

	limiter := ratelimit.New()
	api := handlers.New(pool, forum, reader, authSvc, mediaStore, limiter, notifySvc, siteCfg)
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      api.Routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("xuroi api listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-it/internal/cases"
	"go-it/internal/config"
	"go-it/internal/content"
	web "go-it/internal/http"
	"go-it/internal/http/handlers"
	"go-it/internal/http/render"
	"go-it/internal/repository/memory"
	"go-it/internal/repository/postgres"
	"go-it/internal/service"
	"go-it/internal/session"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	renderer, err := render.New("templates")
	if err != nil {
		log.Error("templates", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	var (
		catalog handlers.Catalog
		store   service.ApplicationStore
		targets service.TargetChecker
		ready   handlers.HealthPinger
		closer  func()
	)

	if cfg.DatabaseURL == "" {
		mem := memory.New()
		catalog, store, targets = mem, mem, mem
		log.Info("using in-memory store")
	} else {
		pg, err := postgres.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Error("postgres", "err", err)
			os.Exit(1)
		}
		catalog, store, targets, ready = pg, pg, pg, pg
		closer = pg.Close
		log.Info("connected to postgres")
	}

	srv := &handlers.Server{
		Render:   renderer,
		Sessions: session.New(cfg.SessionSecret),
		CaseSvc:  service.NewCaseService(cases.All()),
		Apps:     service.NewApplicationService(store, targets),
		Catalog:  catalog,
		Content:  content.New("content"),
		ReadyDB:  ready,
	}

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           web.NewRouter(srv, "static"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	if closer != nil {
		closer()
	}
	log.Info("stopped")
}

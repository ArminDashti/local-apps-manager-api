package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ArminDashti/local-apps-manager-api/internal/apps"
	"github.com/ArminDashti/local-apps-manager-api/internal/auth"
	"github.com/ArminDashti/local-apps-manager-api/internal/config"
	httpserver "github.com/ArminDashti/local-apps-manager-api/internal/http"
	"github.com/ArminDashti/local-apps-manager-api/internal/runner"
	"github.com/ArminDashti/local-apps-manager-api/internal/store"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	ctx := context.Background()
	db, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres required: %v", err)
	}
	defer db.Close()

	authSvc := auth.NewService(db, cfg.JWTSecret)
	if err := authSvc.EnsureDefaultUser(ctx, cfg.DefaultUsername, cfg.DefaultPassword); err != nil {
		log.Fatalf("seed default user: %v", err)
	}

	flight := runner.NewFlight()
	native := runner.NewNativeRunner(cfg.NativeRunnerScript, cfg.NativeAppsConfig)
	docker := runner.NewDockerRunner(cfg.DockerRunnerScript, cfg.GitHubRoot)
	server := runner.NewServerRunner()
	router := runner.NewRouter(native, docker, server, flight)
	appsSvc := apps.NewService(cfg, db, router)
	srv := httpserver.New(cfg, authSvc, appsSvc)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("local-apps-manager-api listening on http://%s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

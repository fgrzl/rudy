package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fgrzl/localagent/internal/agent"
	"github.com/fgrzl/localagent/internal/application"
	"github.com/fgrzl/localagent/internal/config"
	httpapi "github.com/fgrzl/localagent/internal/httpapi"
	"github.com/fgrzl/mux"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("LocalAgent exited", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg := config.Load()
	ag, err := agent.New(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}
	defer func() {
		if err := ag.Close(); err != nil {
			log.Error("failed to close agent", "err", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.AutoIndexOnStart {
		log.Info("indexing workspace on startup", "workspace", cfg.WorkspaceDir)
		if _, err := ag.Rebuild(ctx); err != nil {
			return fmt.Errorf("startup indexing failed: %w", err)
		}
	}

	app := application.New(ag, cfg)

	router := mux.NewRouter(
		mux.WithContextPooling(),
		mux.WithHeadFallbackToGet(),
		mux.WithMaxBodyBytes(2<<20),
	)

	if err := router.Configure(httpapi.New(app, cfg, log).Register); err != nil {
		return fmt.Errorf("failed to configure routes: %w", err)
	}

	server := mux.NewServer(cfg.HTTPAddr, router)
	if err := server.Listen(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	}

	return nil
}

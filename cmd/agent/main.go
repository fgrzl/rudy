package main

import (
	"context"
	"errors"
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

	cfg := config.Load()
	ag, err := agent.New(cfg, log)
	if err != nil {
		log.Error("failed to initialize agent", "err", err)
		os.Exit(1)
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
			log.Error("startup indexing failed", "err", err)
			os.Exit(1)
		}
	}

	app := application.New(ag, cfg)

	router := mux.NewRouter(
		mux.WithContextPooling(),
		mux.WithHeadFallbackToGet(),
		mux.WithMaxBodyBytes(2<<20),
	)

	if err := router.Configure(httpapi.New(app, cfg, log).Register); err != nil {
		log.Error("failed to configure routes", "err", err)
		os.Exit(1)
	}

	server := mux.NewServer(cfg.HTTPAddr, router)
	if err := server.Listen(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("server stopped unexpectedly", "err", err)
		os.Exit(1)
	}
}

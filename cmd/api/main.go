package main

import (
	"log/slog"
	"os"

	"github.com/orgapi/config"
	"github.com/orgapi/internal/app"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()

	if err := app.New(cfg, log).Run(); err != nil {
		log.Error("app error", "err", err)
		os.Exit(1)
	}
}

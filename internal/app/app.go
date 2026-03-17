package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/orgapi/config"
	"github.com/orgapi/internal/handler"
	"github.com/orgapi/internal/middleware"
	"github.com/orgapi/internal/repository"
	"github.com/orgapi/internal/service"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type App struct {
	cfg    *config.Config
	log    *slog.Logger
	server *http.Server
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{cfg: cfg, log: log}
}

func (a *App) Run() error {
	// ── DB ────────────────────────────────────────────────────────────────────
	sqlDB, err := sql.Open("postgres", a.cfg.DSN())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	a.log.Info("connected to database")

	// ── Migrations ────────────────────────────────────────────────────────────
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	a.log.Info("migrations applied")

	// ── GORM ──────────────────────────────────────────────────────────────────
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return fmt.Errorf("gorm open: %w", err)
	}

	// ── Wire (log передаётся в каждый слой) ──────────────────────────────────
	deptRepo := repository.NewDepartmentRepo(gormDB, a.log)
	empRepo  := repository.NewEmployeeRepo(gormDB, a.log)

	deptSvc := service.NewDepartmentService(deptRepo, a.log)
	empSvc  := service.NewEmployeeService(empRepo, a.log)

	h := handler.New(deptSvc, empSvc, a.log)

	// ── Middleware chain ──────────────────────────────────────────────────────
	chain := middleware.Recover(a.log,
		middleware.Logger(a.log,
			h.Routes(),
		),
	)

	// ── HTTP server ───────────────────────────────────────────────────────────
	a.server = &http.Server{
		Addr:         fmt.Sprintf(":%s", a.cfg.ServerPort),
		Handler:      chain,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("server started", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("listen: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		a.log.Info("shutdown signal received", "signal", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	a.log.Info("server stopped gracefully")
	return nil
}

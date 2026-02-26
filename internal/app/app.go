package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	apphttp "github.com/SilentPlaces/simple-task-manager/internal/adapters/http"
	"github.com/SilentPlaces/simple-task-manager/internal/adapters/http/handlers"
	"github.com/SilentPlaces/simple-task-manager/internal/adapters/tasks"
	"github.com/SilentPlaces/simple-task-manager/internal/config"
	"github.com/SilentPlaces/simple-task-manager/internal/database"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	usecasetasks "github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/tasks"
)

type App struct {
	cfg    config.Config
	log    logger.Logger
	db     *sql.DB
	server *http.Server
}

func New(cfg config.Config, log logger.Logger) (*App, error) {
	db, err := openDB(cfg.Database, log)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	if err := database.RunMigrations(db, cfg.Database.MigrationsPath, log); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	repo := tasks.NewPostgresTaskRepository(db)
	useCase := usecasetasks.NewTaskUseCase(repo)
	handler := handlers.NewTaskHandler(useCase, log)
	router := apphttp.NewRouter(handler, log)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{
		cfg:    cfg,
		log:    log,
		db:     db,
		server: server,
	}, nil
}

func (a *App) Run() error {
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("Server starting", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server failed: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		a.log.Info("Received shutdown signal", "signal", sig.String())
	case err := <-errCh:
		return err
	}

	return a.Shutdown()
}

func (a *App) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	a.log.Info("Server stopped")

	if err := a.db.Close(); err != nil {
		return fmt.Errorf("closing database: %w", err)
	}
	a.log.Info("Database connection closed")

	return nil
}

func openDB(cfg config.DatabaseConfig, log logger.Logger) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	log.Info("Database connected", "host", cfg.Host, "port", cfg.Port, "database", cfg.Name)
	return db, nil
}

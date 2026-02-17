package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/pressly/goose/v3"
	"github.com/stpnv0/EventBooker/internal/config"
	"github.com/stpnv0/EventBooker/internal/handler"
	"github.com/stpnv0/EventBooker/internal/middleware"
	"github.com/stpnv0/EventBooker/internal/notification"
	"github.com/stpnv0/EventBooker/internal/repository"
	"github.com/stpnv0/EventBooker/internal/router"
	"github.com/stpnv0/EventBooker/internal/scheduler"
	"github.com/stpnv0/EventBooker/internal/service"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/logger"
)

const migrationsDir = "migrations"

type App struct {
	cfg        *config.Config
	log        logger.Logger
	db         *dbpg.DB
	httpServer *http.Server
	scheduler  *scheduler.Scheduler
}

func New(cfg *config.Config) (*App, error) {
	app := &App{cfg: cfg}

	log, err := logger.InitLogger(
		cfg.Logger.LogEngine(),
		"EventBooker",
		cfg.Gin.Mode,
		logger.WithLevel(cfg.Logger.LogLevel()),
	)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	app.log = log

	if err = app.runMigrations(); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	if err = app.initDB(); err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	if err = app.initServices(); err != nil {
		return nil, fmt.Errorf("init services: %w", err)
	}

	return app, nil
}

func (a *App) initDB() error {
	db, err := dbpg.New(
		a.cfg.Postgres.DSN(),
		nil,
		&dbpg.Options{
			MaxOpenConns: a.cfg.Postgres.MaxOpenConns,
			MaxIdleConns: a.cfg.Postgres.MaxIdleConns,
		},
	)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	if err := db.Master.PingContext(context.Background()); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}

	a.db = db
	a.log.LogAttrs(context.Background(), logger.InfoLevel, "database connected",
		logger.String("host", a.cfg.Postgres.Host),
		logger.Int("port", a.cfg.Postgres.Port),
		logger.String("database", a.cfg.Postgres.Database),
	)

	return nil
}

func (a *App) initServices() error {
	eventRepo := repository.NewEventRepo(a.db)
	bookingRepo := repository.NewBookingRepo(a.db)
	userRepo := repository.NewUserRepo(a.db)

	n, err := notification.NewTelegramNotifier(a.cfg.Telegram.BotToken, a.log)
	if err != nil {
		return fmt.Errorf("init notifier: %w", err)
	}

	eventService := service.NewEventService(eventRepo, bookingRepo)
	userService := service.NewUserService(userRepo)
	bookingService := service.NewBookingService(bookingRepo, eventRepo, userRepo, n, a.log)

	a.scheduler = scheduler.New(
		bookingService,
		a.cfg.Scheduler.Interval,
		a.log,
	)

	h := handler.NewHandler(eventService, bookingService, userService)
	r := router.InitRouter(
		a.cfg.Gin.Mode,
		h,
		middleware.RequestID(),
		middleware.RequestLogger(a.log),
		middleware.Recovery(a.log),
	)

	a.httpServer = &http.Server{
		Addr:         a.cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  a.cfg.Server.ReadTimeout,
		WriteTimeout: a.cfg.Server.WriteTimeout,
		IdleTimeout:  a.cfg.Server.IdleTimeout,
	}

	return nil
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go a.scheduler.Start(ctx)

	errCh := make(chan error, 1)
	go func() {
		a.log.LogAttrs(ctx, logger.InfoLevel, "HTTP server starting",
			logger.String("addr", a.httpServer.Addr),
		)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		a.log.LogAttrs(context.Background(), logger.InfoLevel, "shutdown signal received")
	case err := <-errCh:
		return err
	}

	return a.shutdown()
}

func (a *App) shutdown() error {
	a.log.LogAttrs(context.Background(), logger.InfoLevel, "shutting down...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		a.cfg.Server.WriteTimeout,
	)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}
	a.log.LogAttrs(context.Background(), logger.InfoLevel, "HTTP server stopped")

	if err := a.db.Master.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	a.log.LogAttrs(context.Background(), logger.InfoLevel, "database connection closed")

	a.log.LogAttrs(context.Background(), logger.InfoLevel, "app stopped")

	return nil
}

func (a *App) runMigrations() error {
	db, err := sql.Open("postgres", a.cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	a.log.Info("migrations applied successfully")
	return nil
}

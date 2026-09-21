// 排水管网清淤记录系统 —— 后端服务入口。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/drainage/desilting/internal/config"
	"github.com/drainage/desilting/internal/database"
	"github.com/drainage/desilting/internal/httpx"
	"github.com/drainage/desilting/internal/middleware"
	"github.com/drainage/desilting/internal/router"
)

func main() {
	if err := run(); err != nil {
		slog.Error("服务启动或运行失败", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)
	applyTimeZone(cfg.DBTimeZone, logger)

	db, err := database.Open(cfg)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	if cfg.SeedEnabled {
		if err := database.Seed(db, logger); err != nil {
			return fmt.Errorf("初始化演示数据失败: %w", err)
		}
	}

	app := newApp(cfg, db, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("服务已启动",
			"port", cfg.HTTPPort,
			"env", cfg.AppEnv,
			"db", cfg.Describe(),
			"seed", cfg.SeedEnabled,
		)
		if err := app.Listen(":" + cfg.HTTPPort); err != nil {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("监听端口失败: %w", err)
	case <-ctx.Done():
		logger.Info("收到退出信号，开始优雅关闭")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			return fmt.Errorf("关闭服务失败: %w", err)
		}
		logger.Info("服务已关闭")
		return nil
	}
}

func newApp(cfg *config.Config, db *gorm.DB, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "排水管网清淤记录系统",
		DisableStartupMessage: true,
		BodyLimit:             4 * 1024 * 1024,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          30 * time.Second,
		// 所有错误（含 404、参数错误、业务错误）都渲染成统一信封
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return httpx.WriteError(c, err)
		},
	})

	app.Use(middleware.RequestID())
	app.Use(middleware.AccessLog(logger))
	app.Use(middleware.Recover(logger))
	app.Use(middleware.CORS())

	router.Setup(app, db, cfg)
	return app
}

func newLogger(level string) *slog.Logger {
	var parsed slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		parsed = slog.LevelDebug
	case "warn":
		parsed = slog.LevelWarn
	case "error":
		parsed = slog.LevelError
	default:
		parsed = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parsed})
	return slog.New(handler)
}

// applyTimeZone 把进程时区切到配置的时区，保证日志与业务时间一致。
func applyTimeZone(name string, logger *slog.Logger) {
	if strings.TrimSpace(name) == "" {
		return
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		logger.Warn("加载时区失败，继续使用系统默认时区", "timezone", name, "error", err)
		return
	}
	time.Local = location
}

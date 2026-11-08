package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/database"
	"novel-management/backend/internal/httpapi"
	"novel-management/backend/internal/migrations"
	"novel-management/backend/internal/observability"
	"novel-management/backend/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "启动失败：", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := observability.New(cfg.LogLevel)

	if err := ensureDataDir(cfg.DataDir); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("正在连接数据库", "database", cfg.RedactedDatabaseURL())
	db, err := database.Open(ctx, cfg.DatabaseURL, config.DBRetryLimit, log)
	if err != nil {
		return err
	}
	defer func() { _ = db.SQL.Close() }()

	unlock, err := db.TryAdvisoryLock(ctx, database.AdvisoryLockID)
	if err != nil {
		return fmt.Errorf("获取数据库锁失败：%w", err)
	}
	defer unlock()

	if err := migrations.Up(ctx, db.SQL); err != nil {
		return err
	}
	log.Info("数据库迁移完成")

	cachePing, closeCache := newCachePinger(cfg, log)
	defer closeCache()

	logPgDumpVersion(log)

	snapshotService := service.NewSnapshot(db.Gorm, cfg, log, service.PGDumper{})
	router := httpapi.NewRouter(httpapi.Options{
		Config: cfg,
		Log:    log,
		Health: httpapi.HealthDeps{
			DBPing:          db.SQL.PingContext,
			CachePing:       cachePing,
			DataDirWritable: func() bool { return isWritable(cfg.DataDir) },
			Version:         cfg.Version,
		},
		Works:     service.NewWorks(db.Gorm, snapshotService),
		Trash:     service.NewTrash(db.Gorm, snapshotService),
		Snapshots: snapshotService,
		Settings:  service.NewSettings(db.Gorm),
	})

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("HTTP 服务已启动", "addr", cfg.Addr(), "web_dir", cfg.WebDir)
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("收到退出信号，正在关闭")
	case serveErr := <-errCh:
		return fmt.Errorf("HTTP 服务异常退出：%w", serveErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func ensureDataDir(root string) error {
	for _, sub := range []string{"images", "snapshots", "exports", "tmp/imports"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return fmt.Errorf("创建数据目录失败：%w", err)
		}
	}
	if !isWritable(root) {
		return fmt.Errorf("数据目录不可写：%s", root)
	}
	return nil
}

func isWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".write-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// logPgDumpVersion 启动时记录快照依赖的客户端版本，便于排查主版本不匹配。
func logPgDumpVersion(log *slog.Logger) {
	out, err := exec.Command("pg_dump", "--version").Output()
	if err != nil {
		log.Warn("pg_dump 不可用，快照将失败", "error", err)
		return
	}
	log.Info("pg_dump 可用", "version", strings.TrimSpace(string(out)))
}

func newCachePinger(cfg config.Config, log *slog.Logger) (func(context.Context) error, func()) {
	if !cfg.IsCacheEnabled() {
		return nil, func() {}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	ping := func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	}

	probeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(probeCtx).Err(); err != nil {
		log.Warn("Redis 不可用，缓存自动旁路", "error", err)
	}

	return ping, func() { _ = client.Close() }
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/database"
	"novel-management/backend/internal/migrations"
	"novel-management/backend/internal/observability"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "cli",
		Short:        "小说设定管理系统运维 CLI",
		SilenceUsage: true,
	}
	root.AddCommand(newMigrateCmd(), newDoctorCmd(), newVersionCmd())
	return root
}

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "执行数据库迁移（需先停止 app 以释放 advisory lock）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := load()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			db, err := database.Open(ctx, cfg.DatabaseURL, 10*time.Second, log)
			if err != nil {
				return err
			}
			defer func() { _ = db.SQL.Close() }()

			unlock, err := db.TryAdvisoryLock(ctx, database.AdvisoryLockID)
			if err != nil {
				return err
			}
			defer unlock()

			if err := migrations.Up(ctx, db.SQL); err != nil {
				return err
			}
			fmt.Println("迁移完成")
			return nil
		},
	}
}

type doctorReport struct {
	AppVersion      string `json:"app_version"`
	Addr            string `json:"addr"`
	DataDir         string `json:"data_dir"`
	DataDirWritable bool   `json:"data_dir_writable"`
	WebDir          string `json:"web_dir"`
	Database        string `json:"database"`
	RedisAddr       string `json:"redis_addr"`
	DBOK            bool   `json:"db_ok"`
	CacheStatus     string `json:"cache_status"`
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "输出脱敏配置与依赖连通性自检",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := load()
			if err != nil {
				return err
			}

			report := doctorReport{
				AppVersion:      cfg.Version,
				Addr:            cfg.Addr(),
				DataDir:         cfg.DataDir,
				DataDirWritable: dataDirWritable(cfg.DataDir),
				WebDir:          cfg.WebDir,
				Database:        cfg.RedactedDatabaseURL(),
				RedisAddr:       cfg.RedisAddr,
				CacheStatus:     "disabled",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if db, openErr := database.Open(ctx, cfg.DatabaseURL, 10*time.Second, log); openErr != nil {
				report.DBOK = false
			} else {
				report.DBOK = true
				_ = db.SQL.Close()
			}

			if cfg.IsCacheEnabled() {
				client := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
				defer func() { _ = client.Close() }()
				report.CacheStatus = "ok"
				probeCtx, probeCancel := context.WithTimeout(ctx, 2*time.Second)
				defer probeCancel()
				if pingErr := client.Ping(probeCtx).Err(); pingErr != nil {
					report.CacheStatus = "degraded"
				}
			}

			out, marshalErr := json.MarshalIndent(report, "", "  ")
			if marshalErr != nil {
				return marshalErr
			}
			fmt.Println(string(out))
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "输出版本号",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.Version)
		},
	}
}

func load() (config.Config, *slog.Logger, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, nil, err
	}
	return cfg, observability.New(cfg.LogLevel), nil
}

func dataDirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".write-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

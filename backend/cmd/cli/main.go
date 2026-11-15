package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/database"
	"novel-management/backend/internal/migrations"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/observability"
	"novel-management/backend/internal/search"
	"novel-management/backend/internal/service"
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
	root.AddCommand(newMigrateCmd(), newDoctorCmd(), newReindexCmd(), newRestoreCmd(), newRestoreSnapshotCmd(), newVersionCmd())
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

func newReindexCmd() *cobra.Command {
	var check bool
	var workID string
	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "重建或校验 search_documents 检索文档（需先停止 app）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, log, err := load()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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

			refresher := search.NewRefresher(db.Gorm)
			if check {
				workIDs, err := listReindexWorks(ctx, db.Gorm, workID)
				if err != nil {
					return err
				}
				var drift int64
				for _, id := range workIDs {
					consistency, err := refresher.Consistency(ctx, id)
					if err != nil {
						return err
					}
					drift += consistency.Drift
					fmt.Printf("作品 %s：文档 %d / 实体 %d，漂移 %d\n",
						id, consistency.TotalDocs, consistency.TotalEntities, consistency.Drift)
				}
				if drift != 0 {
					return fmt.Errorf("检索文档存在漂移：%d", drift)
				}
				fmt.Println("一致性检查通过")
				return nil
			}

			if workID != "" {
				if err := refresher.RefreshWork(ctx, workID); err != nil {
					return err
				}
				fmt.Printf("已重建作品 %s 的检索文档\n", workID)
				return nil
			}
			count, err := refresher.RebuildAll(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("已重建 %d 个作品的检索文档\n", count)
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "只做一致性检查，不修改数据")
	cmd.Flags().StringVar(&workID, "work", "", "仅处理指定作品 ID")
	return cmd
}

func listReindexWorks(ctx context.Context, db *gorm.DB, workID string) ([]string, error) {
	if workID != "" {
		return []string{workID}, nil
	}
	var ids []string
	if err := db.WithContext(ctx).Model(&model.Work{}).Order("id").Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func newRestoreCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "restore <zip>",
		Short: "从导出 zip 全量恢复（需先停止 app）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args[0], force, false)
		},
	}
	cmd.Flags().BoolVar(&force, "yes", false, "实例非空时确认覆盖")
	return cmd
}

func newRestoreSnapshotCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "restore-snapshot <dir>",
		Short: "从自动快照目录全量恢复（需先停止 app）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args[0], force, true)
		},
	}
	cmd.Flags().BoolVar(&force, "yes", false, "实例非空时确认覆盖")
	return cmd
}

func runRestore(source string, force, snapshot bool) error {
	cfg, log, err := load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	db, err := database.Open(ctx, cfg.DatabaseURL, 10*time.Second, log)
	if err != nil {
		return err
	}
	defer func() { _ = db.SQL.Close() }()

	unlock, err := db.TryAdvisoryLock(ctx, database.AdvisoryLockID)
	if err != nil {
		if errors.Is(err, database.ErrLockBusy) {
			return fmt.Errorf("应用仍在运行或正在执行其他任务，请先执行 docker compose stop app")
		}
		return err
	}
	defer unlock()

	snapshots := service.NewSnapshot(db.Gorm, cfg, log, service.PGDumper{})
	restore := service.NewRestore(db.Gorm, cfg, log, snapshots, nil)
	options := service.RestoreOptions{Force: force}

	var summary *service.RestoreSummary
	if snapshot {
		summary, err = restore.RestoreSnapshot(ctx, source, options)
	} else {
		summary, err = restore.RestoreArchive(ctx, source, options)
	}
	if err != nil {
		return err
	}

	fmt.Printf("恢复完成：%s\n", summary.Source)
	fmt.Printf("数据版本：%d\n", summary.SchemaVersion)
	fmt.Printf("恢复前快照：%s\n", summary.SnapshotPath)
	fmt.Printf("作品：%d，人物：%d，图片：%d\n",
		summary.Counts["works"], summary.Counts["characters"], summary.Counts["images"])
	if summary.Legacy {
		fmt.Println("提示：旧快照缺少 schema_version，已按兼容模式恢复")
	}
	fmt.Printf("耗时：%s\n", summary.Elapsed.Round(time.Millisecond))
	return nil
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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
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
	root.AddCommand(newMigrateCmd(), newDoctorCmd(), newReindexCmd(), newRestoreCmd(), newRestoreSnapshotCmd(), newSeedCmd(), newVersionCmd())
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

func newSeedCmd() *cobra.Command {
	var workName string
	var characters, factions, locations int
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "批量生成性能基线数据（默认 1000 人物），完成后刷新检索",
		RunE: func(cmd *cobra.Command, args []string) error {
			const maxCharacters = 100000
			const maxRelated = 10000
			if characters < 0 || factions < 0 || locations < 0 {
				return errors.New("造数数量不能为负数")
			}
			if characters > maxCharacters {
				return fmt.Errorf("人物数量不能超过 %d", maxCharacters)
			}
			if factions > maxRelated || locations > maxRelated {
				return fmt.Errorf("势力与地点数量不能超过 %d", maxRelated)
			}
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

			// 允许在 app 运行时造数：写入在单事务内完成，不与非空恢复类命令争抢排他锁。
			work, err := seedWork(ctx, db.Gorm, workName)
			if err != nil {
				return err
			}
			if err := seedRecords(ctx, db.Gorm, work.ID, characters, factions, locations); err != nil {
				return err
			}
			if err := search.NewRefresher(db.Gorm).RefreshWork(ctx, work.ID); err != nil {
				return err
			}
			fmt.Printf("seed work_id=%s characters=%d factions=%d locations=%d\n",
				work.ID, characters, factions, locations)
			return nil
		},
	}
	cmd.Flags().StringVar(&workName, "work", "性能基线", "目标作品名称；不存在时自动创建")
	cmd.Flags().IntVar(&characters, "characters", 1000, "生成的人物数量")
	cmd.Flags().IntVar(&factions, "factions", 50, "生成的势力数量")
	cmd.Flags().IntVar(&locations, "locations", 50, "生成的地点数量")
	return cmd
}

// seedWork 复用同名作品；不存在时连同内置「境界」类别一起创建。
func seedWork(ctx context.Context, db *gorm.DB, rawName string) (*model.Work, error) {
	name, nameKey, err := model.NormalizeName(rawName)
	if err != nil {
		return nil, err
	}
	var work model.Work
	err = db.WithContext(ctx).Where("name_key = ? AND deleted_at IS NULL", nameKey).First(&work).Error
	if err == nil {
		return &work, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	work = model.Work{
		ID:        model.NewID(),
		Name:      name,
		NameKey:   nameKey,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	category := model.SystemCategory{
		ID:           model.NewID(),
		WorkID:       work.ID,
		Name:         model.BuiltinRealmCategory,
		NameKey:      strings.ToLower(model.BuiltinRealmCategory),
		CategoryType: model.CategoryTypeNormal,
		IsBuiltin:    true,
		IsEnabled:    true,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&work).Error; err != nil {
			return err
		}
		return tx.Create(&category).Error
	})
	if err != nil {
		return nil, err
	}
	return &work, nil
}

// seedRecords 在单事务内批量写入势力、地点与人物；名称带批次后缀，重复执行不会冲突。
func seedRecords(ctx context.Context, db *gorm.DB, workID string, characters, factions, locations int) error {
	now := time.Now().UTC()
	tag := now.Format("150405") + "-" + model.NewID()[:4]

	factionRows := make([]model.Faction, 0, factions)
	for i := 1; i <= factions; i++ {
		name := fmt.Sprintf("性能势力-%s-%04d", tag, i)
		factionRows = append(factionRows, model.Faction{
			ID:           model.NewID(),
			WorkID:       workID,
			Name:         name,
			NameKey:      strings.ToLower(name),
			ReviewStatus: model.ReviewStatusReviewed,
			Version:      1,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	locationRows := make([]model.Location, 0, locations)
	for i := 1; i <= locations; i++ {
		name := fmt.Sprintf("性能地点-%s-%04d", tag, i)
		locationRows = append(locationRows, model.Location{
			ID:           model.NewID(),
			WorkID:       workID,
			Name:         name,
			NameKey:      strings.ToLower(name),
			ReviewStatus: model.ReviewStatusReviewed,
			Version:      1,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	characterRows := make([]model.Character, 0, characters)
	for i := 1; i <= characters; i++ {
		name := fmt.Sprintf("性能人物-%s-%04d", tag, i)
		var factionID *string
		if len(factionRows) > 0 {
			id := factionRows[(i-1)%len(factionRows)].ID
			factionID = &id
		}
		characterRows = append(characterRows, model.Character{
			ID:           model.NewID(),
			WorkID:       workID,
			Name:         name,
			NameKey:      strings.ToLower(name),
			FactionID:    factionID,
			ReviewStatus: model.ReviewStatusReviewed,
			Version:      1,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(factionRows) > 0 {
			if err := tx.CreateInBatches(factionRows, 500).Error; err != nil {
				return err
			}
		}
		if len(locationRows) > 0 {
			if err := tx.CreateInBatches(locationRows, 500).Error; err != nil {
				return err
			}
		}
		if len(characterRows) > 0 {
			if err := tx.CreateInBatches(characterRows, 500).Error; err != nil {
				return err
			}
		}
		return nil
	})
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

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/backup"
	"novel-management/backend/internal/config"
	"novel-management/backend/internal/migrations"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
	"novel-management/backend/internal/search"
)

var (
	// ErrRestoreNotEmpty 表示实例已有数据且未确认覆盖。
	ErrRestoreNotEmpty = errors.New("实例已有数据，请确认清空或使用空实例")
	// ErrRestoreInvalid 表示导出包/快照校验失败。
	ErrRestoreInvalid = errors.New("恢复包校验失败")
)

// SnapshotCreator 抽象恢复前快照，便于测试注入。
type SnapshotCreator interface {
	Create(ctx context.Context, reason string) (*model.Snapshot, error)
}

// RestoreRunner 抽象 pg_restore，便于测试注入。
type RestoreRunner interface {
	Restore(ctx context.Context, dsn, dumpPath string) error
}

// PGRestoreRunner 调用运行时镜像内的 pg_restore。
type PGRestoreRunner struct{}

func (PGRestoreRunner) Restore(ctx context.Context, dsn, dumpPath string) error {
	return runPGCommand(ctx, dsn, "pg_restore",
		"--clean", "--if-exists", "--no-owner", "--no-privileges",
		"--single-transaction", "-d", dsn, dumpPath)
}

// RestoreOptions 控制覆盖行为。
type RestoreOptions struct {
	Force bool
}

// RestoreSummary 是 CLI 恢复后的摘要。
type RestoreSummary struct {
	Source        string
	SchemaVersion int
	SnapshotPath  string
	Counts        map[string]int64
	ImageCount    int
	Elapsed       time.Duration
	Legacy        bool
}

// Restore 负责导出包与快照的全量恢复。
type Restore struct {
	db        *gorm.DB
	cfg       config.Config
	log       *slog.Logger
	snapshots SnapshotCreator
	runner    RestoreRunner
	now       func() time.Time
}

func NewRestore(db *gorm.DB, cfg config.Config, log *slog.Logger, snapshots SnapshotCreator, runner RestoreRunner) *Restore {
	if runner == nil {
		runner = PGRestoreRunner{}
	}
	return &Restore{
		db: db, cfg: cfg, log: log, snapshots: snapshots, runner: runner, now: time.Now,
	}
}

// RestoreArchive 校验 zip 后恢复到当前实例。
func (r *Restore) RestoreArchive(ctx context.Context, zipPath string, opts RestoreOptions) (*RestoreSummary, error) {
	workDir := filepath.Join(r.cfg.DataDir, "tmp", "restore", model.NewID())
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	if _, err := backup.ExtractZip(zipPath, workDir); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	manifest, err := backup.ReadExportManifest(workDir)
	if err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	if err := manifest.ValidateSchema(migrations.LatestVersion); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	if err := backup.VerifyChecksums(workDir, filepath.Join(workDir, backup.ChecksumFile)); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	if err := precheckDisk(r.cfg.DataDir, manifest.Files); err != nil {
		return nil, err
	}
	dumpPath := filepath.Join(workDir, "data", "app.dump")
	if _, err := os.Stat(dumpPath); err != nil {
		return nil, fmt.Errorf("%w：导出包缺少 data/app.dump", ErrRestoreInvalid)
	}

	return r.restore(ctx, zipPath, dumpPath, filepath.Join(workDir, "images"),
		manifest.Counts, manifest.SchemaVersion, manifest.Images.Count, opts)
}

// RestoreSnapshot 校验快照目录后恢复到当前实例。
func (r *Restore) RestoreSnapshot(ctx context.Context, dir string, opts RestoreOptions) (*RestoreSummary, error) {
	if _, err := os.Stat(filepath.Join(dir, "snapshot.json")); err != nil {
		return nil, fmt.Errorf("%w：快照缺少 snapshot.json", ErrRestoreInvalid)
	}
	manifest, legacy, err := backup.ReadSnapshotManifest(dir)
	if err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	if err := manifest.ValidateSchema(migrations.LatestVersion); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	if err := backup.VerifyChecksums(dir, filepath.Join(dir, backup.ChecksumFile)); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	if err := precheckDisk(r.cfg.DataDir, manifest.Files); err != nil {
		return nil, err
	}
	dumpPath := filepath.Join(dir, "app.dump")
	if _, err := os.Stat(dumpPath); err != nil {
		return nil, fmt.Errorf("%w：快照缺少 app.dump", ErrRestoreInvalid)
	}

	var meta snapshotMeta
	if err := backup.ReadJSON(filepath.Join(dir, "snapshot.json"), &meta); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrRestoreInvalid, err)
	}
	imageCount := countFiles(filepath.Join(dir, "images"))

	summary, err := r.restore(ctx, dir, dumpPath, filepath.Join(dir, "images"),
		meta.EntityCounts, manifest.SchemaVersion, imageCount, opts)
	if summary != nil {
		summary.Legacy = legacy
	}
	if legacy {
		r.log.Warn("旧快照缺少 schema_version，已按兼容模式恢复", "dir", dir)
	}
	return summary, err
}

func (r *Restore) restore(
	ctx context.Context,
	source, dumpPath, imagesDir string,
	expected map[string]int64,
	schemaVersion, imageCount int,
	opts RestoreOptions,
) (*RestoreSummary, error) {
	started := time.Now()

	var works int64
	if err := r.db.WithContext(ctx).Model(&model.Work{}).Count(&works).Error; err != nil {
		return nil, err
	}
	if works > 0 && !opts.Force {
		return nil, fmt.Errorf("%w：检测到 %d 个作品；确认覆盖请追加 --yes，或改用空实例", ErrRestoreNotEmpty, works)
	}

	snapshot, err := r.snapshots.Create(ctx, model.SnapshotReasonCLIRestore)
	if err != nil {
		return nil, fmt.Errorf("恢复前快照失败，已取消恢复：%w", err)
	}
	rollback := fmt.Sprintf("docker compose run --rm cli restore-snapshot %s --yes", snapshot.Path)

	if err := r.runner.Restore(ctx, r.cfg.DatabaseURL, dumpPath); err != nil {
		return nil, fmt.Errorf("恢复数据库失败：%w；原数据未修改。回滚命令：%s", err, rollback)
	}

	if err := replaceImages(r.cfg.DataDir, imagesDir); err != nil {
		return nil, fmt.Errorf("替换图片失败：%w；数据库已恢复，请执行回滚：%s", err, rollback)
	}

	sqlDB, err := r.db.DB()
	if err != nil {
		return nil, err
	}
	if err := migrations.Up(ctx, sqlDB); err != nil {
		return nil, fmt.Errorf("恢复后迁移失败：%w；请执行回滚：%s", err, rollback)
	}
	if _, err := search.NewRefresher(r.db).RebuildAll(ctx); err != nil {
		return nil, fmt.Errorf("恢复后重建检索失败：%w；请执行回滚：%s", err, rollback)
	}

	finished := r.now().UTC()
	if err := r.db.WithContext(ctx).Table("app_settings").Where("id = 1").
		Update("last_restore_at", finished).Error; err != nil {
		r.log.Warn("写入最近恢复时间失败", "error", err)
	}
	if err := r.afterRestore(ctx, snapshot); err != nil {
		r.log.Warn("恢复后整理快照记录失败", "error", err)
	}

	if err := verifyCounts(ctx, r.db, expected); err != nil {
		return nil, fmt.Errorf("恢复后计数核对失败：%w；请执行回滚：%s", err, rollback)
	}
	counts := map[string]int64{}
	for _, table := range []string{"works", "characters", "images"} {
		if table == "images" {
			counts[table] = int64(countFiles(filepath.Join(r.cfg.DataDir, "images")))
			continue
		}
		total, err := countActive(ctx, r.db, table)
		if err != nil {
			return nil, err
		}
		counts[table] = total
	}

	r.log.Info("恢复完成", "source", source, "snapshot", snapshot.Path,
		"elapsed", time.Since(started).String())

	return &RestoreSummary{
		Source:        source,
		SchemaVersion: schemaVersion,
		SnapshotPath:  snapshot.Path,
		Counts:        counts,
		ImageCount:    imageCount,
		Elapsed:       time.Since(started),
	}, nil
}

// afterRestore 重新登记恢复前快照，并清理指向不存在目录的历史记录。
func (r *Restore) afterRestore(ctx context.Context, snapshot *model.Snapshot) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(snapshot).Error; err != nil {
		return err
	}
	repo := repository.NewSnapshots(r.db)
	rows, err := repo.ListRecent(ctx, 200)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.ID == snapshot.ID {
			continue
		}
		if _, err := os.Stat(row.Path); os.IsNotExist(err) {
			if err := repo.DeleteOne(ctx, row.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// replaceImages 原子替换图片目录；失败时恢复旧目录。
func replaceImages(dataDir, sourceDir string) error {
	target := filepath.Join(dataDir, "images")
	backupDir := filepath.Join(dataDir, fmt.Sprintf("images.pre-restore-%d", time.Now().UnixNano()))

	hasOld := false
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backupDir); err != nil {
			return err
		}
		hasOld = true
	}
	restoreOld := func() {
		_ = os.RemoveAll(target)
		if hasOld {
			_ = os.Rename(backupDir, target)
		}
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		restoreOld()
		return err
	}
	if _, err := os.Stat(sourceDir); err == nil {
		if _, err := copyTree(sourceDir, target); err != nil {
			restoreOld()
			return err
		}
	}
	if hasOld {
		_ = os.RemoveAll(backupDir)
	}
	return nil
}

func precheckDisk(dataDir string, files []backup.ManifestFile) error {
	var required int64
	for _, file := range files {
		required += file.Size
	}
	if required <= 0 {
		return nil
	}
	available, err := backup.AvailableBytes(dataDir)
	if err != nil {
		return nil
	}
	if available == 0 {
		return nil
	}
	const reserve = int64(1 << 30)
	if int64(available) < required+reserve {
		return fmt.Errorf("磁盘空间不足：需要约 %d MB，可用 %d MB", (required+reserve)/(1<<20), int64(available)/(1<<20))
	}
	return nil
}

func verifyCounts(ctx context.Context, db *gorm.DB, expected map[string]int64) error {
	for table, want := range expected {
		if table == "images" {
			continue
		}
		got, err := countActive(ctx, db, table)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("%s 计数不一致：期望 %d，实际 %d", table, want, got)
		}
	}
	return nil
}

func countActive(ctx context.Context, db *gorm.DB, table string) (int64, error) {
	query := db.WithContext(ctx).Table(table)
	if table != "ranking_seats" {
		query = query.Where("deleted_at IS NULL")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func countFiles(root string) int {
	entries, err := backup.ListFiles(root, nil)
	if err != nil {
		return 0
	}
	return len(entries)
}

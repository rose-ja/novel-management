package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/backup"
	"novel-management/backend/internal/config"
	"novel-management/backend/internal/migrations"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

var (
	// ErrExportInProgress 表示已有排队或执行中的导出任务。
	ErrExportInProgress = errors.New("已有导出任务在进行")
	// ErrExportNoWork 表示当前没有任何活跃作品，无需导出。
	ErrExportNoWork = errors.New("暂无作品数据可导出")
)

// ExportDumper 抽象数据库转储；DumpWithoutTrash 需真实排除软删除数据。
type ExportDumper interface {
	Dump(ctx context.Context, dsn, outPath string) error
	DumpWithoutTrash(ctx context.Context, dsn, outPath string) error
}

// Export 负责导出任务的创建、后台执行与状态查询。
type Export struct {
	db     *gorm.DB
	cfg    config.Config
	log    *slog.Logger
	dumper ExportDumper
	now    func() time.Time
}

func NewExport(db *gorm.DB, cfg config.Config, log *slog.Logger, dumper ExportDumper) *Export {
	if dumper == nil {
		dumper = NewPGExportDumper(db)
	}
	return &Export{db: db, cfg: cfg, log: log, dumper: dumper, now: time.Now}
}

// Create 在单事务内检查进行中任务并写入排队记录，返回后可异步执行 Run。
func (s *Export) Create(ctx context.Context, includeTrash bool) (*model.ExportRun, error) {
	now := s.now()
	record := &model.ExportRun{
		ID:           model.NewID(),
		Status:       model.ExportStatusQueued,
		IncludeTrash: includeTrash,
		CreatedAt:    now.UTC(),
	}
	fileName := "novel-settings-" + now.Format("20060102-150405") + ".zip"
	record.FileName = &fileName

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT id FROM app_settings WHERE id = 1 FOR UPDATE").Error; err != nil {
			return err
		}
		var active model.ExportRun
		findErr := tx.Where("status IN ?", []string{model.ExportStatusQueued, model.ExportStatusRunning}).
			First(&active).Error
		if findErr == nil {
			return ErrExportInProgress
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		var works int64
		if err := tx.Model(&model.Work{}).Where("deleted_at IS NULL").Count(&works).Error; err != nil {
			return err
		}
		if works == 0 {
			return ErrExportNoWork
		}
		return tx.Create(record).Error
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}

// Get 返回任务详情；不存在时映射为 ErrNotFound。
func (s *Export) Get(ctx context.Context, id string) (*model.ExportRun, error) {
	run, err := repository.NewExports(s.db).Get(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Export) List(ctx context.Context, limit int) ([]model.ExportRun, error) {
	return repository.NewExports(s.db).ListRecent(ctx, limit)
}

// ExportStatus 是设置页轮询所需的聚合状态。
type ExportStatus struct {
	Active *model.ExportRun `json:"active"`
	Last   *model.ExportRun `json:"last"`
}

func (s *Export) Status(ctx context.Context) (ExportStatus, error) {
	var status ExportStatus
	repo := repository.NewExports(s.db)
	active, err := repo.Active(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return status, err
	}
	if err == nil {
		status.Active = active
	}
	last, err := repo.LastFinished(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return status, err
	}
	if err == nil {
		status.Last = last
	}
	return status, nil
}

// MarkInterrupted 在启动时清理上次进程遗留的任务。
func (s *Export) MarkInterrupted(ctx context.Context) error {
	return repository.NewExports(s.db).MarkInterrupted(ctx, s.now().UTC())
}

// Run 执行导出的全部步骤；任何失败都会把任务置为 failed 并清理半成品。
func (s *Export) Run(ctx context.Context, id string) {
	repo := repository.NewExports(s.db)
	record, err := repo.Get(ctx, id)
	if err != nil {
		s.log.Error("读取导出任务失败", "id", id, "error", err)
		return
	}
	if record.Status != model.ExportStatusQueued {
		return
	}

	fileName := s.uniqueFileName(*record.FileName, record.ID)
	relPath := filepath.ToSlash(filepath.Join("exports", fileName))
	if err := repo.Update(ctx, record.ID, map[string]any{
		"status":    model.ExportStatusRunning,
		"file_name": fileName,
		"file_path": relPath,
	}); err != nil {
		s.log.Error("更新导出任务状态失败", "id", id, "error", err)
		return
	}
	record.FileName = &fileName
	record.FilePath = &relPath

	started := time.Now()
	workDir := filepath.Join(s.cfg.DataDir, "tmp", "exports", record.ID)
	_ = os.RemoveAll(workDir)
	defer func() { _ = os.RemoveAll(workDir) }()

	zipPath := filepath.Join(s.cfg.DataDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		s.fail(ctx, repo, record, model.ExportErrorFailed, err)
		return
	}

	if _, err := s.buildArchive(ctx, workDir, record); err != nil {
		_ = os.Remove(zipPath)
		code := model.ExportErrorFailed
		if errors.Is(err, ErrExportNoWork) {
			code = model.ExportErrorNoWork
		}
		s.fail(ctx, repo, record, code, err)
		return
	}

	info, err := os.Stat(zipPath)
	if err != nil {
		s.fail(ctx, repo, record, model.ExportErrorFailed, err)
		return
	}
	finished := s.now().UTC()
	if err := repo.Update(ctx, record.ID, map[string]any{
		"status":      model.ExportStatusCompleted,
		"byte_size":   info.Size(),
		"finished_at": finished,
	}); err != nil {
		s.fail(ctx, repo, record, model.ExportErrorFailed, err)
		return
	}
	if err := s.db.WithContext(ctx).Table("app_settings").Where("id = 1").
		Update("last_export_at", finished).Error; err != nil {
		s.log.Warn("写入最近导出时间失败", "error", err)
	}
	s.log.Info("导出完成", "id", record.ID, "file", relPath, "bytes", info.Size(),
		"elapsed", time.Since(started).String())
}

func (s *Export) fail(ctx context.Context, repo *repository.Exports, record *model.ExportRun, code string, cause error) {
	s.log.Error("导出失败", "id", record.ID, "code", code, "error", cause)
	if err := repo.Update(ctx, record.ID, map[string]any{
		"status":      model.ExportStatusFailed,
		"error_code":  code,
		"finished_at": s.now().UTC(),
	}); err != nil {
		s.log.Error("写入导出失败状态失败", "id", record.ID, "error", err)
	}
}

// buildArchive 生成 data/app.dump、images/、README、manifest 与 checksums，并打包 zip。
func (s *Export) buildArchive(ctx context.Context, workDir string, record *model.ExportRun) (backup.ExportManifest, error) {
	var manifest backup.ExportManifest
	if err := os.MkdirAll(filepath.Join(workDir, "data"), 0o755); err != nil {
		return manifest, err
	}
	dumpPath := filepath.Join(workDir, "data", "app.dump")
	var err error
	if record.IncludeTrash {
		err = s.dumper.Dump(ctx, s.cfg.DatabaseURL, dumpPath)
	} else {
		err = s.dumper.DumpWithoutTrash(ctx, s.cfg.DatabaseURL, dumpPath)
	}
	if err != nil {
		return manifest, err
	}

	imageFiles, err := s.copyActiveImages(ctx, workDir)
	if err != nil {
		return manifest, err
	}
	works, counts, err := s.activeSummary(ctx)
	if err != nil {
		return manifest, err
	}

	if err := os.WriteFile(filepath.Join(workDir, "README.txt"),
		[]byte(readmeContent(record.CreatedAt, record.IncludeTrash, *record.FileName)), 0o644); err != nil {
		return manifest, err
	}

	files, err := backup.ListFiles(workDir, nil)
	if err != nil {
		return manifest, err
	}
	manifest = backup.ExportManifest{
		ExportVersion: backup.ExportVersion,
		SchemaVersion: migrations.LatestVersion,
		AppVersion:    s.cfg.Version,
		GeneratedAt:   s.now().UTC().Format(time.RFC3339),
		IncludesTrash: record.IncludeTrash,
		Works:         works,
		Counts:        counts,
		Images:        backup.ManifestImages{Count: len(imageFiles), Files: imageFiles},
		Files:         toManifestFiles(files),
	}
	if err := backup.WriteJSON(filepath.Join(workDir, "manifest.json"), manifest); err != nil {
		return manifest, err
	}

	entries, err := backup.ListFiles(workDir, nil)
	if err != nil {
		return manifest, err
	}
	if err := backup.WriteChecksums(workDir, filepath.Join(workDir, backup.ChecksumFile), entries); err != nil {
		return manifest, err
	}
	entries, err = backup.ListFiles(workDir, nil)
	if err != nil {
		return manifest, err
	}
	zipPath := filepath.Join(s.cfg.DataDir, "exports", *record.FileName)
	if err := backup.WriteZip(zipPath, workDir, entries); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func (s *Export) copyActiveImages(ctx context.Context, workDir string) ([]string, error) {
	var assets []model.MediaAsset
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Find(&assets).Error; err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var files []string
	for _, asset := range assets {
		paths := []string{asset.OriginalPath}
		if asset.ThumbPath != nil && *asset.ThumbPath != "" {
			paths = append(paths, *asset.ThumbPath)
		}
		for _, rel := range paths {
			if rel == "" || seen[rel] {
				continue
			}
			source := filepath.Join(s.cfg.DataDir, filepath.FromSlash(rel))
			if _, err := os.Stat(source); err != nil {
				s.log.Warn("导出跳过缺失图片", "path", rel, "error", err)
				continue
			}
			target, err := backup.SafeJoin(workDir, rel)
			if err != nil {
				return nil, err
			}
			if err := copyFile(source, target); err != nil {
				return nil, err
			}
			seen[rel] = true
			files = append(files, rel)
		}
	}
	sort.Strings(files)
	return files, nil
}

func (s *Export) activeSummary(ctx context.Context) ([]backup.ManifestWork, map[string]int64, error) {
	var works []model.Work
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Order("id").Find(&works).Error; err != nil {
		return nil, nil, err
	}
	items := make([]backup.ManifestWork, 0, len(works))
	for _, work := range works {
		items = append(items, backup.ManifestWork{ID: work.ID, Name: work.Name})
	}

	tables := []string{
		"works", "system_categories", "system_nodes", "characters", "factions",
		"locations", "rankings", "ranking_seats", "chronicles", "media_assets",
	}
	counts := make(map[string]int64, len(tables))
	for _, table := range tables {
		query := s.db.WithContext(ctx).Table(table)
		if table != "ranking_seats" {
			query = query.Where("deleted_at IS NULL")
		}
		var total int64
		if err := query.Count(&total).Error; err != nil {
			return nil, nil, err
		}
		counts[table] = total
	}
	return items, counts, nil
}

func toManifestFiles(entries []backup.Entry) []backup.ManifestFile {
	files := make([]backup.ManifestFile, 0, len(entries))
	for _, entry := range entries {
		files = append(files, backup.ManifestFile{Path: entry.Path, Size: entry.Size})
	}
	return files
}

// uniqueFileName 避免同一秒重复导出覆盖旧包。
func (s *Export) uniqueFileName(base, id string) string {
	name := base
	if _, err := os.Stat(filepath.Join(s.cfg.DataDir, "exports", name)); err != nil {
		return name
	}
	suffix := id
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	return strings.TrimSuffix(base, ".zip") + "-" + suffix + ".zip"
}

func readmeContent(created time.Time, includesTrash bool, fileName string) string {
	trash := "否"
	if includesTrash {
		trash = "是"
	}
	return fmt.Sprintf(`小说设定管理系统 — 全量数据导出包

生成时间：%s
包含回收站数据：%s

目录结构：
  manifest.json       导出清单（版本、作品、计数、文件列表）
  checksums.sha256    文件校验和
  data/app.dump       PostgreSQL 自定义格式数据库转储
  images/             人物图片与缩略图

恢复步骤（恢复期间必须停止应用）：
  docker compose stop app
  docker compose run --rm cli restore /data/exports/%s --yes
  docker compose start app

注意：恢复会自动创建当前数据快照；失败不会覆盖现有数据。
`, created.Format("2006-01-02 15:04:05"), trash, fileName)
}

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/backup"
	"novel-management/backend/internal/config"
	"novel-management/backend/internal/imports"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

const maxMarkdownBytes = 20 << 20

var (
	// ErrImportInProgress 表示已有未结束的导入任务。
	ErrImportInProgress = errors.New("已有导入任务正在进行")
	// ErrImportStateConflict 表示当前状态不允许该操作。
	ErrImportStateConflict = errors.New("导入状态不允许该操作")
	// ErrImportInvalid 表示导入校验未通过。
	ErrImportInvalid = errors.New("导入校验未通过")
	// ErrImportInsufficientStorage 表示磁盘空间不足。
	ErrImportInsufficientStorage = errors.New("磁盘空间不足")
	// ErrImportTimeout 表示导入处理超时。
	ErrImportTimeout = errors.New("导入超时")
	// ErrImportSnapshotFailed 表示导入前快照失败。
	ErrImportSnapshotFailed = errors.New("导入前快照失败")
)

// Import 编排模板上传、校验与原子提交。
type Import struct {
	db        *gorm.DB
	cfg       config.Config
	log       *slog.Logger
	snapshots *Snapshot
	now       func() time.Time
}

func NewImport(db *gorm.DB, cfg config.Config, log *slog.Logger, snapshots *Snapshot) *Import {
	return &Import{db: db, cfg: cfg, log: log, snapshots: snapshots, now: time.Now}
}

// CreateUpload 流式保存 Markdown 与可选图片 zip，并创建 uploaded 记录。
func (s *Import) CreateUpload(ctx context.Context, markdownName string, markdown io.Reader, markdownSize int64, archiveName string, archive io.Reader, archiveSize int64) (*model.ImportRun, error) {
	if !strings.EqualFold(filepath.Ext(markdownName), ".md") {
		return nil, &model.ValidationError{Field: "markdown", Message: "请上传固定模板 Markdown（.md）"}
	}
	if markdownSize > maxMarkdownBytes {
		return nil, &model.ValidationError{Field: "markdown", Message: "Markdown 不能超过 20MB"}
	}
	if archiveName != "" && !strings.EqualFold(filepath.Ext(archiveName), ".zip") {
		return nil, &model.ValidationError{Field: "archive", Message: "图片包必须是 .zip"}
	}
	maxArchive := int64(s.cfg.MaxUploadMB) << 20
	if archiveSize > maxArchive {
		return nil, &model.ValidationError{Field: "archive", Message: "图片 zip 不能超过 6GB"}
	}
	if existing, err := repository.NewImports(s.db).Active(ctx); err == nil && existing != nil {
		return nil, ErrImportInProgress
	}

	if available, err := backup.AvailableBytes(s.cfg.DataDir); err == nil && available > 0 && archiveSize > 0 {
		required := uint64(archiveSize)*3 + 1<<30
		if available < required {
			return nil, ErrImportInsufficientStorage
		}
	}

	id := model.NewID()
	dir := s.stagingDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建导入临时目录失败：%w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	markdownPath := filepath.Join(dir, "source.md")
	written, err := writeStream(markdownPath, markdown, maxMarkdownBytes)
	if err != nil {
		cleanup()
		if errors.Is(err, errTooLarge) {
			return nil, &model.ValidationError{Field: "markdown", Message: "Markdown 不能超过 20MB"}
		}
		return nil, fmt.Errorf("保存 Markdown 失败：%w", err)
	}

	now := s.now().UTC()
	run := &model.ImportRun{
		ID:            id,
		Status:        model.ImportStatusUploaded,
		MarkdownName:  filepath.Base(markdownName),
		MarkdownPath:  filepath.ToSlash(filepath.Join("tmp", "imports", id, "source.md")),
		MarkdownBytes: written,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if archiveName != "" {
		archivePath := filepath.Join(dir, "images.zip")
		archived, err := writeStream(archivePath, archive, maxArchive)
		if err != nil {
			cleanup()
			if errors.Is(err, errTooLarge) {
				return nil, &model.ValidationError{Field: "archive", Message: "图片 zip 不能超过 6GB"}
			}
			return nil, fmt.Errorf("保存图片 zip 失败：%w", err)
		}
		inventory, err := imports.InventoryZip(archivePath)
		if err != nil {
			cleanup()
			return nil, &model.ValidationError{Field: "archive", Message: err.Error()}
		}
		var uncompressed int64
		for _, file := range inventory.Files {
			uncompressed += file.Size
		}
		if available, err := backup.AvailableBytes(s.cfg.DataDir); err == nil && available > 0 && available < uint64(uncompressed)+1<<30 {
			cleanup()
			return nil, ErrImportInsufficientStorage
		}
		name := filepath.Base(archiveName)
		rel := filepath.ToSlash(filepath.Join("tmp", "imports", id, "images.zip"))
		run.ArchiveName = &name
		run.ArchivePath = &rel
		run.ArchiveBytes = &archived
	}

	if err := repository.NewImports(s.db).Insert(ctx, run); err != nil {
		cleanup()
		return nil, err
	}
	return run, nil
}

var errTooLarge = errors.New("文件超过大小限制")

func writeStream(path string, reader io.Reader, limit int64) (int64, error) {
	file, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()
	written, err := io.Copy(file, io.LimitReader(reader, limit+1))
	if err != nil {
		return written, err
	}
	if written > limit {
		return written, errTooLarge
	}
	return written, nil
}

// RunValidate 解析并校验上传文件；系统错误置 failed，校验问题保留 validated + error_count。
func (s *Import) RunValidate(ctx context.Context, id string) error {
	run, err := repository.NewImports(s.db).Get(ctx, id)
	if err != nil {
		return model.ErrNotFound
	}
	switch run.Status {
	case model.ImportStatusCompleted, model.ImportStatusRunning:
		return ErrImportStateConflict
	case model.ImportStatusValidating:
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.ImportTimeoutMin)*time.Minute)
	defer cancel()

	now := s.now().UTC()
	_ = repository.NewImports(s.db).Update(ctx, id, map[string]any{
		"status": model.ImportStatusValidating, "stage": model.ImportStageParse, "updated_at": now,
	})

	result, spec, issues, err := s.validateRun(ctx, run)
	if err != nil {
		s.failRun(ctx, id, model.ImportErrorFailed, err.Error())
		return err
	}
	warnings := result.Warnings
	if conflict, err := s.workNameConflict(ctx, spec.Work.Name); err == nil && conflict {
		issues = append(issues, imports.Issue{Line: spec.Work.Line, Field: "作品信息/名称", Code: imports.CodeDuplicate, Message: "作品名称已存在，请修改模板中的作品名称"})
	}
	_ = spec

	templateVersion := spec.TemplateVersion
	workName := strings.TrimSpace(spec.Work.Name)
	fields := map[string]any{
		"status":           model.ImportStatusValidated,
		"stage":            model.ImportStageValidate,
		"template_version": templateVersion,
		"work_name":        workName,
		"counts_json":      marshalString(result.Counts),
		"errors_json":      marshalString(issues),
		"warnings_json":    marshalString(warnings),
		"images_json":      marshalString(result.Images),
		"error_count":      len(issues),
		"error_code":       nil,
		"updated_at":       s.now().UTC(),
	}
	if err := repository.NewImports(s.db).Update(ctx, id, fields); err != nil {
		return err
	}
	return nil
}

func (s *Import) validateRun(ctx context.Context, run *model.ImportRun) (imports.ValidationResult, *imports.Spec, []imports.Issue, error) {
	content, err := os.ReadFile(s.absolute(run.MarkdownPath))
	if err != nil {
		return imports.ValidationResult{}, nil, nil, fmt.Errorf("读取模板失败：%w", err)
	}
	spec, parseIssues := imports.Parse(content)
	var inventory *imports.Inventory
	if run.ArchivePath != nil && *run.ArchivePath != "" {
		inventory, err = imports.InventoryZip(s.absolute(*run.ArchivePath))
		if err != nil {
			return imports.ValidationResult{}, nil, nil, fmt.Errorf("读取图片 zip 失败：%w", err)
		}
	}
	result := imports.Validate(spec, inventory)
	issues := make([]imports.Issue, 0, len(parseIssues)+len(result.Errors))
	issues = append(issues, parseIssues...)
	issues = append(issues, result.Errors...)
	_ = ctx
	return result, spec, issues, nil
}

func (s *Import) workNameConflict(ctx context.Context, raw string) (bool, error) {
	name, nameKey, err := model.NormalizeName(raw)
	if err != nil {
		return false, nil
	}
	_ = name
	return repository.NewWorks(s.db).NameTaken(ctx, nameKey)
}

// Get 返回导入记录。
func (s *Import) Get(ctx context.Context, id string) (*model.ImportRun, error) {
	run, err := repository.NewImports(s.db).Get(ctx, id)
	if err != nil {
		return nil, model.ErrNotFound
	}
	return run, nil
}

// Detail 把记录转换为 API 输出。
func Detail(run *model.ImportRun) *model.ImportDetail {
	detail := &model.ImportDetail{
		ImportID:        run.ID,
		Status:          run.Status,
		Stage:           run.Stage,
		FileName:        run.MarkdownName,
		ArchiveName:     run.ArchiveName,
		TemplateVersion: run.TemplateVersion,
		WorkName:        run.WorkName,
		WorkID:          run.WorkID,
		SnapshotID:      run.SnapshotID,
		ErrorCount:      run.ErrorCount,
		ErrorCode:       run.ErrorCode,
		CreatedAt:       run.CreatedAt,
		FinishedAt:      run.FinishedAt,
		Errors:          []model.ImportIssue{},
		Warnings:        []model.ImportIssue{},
	}
	if run.CountsJSON != nil {
		_ = json.Unmarshal([]byte(*run.CountsJSON), &detail.Counts)
	}
	if run.ErrorsJSON != nil {
		_ = json.Unmarshal([]byte(*run.ErrorsJSON), &detail.Errors)
	}
	if run.WarningsJSON != nil {
		_ = json.Unmarshal([]byte(*run.WarningsJSON), &detail.Warnings)
	}
	if run.ImagesJSON != nil {
		_ = json.Unmarshal([]byte(*run.ImagesJSON), &detail.Images)
	}
	return detail
}

// Report 返回导入报告；未提交时按校验结果生成。
func (s *Import) Report(ctx context.Context, id string) (*model.ImportReport, error) {
	run, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run.ReportJSON != nil {
		var report model.ImportReport
		if err := json.Unmarshal([]byte(*run.ReportJSON), &report); err == nil {
			return &report, nil
		}
	}
	detail := Detail(run)
	elapsed := int64(0)
	if run.FinishedAt != nil {
		elapsed = run.FinishedAt.Sub(run.CreatedAt).Milliseconds()
	}
	templateVersion := ""
	if run.TemplateVersion != nil {
		templateVersion = *run.TemplateVersion
	}
	return &model.ImportReport{
		ImportID:        run.ID,
		Status:          run.Status,
		FileName:        run.MarkdownName,
		ArchiveName:     run.ArchiveName,
		TemplateVersion: templateVersion,
		WorkID:          run.WorkID,
		WorkName:        run.WorkName,
		SnapshotID:      run.SnapshotID,
		Counts:          detail.Counts,
		Images:          detail.Images,
		Errors:          detail.Errors,
		Warnings:        detail.Warnings,
		ErrorCount:      run.ErrorCount,
		ElapsedMS:       elapsed,
		CreatedAt:       run.CreatedAt,
		FinishedAt:      run.FinishedAt,
	}, nil
}

// Commit 在快照后以单事务写入新作品；重复提交返回同一报告。
func (s *Import) Commit(ctx context.Context, id, workName string) (*model.ImportReport, error) {
	run, err := repository.NewImports(s.db).GetForUpdate(ctx, id)
	if err != nil {
		return nil, model.ErrNotFound
	}
	if run.Status == model.ImportStatusCompleted {
		return s.Report(ctx, id)
	}
	if run.Status != model.ImportStatusValidated {
		return nil, ErrImportStateConflict
	}
	if run.ErrorCount > 0 {
		return nil, ErrImportInvalid
	}
	if active, err := repository.NewImports(s.db).Active(ctx); err == nil && active != nil && active.ID != id {
		return nil, ErrImportInProgress
	}

	name, nameKey, err := model.NormalizeName(firstNonEmpty(workName, deref(run.WorkName)))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.ImportTimeoutMin)*time.Minute)
	defer cancel()

	now := s.now().UTC()
	_ = repository.NewImports(s.db).Update(ctx, id, map[string]any{
		"status": model.ImportStatusRunning, "stage": model.ImportStageImages, "updated_at": now,
	})

	snapshot, err := s.snapshots.Create(ctx, model.SnapshotReasonImport)
	if err != nil {
		s.failRun(ctx, id, "SNAPSHOT_FAILED", "导入前快照失败，请稍后重试")
		return nil, ErrImportSnapshotFailed
	}

	spec, inventory, result, issues, err := s.revalidate(ctx, run)
	if err != nil {
		s.failRun(ctx, id, model.ImportErrorFailed, err.Error())
		return nil, err
	}
	if len(issues) > 0 {
		s.failRun(ctx, id, model.ImportErrorInvalid, "校验结果已变化，请重新校验")
		return nil, ErrImportStateConflict
	}

	workID := model.NewID()
	media, err := s.prepareImages(workID, spec, inventory, run)
	if err != nil {
		s.failRun(ctx, id, model.ImportErrorFailed, err.Error())
		return nil, err
	}

	plan := buildImportPlan(workID, name, nameKey, spec, media, now)
	plan.counts = result.Counts
	plan.images = result.Images
	plan.warnings = toModelIssues(result.Warnings)

	finished := s.now().UTC()
	report := &model.ImportReport{
		ImportID:        id,
		Status:          model.ImportStatusCompleted,
		FileName:        run.MarkdownName,
		ArchiveName:     run.ArchiveName,
		TemplateVersion: spec.TemplateVersion,
		WorkID:          &workID,
		WorkName:        &name,
		SnapshotID:      &snapshot.ID,
		Counts:          plan.counts,
		Images:          plan.images,
		Errors:          []model.ImportIssue{},
		Warnings:        plan.warnings,
		ErrorCount:      0,
		ElapsedMS:       finished.Sub(run.CreatedAt).Milliseconds(),
		CreatedAt:       run.CreatedAt,
		FinishedAt:      &finished,
	}
	reportJSON, _ := json.Marshal(report)
	if err := s.writePlan(ctx, plan, id, snapshot.ID, string(reportJSON), finished); err != nil {
		s.removeImages(workID)
		s.failRun(ctx, id, model.ImportErrorFailed, err.Error())
		return nil, err
	}

	s.cleanupStaging(id)
	return report, nil
}

func (s *Import) revalidate(ctx context.Context, run *model.ImportRun) (*imports.Spec, *imports.Inventory, imports.ValidationResult, []imports.Issue, error) {
	result, spec, issues, err := s.validateRun(ctx, run)
	if err != nil {
		return nil, nil, imports.ValidationResult{}, nil, err
	}
	var inventory *imports.Inventory
	if run.ArchivePath != nil && *run.ArchivePath != "" {
		inventory, err = imports.InventoryZip(s.absolute(*run.ArchivePath))
		if err != nil {
			return nil, nil, imports.ValidationResult{}, nil, err
		}
	}
	if len(result.Errors) > 0 && len(issues) == 0 {
		issues = result.Errors
	}
	return spec, inventory, result, issues, nil
}

// MarkInterrupted 清理进程重启后遗留的进行中任务。
func (s *Import) MarkInterrupted(ctx context.Context) {
	if err := repository.NewImports(s.db).MarkInterrupted(ctx, s.now().UTC()); err != nil {
		s.log.Error("标记中断导入失败", "error", err)
	}
}

func (s *Import) stagingDir(id string) string {
	return filepath.Join(s.cfg.DataDir, "tmp", "imports", id)
}

func (s *Import) absolute(rel string) string {
	return filepath.Join(s.cfg.DataDir, filepath.FromSlash(rel))
}

func (s *Import) cleanupStaging(id string) {
	if err := os.RemoveAll(s.stagingDir(id)); err != nil {
		s.log.Warn("清理导入临时文件失败", "import_id", id, "error", err)
	}
}

func (s *Import) failRun(ctx context.Context, id, code, message string) {
	ctx = context.WithoutCancel(ctx)
	now := s.now().UTC()
	issue := model.ImportIssue{Code: code, Message: message}
	fields := map[string]any{
		"status":      model.ImportStatusFailed,
		"error_code":  code,
		"errors_json": marshalString([]model.ImportIssue{issue}),
		"error_count": 1,
		"finished_at": now,
		"updated_at":  now,
	}
	if err := repository.NewImports(s.db).Update(ctx, id, fields); err != nil {
		s.log.Error("更新导入失败状态失败", "import_id", id, "error", err)
	}
}

func (s *Import) removeImages(workID string) {
	if err := os.RemoveAll(filepath.Join(s.cfg.DataDir, "images", workID)); err != nil {
		s.log.Warn("清理导入图片失败", "work_id", workID, "error", err)
	}
}

func (s *Import) prepareImages(workID string, spec *imports.Spec, inventory *imports.Inventory, run *model.ImportRun) (map[string]model.MediaAsset, error) {
	result := map[string]model.MediaAsset{}
	if run.ArchivePath == nil || *run.ArchivePath == "" {
		return result, nil
	}
	zipPath := s.absolute(*run.ArchivePath)
	for _, character := range spec.Characters {
		if strings.TrimSpace(character.Image) == "" {
			continue
		}
		name, err := imports.NormalizeImagePath(character.Image)
		if err != nil {
			return nil, err
		}
		if inventory != nil {
			if file, ok := inventory.Files[name]; !ok {
				return nil, fmt.Errorf("图片 %s 缺失", name)
			} else if file.Size > model.MaxImageBytes {
				return nil, fmt.Errorf("图片 %s 超过 10MB", name)
			}
		}
		raw, err := backup.ReadZipFile(zipPath, name, model.MaxImageBytes+1)
		if err != nil {
			return nil, fmt.Errorf("读取图片 %s 失败：%w", name, err)
		}
		original, thumb, err := encodeImage(raw)
		if err != nil {
			return nil, fmt.Errorf("图片 %s 无法处理：%w", name, err)
		}
		assetID := model.NewID()
		relOriginal := filepath.ToSlash(filepath.Join("images", workID, assetID+original.ext))
		relThumb := filepath.ToSlash(filepath.Join("images", workID, assetID+"_thumb"+thumb.ext))
		if err := os.MkdirAll(filepath.Join(s.cfg.DataDir, "images", workID), 0o755); err != nil {
			return nil, err
		}
		if err := writeAtomic(filepath.Join(s.cfg.DataDir, filepath.FromSlash(relOriginal)), original.data); err != nil {
			return nil, err
		}
		if err := writeAtomic(filepath.Join(s.cfg.DataDir, filepath.FromSlash(relThumb)), thumb.data); err != nil {
			return nil, err
		}
		sum := sha256.Sum256(original.data)
		width, height := original.width, original.height
		thumbPath := relThumb
		result[character.Name] = model.MediaAsset{
			ID:           assetID,
			WorkID:       workID,
			EntityType:   model.MediaEntityCharacter,
			OriginalPath: relOriginal,
			ThumbPath:    &thumbPath,
			Mime:         original.mime,
			ByteSize:     int64(len(original.data)),
			Width:        &width,
			Height:       &height,
			SHA256:       hex.EncodeToString(sum[:]),
			Version:      1,
		}
	}
	return result, nil
}

func marshalString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(data)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ptr(value string) *string { return &value }

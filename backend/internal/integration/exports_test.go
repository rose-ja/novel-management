//go:build integration

package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"novel-management/backend/internal/backup"
	"novel-management/backend/internal/migrations"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

func (h *harness) createMediaAsset(workID string) {
	h.t.Helper()
	content := []byte("fake-image")
	rel := filepath.ToSlash(filepath.Join("images", workID, "test.jpg"))
	target := filepath.Join(h.dataDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		h.t.Fatalf("准备图片目录失败：%v", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		h.t.Fatalf("写入图片失败：%v", err)
	}
	sum := sha256.Sum256(content)
	now := time.Now().UTC()
	asset := &model.MediaAsset{
		ID:           model.NewID(),
		WorkID:       workID,
		EntityType:   model.MediaEntityCharacter,
		EntityID:     workID,
		OriginalPath: rel,
		Mime:         "image/jpeg",
		ByteSize:     int64(len(content)),
		SHA256:       hex.EncodeToString(sum[:]),
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.tx.Create(asset).Error; err != nil {
		h.t.Fatalf("写入图片记录失败：%v", err)
	}
}

func (h *harness) runExport(includeTrash bool) *model.ExportRun {
	h.t.Helper()
	record, err := h.exports.Create(context.Background(), includeTrash)
	if err != nil {
		h.t.Fatalf("创建导出任务失败：%v", err)
	}
	h.exports.Run(context.Background(), record.ID)
	run, err := h.exports.Get(context.Background(), record.ID)
	if err != nil {
		h.t.Fatalf("读取导出任务失败：%v", err)
	}
	return run
}

func TestExportArchiveStructureAndChecksums(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("导出作品")
	h.createMediaAsset(work.ID)

	run := h.runExport(false)
	if run.Status != model.ExportStatusCompleted {
		t.Fatalf("导出应完成，实际 %s（%v）", run.Status, run.ErrorCode)
	}
	if run.FileName == nil || run.ByteSize == nil || *run.ByteSize == 0 {
		t.Fatalf("导出结果缺少文件名或大小：%+v", run)
	}

	zipPath := filepath.Join(h.dataDir, "exports", *run.FileName)
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("读取导出包失败：%v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("解析导出包失败：%v", err)
	}
	names := map[string]bool{}
	for _, file := range reader.File {
		names[file.Name] = true
	}
	for _, want := range []string{"manifest.json", "checksums.sha256", "README.txt", "data/app.dump"} {
		if !names[want] {
			t.Fatalf("zip 缺少 %s：%v", want, names)
		}
	}
	if !names["images/"+work.ID+"/test.jpg"] {
		t.Fatalf("zip 缺少图片：%v", names)
	}

	extracted := t.TempDir()
	if _, err := backup.ExtractZip(zipPath, extracted); err != nil {
		t.Fatalf("解压失败：%v", err)
	}
	if err := backup.VerifyChecksums(extracted, filepath.Join(extracted, backup.ChecksumFile)); err != nil {
		t.Fatalf("校验和验证失败：%v", err)
	}

	var manifest backup.ExportManifest
	raw, err := os.ReadFile(filepath.Join(extracted, "manifest.json"))
	if err != nil {
		t.Fatalf("读取 manifest 失败：%v", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("解析 manifest 失败：%v", err)
	}
	if manifest.ExportVersion != backup.ExportVersion || manifest.SchemaVersion != migrations.LatestVersion {
		t.Fatalf("manifest 版本不符：%+v", manifest)
	}
	if manifest.IncludesTrash {
		t.Fatal("include_trash=false 时 manifest 应标记为 false")
	}
	if manifest.Counts["works"] != 1 || manifest.Images.Count != 1 {
		t.Fatalf("manifest 计数不符：%+v，images=%+v", manifest.Counts, manifest.Images)
	}
	if len(manifest.Works) != 1 || manifest.Works[0].ID != work.ID {
		t.Fatalf("manifest 作品不符：%+v", manifest.Works)
	}
}

func TestExportRejectsNoWorkAndInProgress(t *testing.T) {
	h := newHarness(t)

	rec := h.do(http.MethodPost, "/api/v1/exports", map[string]any{}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "EXPORT_NO_WORK")

	h.createWork("导出作品")
	if _, err := h.exports.Create(context.Background(), true); err != nil {
		t.Fatalf("首次创建导出任务失败：%v", err)
	}
	if _, err := h.exports.Create(context.Background(), true); !errors.Is(err, service.ErrExportInProgress) {
		t.Fatalf("并发导出应返回 ErrExportInProgress，实际 %v", err)
	}

	rec = h.do(http.MethodPost, "/api/v1/exports", map[string]any{}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "EXPORT_IN_PROGRESS")
}

func TestExportFailureRecordsErrorCode(t *testing.T) {
	h := newHarness(t)
	h.createWork("失败作品")
	h.dumper.fail = true

	record, err := h.exports.Create(context.Background(), true)
	if err != nil {
		t.Fatalf("创建导出任务失败：%v", err)
	}
	h.exports.Run(context.Background(), record.ID)

	run, err := h.exports.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("读取导出任务失败：%v", err)
	}
	if run.Status != model.ExportStatusFailed || run.ErrorCode == nil || *run.ErrorCode != model.ExportErrorFailed {
		t.Fatalf("失败任务状态不符：%+v", run)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "exports", *run.FileName)); !os.IsNotExist(err) {
		t.Fatal("失败导出不应留下 zip")
	}
}

func TestExportReadEndpoints(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("端点作品")
	h.createMediaAsset(work.ID)
	run := h.runExport(true)

	rec := h.do(http.MethodGet, "/api/v1/exports", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("列表失败：%d %s", rec.Code, rec.Body.String())
	}
	var items []model.ExportRun
	if err := decodeJSON(t, rec, &items); err != nil {
		t.Fatalf("解析列表失败：%v", err)
	}
	if len(items) == 0 || items[0].ID != run.ID {
		t.Fatalf("列表未包含刚完成的导出：%+v", items)
	}

	rec = h.do(http.MethodGet, "/api/v1/exports/"+run.ID, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("详情失败：%d %s", rec.Code, rec.Body.String())
	}
	var detail model.ExportRun
	if err := decodeJSON(t, rec, &detail); err != nil || detail.ID != run.ID {
		t.Fatalf("详情不符：%v %+v", err, detail)
	}

	rec = h.do(http.MethodGet, "/api/v1/exports/"+run.ID+"/download", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("下载失败：%d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("下载类型不符：%s", rec.Header().Get("Content-Type"))
	}
	if body := rec.Body.Bytes(); len(body) == 0 || !bytes.HasPrefix(body, []byte("PK")) {
		t.Fatal("下载内容不是 zip")
	}

	rec = h.do(http.MethodGet, "/api/v1/export/status", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态失败：%d %s", rec.Code, rec.Body.String())
	}
	var status service.ExportStatus
	if err := decodeJSON(t, rec, &status); err != nil {
		t.Fatalf("解析状态失败：%v", err)
	}
	if status.Active != nil || status.Last == nil || status.Last.ID != run.ID {
		t.Fatalf("状态聚合不符：%+v", status)
	}

	rec = h.do(http.MethodGet, "/api/v1/exports/not-exist", nil, nil)
	assertErrorCode(t, rec, http.StatusNotFound, "NOT_FOUND")
}

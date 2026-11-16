//go:build integration

package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"novel-management/backend/internal/imports"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

func importZip(t *testing.T, markdown []byte, withExampleImage bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	mdFile, err := writer.Create("import.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mdFile.Write(markdown); err != nil {
		t.Fatal(err)
	}
	if withExampleImage {
		image := exampleImageBytes(t)
		imageFile, err := writer.Create("images/example.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := imageFile.Write(image); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func exampleImageBytes(t *testing.T) []byte {
	t.Helper()
	data, err := imports.ExampleZip()
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if file.Name != "images/example.png" {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = opened.Close() }()
		content, err := io.ReadAll(opened)
		if err != nil {
			t.Fatal(err)
		}
		return content
	}
	t.Fatal("示例图片不存在")
	return nil
}

func createImport(t *testing.T, h *harness, markdown, archive []byte) *model.ImportRun {
	t.Helper()
	markdownName := "import.md"
	run, err := h.imports.CreateUpload(context.Background(), markdownName, bytes.NewReader(markdown), int64(len(markdown)), "images.zip", bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("创建导入失败：%v", err)
	}
	return run
}

func TestImportHappyPathAndReview(t *testing.T) {
	h := newHarness(t)
	markdown := imports.ExampleMarkdown()
	archive := importZip(t, markdown, true)
	run := createImport(t, h, markdown, archive)

	if run.Status != model.ImportStatusUploaded {
		t.Fatalf("上传状态=%s", run.Status)
	}
	if err := h.imports.RunValidate(context.Background(), run.ID); err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	validated, err := h.imports.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if validated.Status != model.ImportStatusValidated || validated.ErrorCount != 0 {
		t.Fatalf("校验状态=%s errors=%d，详情=%+v", validated.Status, validated.ErrorCount, service.Detail(validated))
	}
	detail := service.Detail(validated)
	if detail.Counts.Characters != 1 || detail.Counts.SystemNodes != 3 || detail.Images.Matched != 1 {
		t.Fatalf("计数不符：%+v images=%+v", detail.Counts, detail.Images)
	}

	report, err := h.imports.Commit(context.Background(), run.ID, "")
	if err != nil {
		t.Fatalf("提交失败：%v", err)
	}
	if report.WorkID == nil || *report.WorkID == "" {
		t.Fatal("缺少 work_id")
	}
	workID := *report.WorkID

	var characters, factions, locations, rankings, nodes, chronicles, media int64
	for table, target := range map[string]*int64{
		"characters": &characters, "factions": &factions, "locations": &locations,
		"rankings": &rankings, "system_nodes": &nodes, "chronicles": &chronicles, "media_assets": &media,
	} {
		if err := h.tx.Table(table).Where("work_id = ?", workID).Count(target).Error; err != nil {
			t.Fatalf("统计 %s 失败：%v", table, err)
		}
	}
	if characters != 1 || factions != 1 || locations != 1 || rankings != 1 || nodes != 3 || chronicles != 1 || media != 1 {
		t.Fatalf("写入计数不符：characters=%d factions=%d locations=%d rankings=%d nodes=%d chronicles=%d media=%d",
			characters, factions, locations, rankings, nodes, chronicles, media)
	}
	if current := h.currentWorkID(); current == nil || *current != workID {
		t.Fatalf("当前作品未切换：%+v", current)
	}

	second, err := h.imports.Commit(context.Background(), run.ID, "")
	if err != nil {
		t.Fatalf("重复提交失败：%v", err)
	}
	if second.WorkID == nil || *second.WorkID != workID {
		t.Fatalf("重复提交应返回同一作品：%+v", second.WorkID)
	}

	var characterID string
	if err := h.tx.Table("characters").Select("id").Where("work_id = ?", workID).Scan(&characterID).Error; err != nil {
		t.Fatal(err)
	}
	rec := h.do(http.MethodPost, "/api/v1/characters/"+characterID+"/review", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("单条校对失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	result := decodeBody[model.ReviewResult](t, rec)
	if result.ReviewStatus != model.ReviewStatusAPIConfirmed {
		t.Fatalf("校对状态=%s", result.ReviewStatus)
	}
	version := result.Version
	rec = h.do(http.MethodPost, "/api/v1/characters/"+characterID+"/review", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("重复校对失败：%s", rec.Body.String())
	}
	again := decodeBody[model.ReviewResult](t, rec)
	if again.Version != version {
		t.Fatalf("重复校对不应递增版本：%d → %d", version, again.Version)
	}

	var factionID string
	if err := h.tx.Table("factions").Select("id").Where("work_id = ?", workID).Scan(&factionID).Error; err != nil {
		t.Fatal(err)
	}
	batch := map[string]any{"entity_type": "factions", "ids": []string{factionID}}
	rec = h.do(http.MethodPost, "/api/v1/reviews/batch", batch, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("批量校对失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	batchResult := decodeBody[model.ReviewBatchResult](t, rec)
	if batchResult.Updated != 1 {
		t.Fatalf("批量更新=%d", batchResult.Updated)
	}

	rec = h.do(http.MethodPost, "/api/v1/reviews/batch", map[string]any{
		"entity_type": "factions", "ids": []string{"missing"},
	}, nil)
	assertErrorCode(t, rec, http.StatusNotFound, "REVIEW_NOT_FOUND")

	rec = h.do(http.MethodGet, "/api/v1/imports/"+run.ID, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("读取导入详情失败：%s", rec.Body.String())
	}
	detailPayload := decodeBody[model.ImportDetail](t, rec)
	if detailPayload.Status != model.ImportStatusCompleted || detailPayload.WorkID == nil || *detailPayload.WorkID != workID {
		t.Fatalf("导入详情不符：%+v", detailPayload)
	}
}

func TestImportRejectsMissingImage(t *testing.T) {
	h := newHarness(t)
	markdown := []byte(strings.Replace(string(imports.ExampleMarkdown()), "images/example.png", "images/missing.png", 1))
	archive := importZip(t, markdown, true)
	run := createImport(t, h, markdown, archive)
	if err := h.imports.RunValidate(context.Background(), run.ID); err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	validated, err := h.imports.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if validated.ErrorCount == 0 {
		t.Fatal("缺失图片应进入错误清单")
	}
	detail := service.Detail(validated)
	if len(detail.Images.Missing) != 1 || len(detail.Images.Extra) != 1 {
		t.Fatalf("图片摘要不符：%+v", detail.Images)
	}
	if _, err := h.imports.Commit(context.Background(), run.ID, ""); !errors.Is(err, service.ErrImportInvalid) {
		t.Fatalf("存在错误时提交应被拒绝，实际 %v", err)
	}
	var works int64
	if err := h.tx.Table("works").Count(&works).Error; err != nil {
		t.Fatal(err)
	}
	if works != 0 {
		t.Fatalf("提交被拒绝后不应创建作品，实际 %d", works)
	}
}

func TestImportSnapshotFailureRollsBack(t *testing.T) {
	h := newHarness(t)
	markdown := imports.ExampleMarkdown()
	archive := importZip(t, markdown, true)
	run := createImport(t, h, markdown, archive)
	if err := h.imports.RunValidate(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	h.dumper.fail = true
	if _, err := h.imports.Commit(context.Background(), run.ID, ""); !errors.Is(err, service.ErrImportSnapshotFailed) {
		t.Fatalf("快照失败应阻止导入，实际 %v", err)
	}
	failed, err := h.imports.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != model.ImportStatusFailed || failed.ErrorCode == nil || *failed.ErrorCode != "SNAPSHOT_FAILED" {
		t.Fatalf("失败状态不符：%+v", failed)
	}
	var works int64
	if err := h.tx.Table("works").Count(&works).Error; err != nil {
		t.Fatal(err)
	}
	if works != 0 {
		t.Fatalf("快照失败后不应创建作品，实际 %d", works)
	}
}

func TestImportTemplateDownload(t *testing.T) {
	h := newHarness(t)
	rec := h.do(http.MethodGet, "/api/v1/imports/template", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("模板下载失败（status=%d）", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "模板版本：1.0") {
		t.Fatal("模板内容不符")
	}
	rec = h.do(http.MethodGet, "/api/v1/imports/template/example", nil, nil)
	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("示例下载失败（status=%d）", rec.Code)
	}
	path := filepath.Join(t.TempDir(), "example.zip")
	if err := os.WriteFile(path, rec.Body.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("示例不是有效 zip：%v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}

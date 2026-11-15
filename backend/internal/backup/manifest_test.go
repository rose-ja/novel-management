package backup

import (
	"path/filepath"
	"testing"
)

func TestReadExportManifestValidatesVersion(t *testing.T) {
	root := t.TempDir()
	WriteJSON(filepath.Join(root, "manifest.json"), ExportManifest{
		ExportVersion: ExportVersion,
		SchemaVersion: 10,
		Counts:        map[string]int64{"works": 1},
	})

	manifest, err := ReadExportManifest(root)
	if err != nil {
		t.Fatalf("读取清单失败：%v", err)
	}
	if err := manifest.ValidateSchema(10); err != nil {
		t.Fatalf("同版本应兼容：%v", err)
	}
	if err := manifest.ValidateSchema(11); err != nil {
		t.Fatalf("低版本应兼容：%v", err)
	}
	if err := manifest.ValidateSchema(9); err == nil {
		t.Fatal("高版本应拒绝")
	}

	WriteJSON(filepath.Join(root, "manifest.json"), map[string]any{"export_version": 2, "schema_version": 10})
	if _, err := ReadExportManifest(root); err == nil {
		t.Fatal("导出版本不兼容应拒绝")
	}

	WriteJSON(filepath.Join(root, "manifest.json"), map[string]any{"export_version": 1})
	if _, err := ReadExportManifest(root); err == nil {
		t.Fatal("缺少 schema_version 应拒绝")
	}
}

func TestSnapshotManifestLegacyWithoutSchemaVersion(t *testing.T) {
	root := t.TempDir()
	WriteJSON(filepath.Join(root, "manifest.json"), map[string]any{
		"created_at": "2026-11-15T10:00:00Z",
		"files":      []ManifestFile{{Path: "app.dump", Size: 4}},
	})

	manifest, legacy, err := ReadSnapshotManifest(root)
	if err != nil {
		t.Fatalf("读取快照清单失败：%v", err)
	}
	if !legacy {
		t.Fatal("缺少 schema_version 应标记为旧快照")
	}
	if err := manifest.ValidateSchema(10); err != nil {
		t.Fatalf("旧快照应按兼容处理：%v", err)
	}
}

func TestSnapshotManifestRejectsNewerSchema(t *testing.T) {
	root := t.TempDir()
	WriteJSON(filepath.Join(root, "manifest.json"), SnapshotManifest{
		CreatedAt:     "2026-11-15T10:00:00Z",
		SchemaVersion: 11,
	})

	manifest, legacy, err := ReadSnapshotManifest(root)
	if err != nil {
		t.Fatalf("读取快照清单失败：%v", err)
	}
	if legacy {
		t.Fatal("有 schema_version 不应标记为旧快照")
	}
	if err := manifest.ValidateSchema(10); err == nil {
		t.Fatal("高版本快照应拒绝")
	}
}

package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ExportVersion 是导出包格式版本，恢复时强校验。
const ExportVersion = 1

// ManifestWork 是导出时的活跃作品摘要。
type ManifestWork struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ManifestFile 记录包内文件大小。
type ManifestFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ManifestImages 记录图片数量与相对路径。
type ManifestImages struct {
	Count int      `json:"count"`
	Files []string `json:"files"`
}

// ExportManifest 是导出包 manifest.json 的结构。
type ExportManifest struct {
	ExportVersion int              `json:"export_version"`
	SchemaVersion int              `json:"schema_version"`
	AppVersion    string           `json:"app_version"`
	GeneratedAt   string           `json:"generated_at"`
	IncludesTrash bool             `json:"includes_trash"`
	Works         []ManifestWork   `json:"works"`
	Counts        map[string]int64 `json:"counts"`
	Images        ManifestImages   `json:"images"`
	Files         []ManifestFile   `json:"files"`
}

// SnapshotManifest 是快照目录 manifest.json 的结构；旧快照缺少版本字段。
type SnapshotManifest struct {
	CreatedAt     string         `json:"created_at"`
	SchemaVersion int            `json:"schema_version"`
	AppVersion    string         `json:"app_version"`
	Files         []ManifestFile `json:"files"`
}

// ReadExportManifest 读取并校验导出包 manifest 的基本字段。
func ReadExportManifest(root string) (ExportManifest, error) {
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return ExportManifest{}, fmt.Errorf("读取导出清单失败：%w", err)
	}
	return ParseExportManifest(data)
}

// ParseExportManifest 解析并校验导出清单，用于恢复前从 zip 内直接读取。
func ParseExportManifest(data []byte) (ExportManifest, error) {
	var manifest ExportManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("读取导出清单失败：%w", err)
	}
	if manifest.ExportVersion != ExportVersion {
		return manifest, fmt.Errorf("导出包版本不兼容：%d（当前支持 %d）", manifest.ExportVersion, ExportVersion)
	}
	if manifest.SchemaVersion <= 0 {
		return manifest, fmt.Errorf("导出包缺少 schema_version")
	}
	if manifest.Counts == nil {
		manifest.Counts = map[string]int64{}
	}
	return manifest, nil
}

// ValidateSchema 校验导出包的数据版本不高于当前二进制支持的版本。
func (m ExportManifest) ValidateSchema(latest int) error {
	if m.SchemaVersion > latest {
		return fmt.Errorf("导出包数据版本 %d 高于当前系统支持的 %d，请先升级系统", m.SchemaVersion, latest)
	}
	return nil
}

// ReadSnapshotManifest 读取快照 manifest；legacy 表示缺少 schema_version 的旧快照。
func ReadSnapshotManifest(root string) (manifest SnapshotManifest, legacy bool, err error) {
	if err := ReadJSON(filepath.Join(root, "manifest.json"), &manifest); err != nil {
		return manifest, false, fmt.Errorf("读取快照清单失败：%w", err)
	}
	if manifest.CreatedAt == "" {
		return manifest, false, fmt.Errorf("快照清单缺少 created_at")
	}
	return manifest, manifest.SchemaVersion == 0, nil
}

// ValidateSchema 校验快照数据版本；旧快照（schema=0）按兼容处理。
func (m SnapshotManifest) ValidateSchema(latest int) error {
	if m.SchemaVersion > latest {
		return fmt.Errorf("快照数据版本 %d 高于当前系统支持的 %d，请先升级系统", m.SchemaVersion, latest)
	}
	return nil
}

// ReadJSON 读取 JSON 文件。
func ReadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// WriteJSON 写入带缩进的 JSON 文件。
func WriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

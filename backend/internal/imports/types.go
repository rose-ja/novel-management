// Package imports 解析与校验飞书迁移模板 v1.0。
package imports

import "novel-management/backend/internal/model"

// TemplateVersion 是当前支持的模板版本。
const TemplateVersion = "1.0"

// 校验错误码；对外映射为 IMPORT_INVALID 等稳定错误。
const (
	CodeInvalid          = "IMPORT_INVALID"
	CodeTemplateVersion  = "IMPORT_TEMPLATE_VERSION"
	CodeFieldRequired    = "IMPORT_FIELD_REQUIRED"
	CodeUnknownSection   = "IMPORT_UNKNOWN_SECTION"
	CodeUnknownField     = "IMPORT_UNKNOWN_FIELD"
	CodeDuplicate        = "IMPORT_DUPLICATE"
	CodeReferenceMissing = "IMPORT_REFERENCE_MISSING"
	CodeCategoryLimit    = "IMPORT_CATEGORY_LIMIT"
	CodeNodeLevel        = "IMPORT_NODE_LEVEL"
	CodeNodeParent       = "IMPORT_NODE_PARENT"
	CodeImageMissing     = "IMPORT_IMAGE_MISSING"
	CodeImageExtra       = "IMPORT_IMAGE_EXTRA"
	CodeImageUnsupported = "IMPORT_IMAGE_UNSUPPORTED"
	CodeImageTooLarge    = "IMPORT_IMAGE_TOO_LARGE"
	CodePathTraversal    = "PATH_TRAVERSAL"
	CodeNoCategories     = "IMPORT_NO_CATEGORIES"
)

// Issue 是可定位的校验问题。
type Issue struct {
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Spec 是模板解析结果。
type Spec struct {
	TemplateVersion string
	Work            WorkSpec
	Categories      []CategorySpec
	Nodes           []NodeSpec
	Factions        []FactionSpec
	Locations       []LocationSpec
	Characters      []CharacterSpec
	Rankings        []RankingSpec
	Chronicles      []ChronicleSpec
}

type WorkSpec struct {
	Name        string
	Description string
	Line        int
}

type CategorySpec struct {
	Name    string
	Builtin bool
	Line    int
}

type NodeSpec struct {
	Category string
	Parent   string
	Name     string
	Line     int
}

type RelationSpec struct {
	Name string
	Note string
	Line int
}

type FactionSpec struct {
	Name        string
	Parent      string
	Leader      string
	Description string
	Line        int
}

type LocationSpec struct {
	Name        string
	Parent      string
	TypeText    string
	Description string
	Factions    []RelationSpec
	Line        int
}

type CharacterSpec struct {
	Name        string
	Aliases     []string
	Titles      []string
	Description string
	Faction     string
	Locations   []RelationSpec
	SystemRefs  map[string]string
	Extras      map[string]string
	Image       string
	Line        int
}

type SeatSpec struct {
	No        int
	Character string
	Note      string
	Line      int
}

type RankingSpec struct {
	Name      string
	SeatCount int
	Seats     []SeatSpec
	Line      int
}

type ChronicleSpec struct {
	Title     string
	OwnerType string
	OwnerName string
	TimeLabel string
	Text      string
	Line      int
}

// ImageFile 是 zip 内一个图片条目的元信息。
type ImageFile struct {
	Path string
	Size int64
	MIME string
}

// Inventory 是图片 zip 的清单。
type Inventory struct {
	Files map[string]ImageFile
}

// ValidationResult 是校验输出。
type ValidationResult struct {
	Counts   model.ImportCounts
	Images   model.ImportImageSummary
	Errors   []Issue
	Warnings []Issue
}

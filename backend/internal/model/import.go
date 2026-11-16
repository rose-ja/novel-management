package model

import "time"

// 导入任务状态，与迁移 CHECK 约束保持一致。
const (
	ImportStatusUploaded   = "uploaded"
	ImportStatusValidating = "validating"
	ImportStatusValidated  = "validated"
	ImportStatusRunning    = "running"
	ImportStatusCompleted  = "completed"
	ImportStatusFailed     = "failed"
)

// 导入阶段，供前端展示不确定进度。
const (
	ImportStageUpload    = "upload"
	ImportStageParse     = "parse"
	ImportStageValidate  = "validate"
	ImportStageImages    = "images"
	ImportStageWrite     = "write"
	ImportStageSearch    = "search"
	ImportStageCompleted = "completed"
)

// 导入错误码：对外暴露稳定标识，前端据此显示中文提示。
const (
	ImportErrorInProgress          = "IMPORT_IN_PROGRESS"
	ImportErrorStateConflict       = "IMPORT_STATE_CONFLICT"
	ImportErrorInvalid             = "IMPORT_INVALID"
	ImportErrorTemplateVersion     = "IMPORT_TEMPLATE_VERSION"
	ImportErrorInsufficientStorage = "IMPORT_INSUFFICIENT_STORAGE"
	ImportErrorTimeout             = "IMPORT_TIMEOUT"
	ImportErrorInterrupted         = "IMPORT_INTERRUPTED"
	ImportErrorFailed              = "IMPORT_FAILED"
	ImportErrorPathTraversal       = "PATH_TRAVERSAL"
)

// ImportReviewSource 是导入实体的校对来源标记。
const ImportReviewSource = "import"

// ImportRun 对应 import_runs，记录一次模板导入的上传、校验与提交状态。
type ImportRun struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	Status          string     `gorm:"column:status" json:"status"`
	Stage           *string    `gorm:"column:stage" json:"stage"`
	MarkdownName    string     `gorm:"column:markdown_name" json:"file_name"`
	ArchiveName     *string    `gorm:"column:archive_name" json:"archive_name"`
	MarkdownPath    string     `gorm:"column:markdown_path" json:"-"`
	ArchivePath     *string    `gorm:"column:archive_path" json:"-"`
	MarkdownBytes   int64      `gorm:"column:markdown_bytes" json:"markdown_bytes"`
	ArchiveBytes    *int64     `gorm:"column:archive_bytes" json:"archive_bytes"`
	TemplateVersion *string    `gorm:"column:template_version" json:"template_version"`
	WorkName        *string    `gorm:"column:work_name" json:"work_name"`
	WorkID          *string    `gorm:"column:work_id" json:"work_id"`
	SnapshotID      *string    `gorm:"column:snapshot_id" json:"snapshot_id"`
	CountsJSON      *string    `gorm:"column:counts_json" json:"-"`
	ErrorsJSON      *string    `gorm:"column:errors_json" json:"-"`
	WarningsJSON    *string    `gorm:"column:warnings_json" json:"-"`
	ReportJSON      *string    `gorm:"column:report_json" json:"-"`
	ErrorCount      int        `gorm:"column:error_count" json:"error_count"`
	ErrorCode       *string    `gorm:"column:error_code" json:"error_code"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updated_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

func (ImportRun) TableName() string { return "import_runs" }

// ImportCounts 是预览与报告中的实体计数。
type ImportCounts struct {
	Characters       int `json:"characters"`
	Factions         int `json:"factions"`
	Locations        int `json:"locations"`
	Rankings         int `json:"rankings"`
	SystemCategories int `json:"system_categories"`
	SystemNodes      int `json:"system_nodes"`
	Chronicles       int `json:"chronicles"`
	Images           int `json:"images"`
	Reviewable       int `json:"reviewable"`
}

// ImportIssue 是可定位到行/字段/图片的校验问题。
type ImportIssue struct {
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ImportImageSummary 是图片匹配摘要。
type ImportImageSummary struct {
	Matched     int      `json:"matched"`
	Missing     []string `json:"missing"`
	Extra       []string `json:"extra"`
	Unsupported []string `json:"unsupported"`
	TooLarge    []string `json:"too_large"`
}

// ImportReport 是导入报告；成功后随报告页展示并留存。
type ImportReport struct {
	ImportID        string             `json:"import_id"`
	Status          string             `json:"status"`
	FileName        string             `json:"file_name"`
	ArchiveName     *string            `json:"archive_name"`
	TemplateVersion string             `json:"template_version"`
	WorkID          *string            `json:"work_id"`
	WorkName        *string            `json:"work_name"`
	SnapshotID      *string            `json:"snapshot_id"`
	Counts          ImportCounts       `json:"counts"`
	Images          ImportImageSummary `json:"images"`
	Errors          []ImportIssue      `json:"errors"`
	Warnings        []ImportIssue      `json:"warnings"`
	ErrorCount      int                `json:"error_count"`
	ElapsedMS       int64              `json:"elapsed_ms"`
	CreatedAt       time.Time          `json:"created_at"`
	FinishedAt      *time.Time         `json:"finished_at"`
}

// ImportDetail 是 GET /imports/{id} 的响应。
type ImportDetail struct {
	ImportID        string             `json:"import_id"`
	Status          string             `json:"status"`
	Stage           *string            `json:"stage"`
	FileName        string             `json:"file_name"`
	ArchiveName     *string            `json:"archive_name"`
	TemplateVersion *string            `json:"template_version"`
	WorkName        *string            `json:"work_name"`
	WorkID          *string            `json:"work_id"`
	SnapshotID      *string            `json:"snapshot_id"`
	Counts          ImportCounts       `json:"counts"`
	Images          ImportImageSummary `json:"images"`
	Errors          []ImportIssue      `json:"errors"`
	Warnings        []ImportIssue      `json:"warnings"`
	ErrorCount      int                `json:"error_count"`
	ErrorCode       *string            `json:"error_code"`
	CreatedAt       time.Time          `json:"created_at"`
	FinishedAt      *time.Time         `json:"finished_at"`
}

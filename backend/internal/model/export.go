package model

import "time"

// 导出任务状态，与迁移 CHECK 约束保持一致。
const (
	ExportStatusQueued    = "queued"
	ExportStatusRunning   = "running"
	ExportStatusCompleted = "completed"
	ExportStatusFailed    = "failed"
)

// 导出错误码：对外暴露稳定标识，前端据此显示中文提示。
const (
	ExportErrorInProgress          = "EXPORT_IN_PROGRESS"
	ExportErrorNoWork              = "EXPORT_NO_WORK"
	ExportErrorInterrupted         = "EXPORT_INTERRUPTED"
	ExportErrorInsufficientStorage = "EXPORT_INSUFFICIENT_STORAGE"
	ExportErrorFailed              = "EXPORT_FAILED"
)

// ExportRun 对应 export_runs，记录一次全量导出的状态与产物。
type ExportRun struct {
	ID           string     `gorm:"column:id;primaryKey" json:"id"`
	Status       string     `gorm:"column:status" json:"status"`
	IncludeTrash bool       `gorm:"column:include_trash" json:"include_trash"`
	FileName     *string    `gorm:"column:file_name" json:"file_name"`
	FilePath     *string    `gorm:"column:file_path" json:"file_path"`
	ByteSize     *int64     `gorm:"column:byte_size" json:"byte_size"`
	ErrorCode    *string    `gorm:"column:error_code" json:"error_code"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

func (ExportRun) TableName() string { return "export_runs" }

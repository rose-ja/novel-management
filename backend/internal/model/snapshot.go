package model

import "time"

// 快照触发原因，与迁移 CHECK 约束保持一致。
const (
	SnapshotReasonWorkDelete = "work_delete"
	SnapshotReasonTrashPurge = "trash_purge"
	SnapshotReasonPurgeAll   = "purge_all"
	SnapshotReasonImport     = "import"
	SnapshotReasonCLIRestore = "cli_restore"
	SnapshotStatusOK         = "ok"
	SnapshotStatusFailed     = "failed"
	SnapshotKeepDefault      = 3
	UndoWindowSeconds        = 8
)

// Snapshot 对应 {DATA_DIR}/snapshots/{ts}-{reason} 目录与数据库记录。
type Snapshot struct {
	ID         string    `gorm:"column:id;primaryKey" json:"id"`
	Reason     string    `gorm:"column:reason" json:"reason"`
	Path       string    `gorm:"column:path" json:"path"`
	DBBytes    int64     `gorm:"column:db_bytes" json:"db_bytes"`
	ImageCount int       `gorm:"column:image_count" json:"image_count"`
	Status     string    `gorm:"column:status" json:"status"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Snapshot) TableName() string { return "snapshots" }

// UndoToken 是一次性撤销令牌，root_id 指向被软删除的根记录。
type UndoToken struct {
	Token     string     `gorm:"column:token;primaryKey" json:"token"`
	RootType  string     `gorm:"column:root_type" json:"root_type"`
	RootID    string     `gorm:"column:root_id" json:"root_id"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
	ExpiresAt time.Time  `gorm:"column:expires_at" json:"expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at,omitempty"`
}

func (UndoToken) TableName() string { return "undo_tokens" }

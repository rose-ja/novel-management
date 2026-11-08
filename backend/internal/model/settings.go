package model

import "time"

// AppSettings 是单行设置表（id 恒为 1）。
type AppSettings struct {
	ID            int16      `gorm:"column:id;primaryKey" json:"-"`
	CurrentWorkID *string    `gorm:"column:current_work_id" json:"current_work_id"`
	LastExportAt  *time.Time `gorm:"column:last_export_at" json:"last_export_at"`
	LastRestoreAt *time.Time `gorm:"column:last_restore_at" json:"last_restore_at"`
	SnapshotKeep  int16      `gorm:"column:snapshot_keep" json:"snapshot_keep"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"-"`
}

func (AppSettings) TableName() string { return "app_settings" }

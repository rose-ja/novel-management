package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
)

// Settings 读写单行 app_settings（id=1）。
type Settings struct {
	DB *gorm.DB
}

func NewSettings(db *gorm.DB) *Settings { return &Settings{DB: db} }

func (r *Settings) Get(ctx context.Context) (*model.AppSettings, error) {
	var row model.AppSettings
	err := r.DB.WithContext(ctx).Where("id = 1").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// SetCurrentWork 写入当前作品；nil 表示清空。
func (r *Settings) SetCurrentWork(ctx context.Context, workID *string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("app_settings").
		Where("id = 1").
		Updates(map[string]any{"current_work_id": workID, "updated_at": now}).Error
}

// ClearCurrentWorkIf 仅当当前作品为指定值时清空（删除/永久删除后调用）。
func (r *Settings) ClearCurrentWorkIf(ctx context.Context, workID string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("app_settings").
		Where("id = 1 AND current_work_id = ?", workID).
		Updates(map[string]any{"current_work_id": nil, "updated_at": now}).Error
}

package repository

import (
	"context"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
)

// Snapshots 是快照记录的数据访问层。
type Snapshots struct {
	DB *gorm.DB
}

func NewSnapshots(db *gorm.DB) *Snapshots { return &Snapshots{DB: db} }

func (r *Snapshots) Insert(ctx context.Context, snapshot *model.Snapshot) error {
	return r.DB.WithContext(ctx).Create(snapshot).Error
}

// ListRecent 按时间倒序返回最近的快照记录（含失败记录，供审计展示）。
func (r *Snapshots) ListRecent(ctx context.Context, limit int) ([]model.Snapshot, error) {
	var items []model.Snapshot
	err := r.DB.WithContext(ctx).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

// ListSuccessful 按时间倒序返回成功快照，用于保留策略裁剪。
func (r *Snapshots) ListSuccessful(ctx context.Context) ([]model.Snapshot, error) {
	var items []model.Snapshot
	err := r.DB.WithContext(ctx).
		Where("status = ?", model.SnapshotStatusOK).
		Order("created_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

func (r *Snapshots) DeleteOne(ctx context.Context, id string) error {
	return r.DB.WithContext(ctx).Where("id = ?", id).Delete(&model.Snapshot{}).Error
}

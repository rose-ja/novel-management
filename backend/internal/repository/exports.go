package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
)

// Exports 是导出任务记录的数据访问层。
type Exports struct {
	DB *gorm.DB
}

func NewExports(db *gorm.DB) *Exports { return &Exports{DB: db} }

func (r *Exports) Insert(ctx context.Context, run *model.ExportRun) error {
	return r.DB.WithContext(ctx).Create(run).Error
}

func (r *Exports) Get(ctx context.Context, id string) (*model.ExportRun, error) {
	var run model.ExportRun
	if err := r.DB.WithContext(ctx).Where("id = ?", id).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// ListRecent 按创建时间倒序返回最近任务。
func (r *Exports) ListRecent(ctx context.Context, limit int) ([]model.ExportRun, error) {
	var items []model.ExportRun
	err := r.DB.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&items).Error
	return items, err
}

// Active 返回正在排队或执行的任务。
func (r *Exports) Active(ctx context.Context) (*model.ExportRun, error) {
	var run model.ExportRun
	err := r.DB.WithContext(ctx).
		Where("status IN ?", []string{model.ExportStatusQueued, model.ExportStatusRunning}).
		Order("created_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// LastFinished 返回最近一次已完成或失败的任务。
func (r *Exports) LastFinished(ctx context.Context) (*model.ExportRun, error) {
	var run model.ExportRun
	err := r.DB.WithContext(ctx).
		Where("status IN ?", []string{model.ExportStatusCompleted, model.ExportStatusFailed}).
		Order("created_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// Update 更新任务字段。
func (r *Exports) Update(ctx context.Context, id string, fields map[string]any) error {
	return r.DB.WithContext(ctx).Model(&model.ExportRun{}).Where("id = ?", id).Updates(fields).Error
}

// MarkInterrupted 把进程重启后遗留的排队/执行中任务标记为失败。
func (r *Exports) MarkInterrupted(ctx context.Context, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.ExportRun{}).
		Where("status IN ?", []string{model.ExportStatusQueued, model.ExportStatusRunning}).
		Updates(map[string]any{
			"status":      model.ExportStatusFailed,
			"error_code":  model.ExportErrorInterrupted,
			"finished_at": now,
		}).Error
}

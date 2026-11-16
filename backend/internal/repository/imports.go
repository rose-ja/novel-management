package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Imports 是导入任务记录的数据访问层。
type Imports struct {
	DB *gorm.DB
}

func NewImports(db *gorm.DB) *Imports { return &Imports{DB: db} }

func (r *Imports) Insert(ctx context.Context, run *model.ImportRun) error {
	return r.DB.WithContext(ctx).Create(run).Error
}

func (r *Imports) Get(ctx context.Context, id string) (*model.ImportRun, error) {
	var run model.ImportRun
	if err := r.DB.WithContext(ctx).Where("id = ?", id).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// GetForUpdate 以行锁读取导入记录，避免并发提交。
func (r *Imports) GetForUpdate(ctx context.Context, id string) (*model.ImportRun, error) {
	var run model.ImportRun
	err := r.DB.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *Imports) Update(ctx context.Context, id string, fields map[string]any) error {
	return r.DB.WithContext(ctx).Model(&model.ImportRun{}).Where("id = ?", id).Updates(fields).Error
}

// Active 返回正在校验或提交的导入任务。
func (r *Imports) Active(ctx context.Context) (*model.ImportRun, error) {
	var run model.ImportRun
	err := r.DB.WithContext(ctx).
		Where("status IN ?", []string{model.ImportStatusValidating, model.ImportStatusRunning}).
		Order("created_at DESC").First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// MarkInterrupted 把进程重启后遗留的校验中/提交中任务标记为失败。
func (r *Imports) MarkInterrupted(ctx context.Context, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.ImportRun{}).
		Where("status IN ?", []string{model.ImportStatusValidating, model.ImportStatusRunning}).
		Updates(map[string]any{
			"status":      model.ImportStatusFailed,
			"error_code":  model.ImportErrorInterrupted,
			"finished_at": now,
			"updated_at":  now,
		}).Error
}

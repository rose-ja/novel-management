package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Media 是图片资产的数据访问层；事务场景通过 NewMedia(tx) 重建。
type Media struct {
	DB *gorm.DB
}

func NewMedia(db *gorm.DB) *Media { return &Media{DB: db} }

// GetActive 返回某实体当前未删除的主图。
func (r *Media) GetActive(ctx context.Context, entityType, entityID string) (*model.MediaAsset, error) {
	var item model.MediaAsset
	err := r.DB.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ? AND deleted_at IS NULL", entityType, entityID).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetActiveForUpdate 在事务内锁定当前主图，供替换/删除串行化。
func (r *Media) GetActiveForUpdate(ctx context.Context, entityType, entityID string) (*model.MediaAsset, error) {
	var item model.MediaAsset
	err := r.DB.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("entity_type = ? AND entity_id = ? AND deleted_at IS NULL", entityType, entityID).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetActiveByID 返回未删除的资产，用于媒体读取。
func (r *Media) GetActiveByID(ctx context.Context, id string) (*model.MediaAsset, error) {
	var item model.MediaAsset
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ListActiveByEntityIDs 批量返回人物主图，避免列表 N+1。
func (r *Media) ListActiveByEntityIDs(ctx context.Context, entityType string, ids []string) ([]model.MediaAsset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []model.MediaAsset
	err := r.DB.WithContext(ctx).
		Where("entity_type = ? AND entity_id IN ? AND deleted_at IS NULL", entityType, ids).
		Find(&items).Error
	return items, err
}

func (r *Media) Insert(ctx context.Context, item *model.MediaAsset) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// MarkEntityDeleted 软删除某实体当前主图，并返回被删除的资产（用于删文件）。
func (r *Media) MarkEntityDeleted(ctx context.Context, entityType, entityID, rootID, action string, now time.Time) ([]model.MediaAsset, error) {
	items, err := r.listActiveForEntity(ctx, entityType, entityID)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	err = r.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Where("entity_type = ? AND entity_id = ? AND deleted_at IS NULL", entityType, entityID).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   rootID,
			"deleted_by_action": action,
			"updated_at":        now,
		}).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// MarkWorkDeleted 软删除作品下全部活跃主图。
func (r *Media) MarkWorkDeleted(ctx context.Context, workID, rootID, action string, now time.Time) ([]model.MediaAsset, error) {
	var items []model.MediaAsset
	if err := r.DB.WithContext(ctx).
		Where("work_id = ? AND deleted_at IS NULL", workID).
		Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	err := r.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Where("work_id = ? AND deleted_at IS NULL", workID).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   rootID,
			"deleted_by_action": action,
			"updated_at":        now,
		}).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Media) listActiveForEntity(ctx context.Context, entityType, entityID string) ([]model.MediaAsset, error) {
	var items []model.MediaAsset
	err := r.DB.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ? AND deleted_at IS NULL", entityType, entityID).
		Find(&items).Error
	return items, err
}

// RestoreByRoot 恢复随根删除的主图；不恢复用户主动移除/替换的旧图。
func (r *Media) RestoreByRoot(ctx context.Context, rootID, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Where("deleted_root_id = ? AND deleted_by_action = ?", rootID, action).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"updated_at":        now,
		}).Error
}

// RestoreByWork 恢复随作品整组删除的主图。
func (r *Media) RestoreByWork(ctx context.Context, workID, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Where("work_id = ? AND deleted_root_id = ? AND deleted_by_action = ?", workID, workID, action).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"updated_at":        now,
		}).Error
}

// PurgeByRoot 物理删除根下全部资产并返回文件路径（含替换/移除的旧图）。
func (r *Media) PurgeByRoot(ctx context.Context, rootID string) ([]string, error) {
	return r.purge(ctx, r.DB.WithContext(ctx).Where("deleted_root_id = ?", rootID))
}

// PurgeByWork 物理删除作品下全部资产并返回文件路径。
func (r *Media) PurgeByWork(ctx context.Context, workID string) ([]string, error) {
	return r.purge(ctx, r.DB.WithContext(ctx).Where("work_id = ?", workID))
}

// PurgeAll 物理删除全部资产（仅供测试/全量清理使用）。
func (r *Media) PurgeAll(ctx context.Context) ([]string, error) {
	return r.purge(ctx, r.DB.WithContext(ctx))
}

// PurgeByDeletedWorks 物理删除回收站作品的资产。
func (r *Media) PurgeByDeletedWorks(ctx context.Context) ([]string, error) {
	return r.purge(ctx, r.DB.WithContext(ctx).
		Where("work_id IN (SELECT id FROM works WHERE deleted_at IS NOT NULL)"))
}

func (r *Media) purge(ctx context.Context, scope *gorm.DB) ([]string, error) {
	var items []model.MediaAsset
	if err := scope.Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	paths := make([]string, 0, len(items)*2)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
		paths = append(paths, item.OriginalPath)
		if item.ThumbPath != nil && *item.ThumbPath != "" {
			paths = append(paths, *item.ThumbPath)
		}
	}
	if err := r.DB.WithContext(ctx).Where("id IN ?", ids).Delete(&model.MediaAsset{}).Error; err != nil {
		return nil, err
	}
	return paths, nil
}

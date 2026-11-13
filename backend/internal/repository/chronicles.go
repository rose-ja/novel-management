package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Chronicles 是大事记的数据访问层。
type Chronicles struct {
	DB *gorm.DB
}

func NewChronicles(db *gorm.DB) *Chronicles { return &Chronicles{DB: db} }

// ListActive 返回某归属实体未删除的大事记，按手工顺序。
func (r *Chronicles) ListActive(ctx context.Context, ownerType, ownerID string) ([]model.Chronicle, error) {
	var items []model.Chronicle
	err := r.DB.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ? AND deleted_at IS NULL", ownerType, ownerID).
		Order("sort_order ASC, id ASC").
		Find(&items).Error
	return items, err
}

// GetActive 返回未删除的大事记。
func (r *Chronicles) GetActive(ctx context.Context, id string) (*model.Chronicle, error) {
	var item model.Chronicle
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetForUpdate 锁定大事记行。
func (r *Chronicles) GetForUpdate(ctx context.Context, id string) (*model.Chronicle, error) {
	var item model.Chronicle
	err := r.DB.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// NextSortOrder 返回追加位置。
func (r *Chronicles) NextSortOrder(ctx context.Context, ownerType, ownerID string) (int, error) {
	var maxOrder *int
	err := r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Select("MAX(sort_order)").Scan(&maxOrder).Error
	if err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return 0, nil
	}
	return *maxOrder + 1, nil
}

func (r *Chronicles) Insert(ctx context.Context, item *model.Chronicle) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// Update 按字段更新并递增版本。
func (r *Chronicles) Update(ctx context.Context, id string, fields map[string]any, version int, now time.Time) error {
	fields["version"] = version + 1
	fields["updated_at"] = now
	res := r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, version).
		Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return model.ErrVersionConflict
	}
	return nil
}

// SoftDelete 软删除单条大事记，作为人物子内容保留 deleted_root_id=owner_id。
func (r *Chronicles) SoftDelete(ctx context.Context, id, rootID, action string, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   rootID,
			"deleted_by_action": action,
			"updated_at":        now,
		})
	return res.RowsAffected, res.Error
}

// MarkOwnerDeleted 软删除某人物下全部活跃大事记。
func (r *Chronicles) MarkOwnerDeleted(ctx context.Context, ownerID, rootID, action string, now time.Time) error {
	return r.MarkOwnerDeletedByType(ctx, model.ChronicleOwnerCharacter, ownerID, rootID, action, now)
}

// MarkOwnerDeletedByType 软删除指定归属类型下全部活跃大事记（人物/势力随根删除）。
func (r *Chronicles) MarkOwnerDeletedByType(ctx context.Context, ownerType, ownerID, rootID, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("owner_type = ? AND owner_id = ? AND deleted_at IS NULL", ownerType, ownerID).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   rootID,
			"deleted_by_action": action,
			"updated_at":        now,
		}).Error
}

// CountActive 返回某归属实体的未删除大事记数量。
func (r *Chronicles) CountActive(ctx context.Context, ownerType, ownerID string) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("owner_type = ? AND owner_id = ? AND deleted_at IS NULL", ownerType, ownerID).
		Count(&total).Error
	return total, err
}

// CountByOwner 返回某归属实体的全部大事记数量（含随根删除），用于回收站影响条数。
func (r *Chronicles) CountByOwner(ctx context.Context, ownerType, ownerID string) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Count(&total).Error
	return total, err
}

// CountByRoot 返回随指定根删除的大事件数量，用于恢复结果计数。
func (r *Chronicles) CountByRoot(ctx context.Context, rootID, action string) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("deleted_root_id = ? AND deleted_by_action = ?", rootID, action).
		Count(&total).Error
	return total, err
}

// MarkWorkDeleted 软删除作品下全部活跃大事记。
func (r *Chronicles) MarkWorkDeleted(ctx context.Context, workID, rootID, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("work_id = ? AND deleted_at IS NULL", workID).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   rootID,
			"deleted_by_action": action,
			"updated_at":        now,
		}).Error
}

// RestoreByRoot 恢复随根删除的大事记，不恢复用户单独删除的条目。
func (r *Chronicles) RestoreByRoot(ctx context.Context, rootID, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("deleted_root_id = ? AND deleted_by_action = ?", rootID, action).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"updated_at":        now,
		}).Error
}

// RestoreByWork 恢复随作品整组删除的大事记。
func (r *Chronicles) RestoreByWork(ctx context.Context, workID, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("work_id = ? AND deleted_root_id = ? AND deleted_by_action = ?", workID, workID, action).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"updated_at":        now,
		}).Error
}

// RestoreDeleted 恢复单条（撤销）。
func (r *Chronicles) RestoreDeleted(ctx context.Context, id string, now time.Time) error {
	res := r.DB.WithContext(ctx).Model(&model.Chronicle{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"updated_at":        now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return model.ErrAlreadyDeleted
	}
	return nil
}

// GetDeleted 返回已删大事记，供撤销校验归属。
func (r *Chronicles) GetDeleted(ctx context.Context, id string) (*model.Chronicle, error) {
	var item model.Chronicle
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NOT NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrAlreadyDeleted
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// PurgeByRoot 物理删除根下全部大事记。
func (r *Chronicles) PurgeByRoot(ctx context.Context, rootID string) error {
	return r.DB.WithContext(ctx).Where("deleted_root_id = ?", rootID).Delete(&model.Chronicle{}).Error
}

// PurgeByWork 物理删除作品下全部大事记。
func (r *Chronicles) PurgeByWork(ctx context.Context, workID string) error {
	return r.DB.WithContext(ctx).Where("work_id = ?", workID).Delete(&model.Chronicle{}).Error
}

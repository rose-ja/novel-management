package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Works 是作品与内置类别的数据访问层；事务场景通过 NewWorks(tx) 重建实例。
type Works struct {
	DB *gorm.DB
}

func NewWorks(db *gorm.DB) *Works { return &Works{DB: db} }

// List 返回未删除作品与总数；q 为空表示不过滤。
func (r *Works) List(ctx context.Context, q string, page, size int) ([]model.Work, int64, error) {
	build := func() *gorm.DB {
		tx := r.DB.WithContext(ctx).Model(&model.Work{}).Where("deleted_at IS NULL")
		if pattern := likePattern(q); pattern != "" {
			tx = tx.Where(`name_key LIKE ? ESCAPE '\'`, pattern)
		}
		return tx
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.Work
	if err := build().Order("created_at ASC, id ASC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get 返回未删除作品；不存在或已软删除统一 ErrNotFound。
func (r *Works) Get(ctx context.Context, id string) (*model.Work, error) {
	var work model.Work
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&work).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &work, nil
}

// GetForUpdate 以行锁读取未删除作品，供子资源写入/恢复时避免与整组软删除并发。
func (r *Works) GetForUpdate(ctx context.Context, id string) (*model.Work, error) {
	var work model.Work
	err := r.DB.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", id).First(&work).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &work, nil
}

// GetDeleted 返回回收站中的作品根。
func (r *Works) GetDeleted(ctx context.Context, id string) (*model.Work, error) {
	var work model.Work
	err := r.DB.WithContext(ctx).
		Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id).
		First(&work).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &work, nil
}

// Insert 创建作品；名称冲突由调用方映射为 ErrNameConflict。
func (r *Works) Insert(ctx context.Context, work *model.Work) error {
	return r.DB.WithContext(ctx).Create(work).Error
}

// UpdateWithVersion 使用 version 做乐观锁；返回受影响行数（0 表示版本冲突或已删除）。
func (r *Works) UpdateWithVersion(ctx context.Context, id, name, nameKey string, description *string, version int, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("works").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, version).
		Updates(map[string]any{
			"name":        name,
			"name_key":    nameKey,
			"description": description,
			"version":     version + 1,
			"updated_at":  now,
		})
	return res.RowsAffected, res.Error
}

// SoftDeleteGroup 在同一事务内软删除作品根、类别、节点与人物。
func (r *Works) SoftDeleteGroup(ctx context.Context, id, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Table("works").
			Where("id = ? AND deleted_at IS NULL", id).
			Updates(map[string]any{
				"deleted_at":        now,
				"deleted_root_id":   id,
				"deleted_by_action": action,
				"version":           gorm.Expr("version + 1"),
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return model.ErrNotFound
		}

		if err := tx.Table("system_categories").
			Where("work_id = ? AND deleted_at IS NULL", id).
			Updates(map[string]any{
				"deleted_at":        now,
				"deleted_root_id":   id,
				"deleted_by_action": action,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Table("system_nodes").
			Where("work_id = ? AND deleted_at IS NULL", id).
			Updates(map[string]any{
				"deleted_at":        now,
				"deleted_root_id":   id,
				"deleted_by_action": action,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Table("characters").
			Where("work_id = ? AND deleted_at IS NULL", id).
			Updates(map[string]any{
				"deleted_at":        now,
				"deleted_root_id":   id,
				"deleted_by_action": action,
				"version":           gorm.Expr("version + 1"),
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		if _, err := NewMedia(tx).MarkWorkDeleted(ctx, id, id, model.ActionWorkDelete, now); err != nil {
			return err
		}
		return NewChronicles(tx).MarkWorkDeleted(ctx, id, id, model.ActionWorkDelete, now)
	})
}

// RestoreGroup 整组恢复（作品根 + 类别 + 节点 + 人物）；任一缺失返回 ErrNotFound。
func (r *Works) RestoreGroup(ctx context.Context, id string, now time.Time) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Table("works").
			Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id).
			Updates(map[string]any{
				"deleted_at":        nil,
				"deleted_root_id":   nil,
				"deleted_by_action": nil,
				"version":           gorm.Expr("version + 1"),
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return model.ErrNotFound
		}

		if err := tx.Table("system_categories").
			Where("work_id = ? AND deleted_root_id = ?", id, id).
			Updates(map[string]any{
				"deleted_at":        nil,
				"deleted_root_id":   nil,
				"deleted_by_action": nil,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Table("system_nodes").
			Where("work_id = ? AND deleted_root_id = ?", id, id).
			Updates(map[string]any{
				"deleted_at":        nil,
				"deleted_root_id":   nil,
				"deleted_by_action": nil,
				"version":           gorm.Expr("version + 1"),
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Table("characters").
			Where("work_id = ? AND deleted_root_id = ?", id, id).
			Updates(map[string]any{
				"deleted_at":        nil,
				"deleted_root_id":   nil,
				"deleted_by_action": nil,
				"version":           gorm.Expr("version + 1"),
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		if err := NewMedia(tx).RestoreByWork(ctx, id, model.ActionWorkDelete, now); err != nil {
			return err
		}
		return NewChronicles(tx).RestoreByWork(ctx, id, model.ActionWorkDelete, now)
	})
}

// RenameDeleted 在恢复前修正回收站中的作品名称。
func (r *Works) RenameDeleted(ctx context.Context, id, name, nameKey string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("works").
		Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id).
		Updates(map[string]any{"name": name, "name_key": nameKey, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

// ListDeletedRoots 返回全部回收站作品根（按删除时间倒序）。
func (r *Works) ListDeletedRoots(ctx context.Context) ([]model.Work, error) {
	var items []model.Work
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_root_id = id").
		Order("deleted_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

// CountDeletedChildren 返回该作品根下被软删除的类别与节点数量。
func (r *Works) CountDeletedChildren(ctx context.Context, rootID string) (int64, error) {
	var total int64
	if err := r.DB.WithContext(ctx).Model(&model.SystemCategory{}).
		Where("deleted_root_id = ?", rootID).
		Count(&total).Error; err != nil {
		return 0, err
	}
	var nodes int64
	if err := r.DB.WithContext(ctx).Model(&model.SystemNode{}).
		Where("deleted_root_id = ?", rootID).
		Count(&nodes).Error; err != nil {
		return 0, err
	}
	return total + nodes, nil
}

// CountDeletedByRoot 一次性统计各回收站作品根下的类别与节点数量，避免列表 N+1 查询。
func (r *Works) CountDeletedByRoot(ctx context.Context) (map[string]int64, error) {
	type row struct {
		RootID string
		Total  int64
	}
	counts := make(map[string]int64)
	tables := []any{&model.SystemCategory{}, &model.SystemNode{}}
	for _, target := range tables {
		var rows []row
		err := r.DB.WithContext(ctx).Model(target).
			Select("deleted_root_id AS root_id, count(*) AS total").
			Where("deleted_root_id IS NOT NULL").
			Group("deleted_root_id").
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, item := range rows {
			// 仅统计作品根（类别/节点的 deleted_root_id 指向作品 ID）。
			counts[item.RootID] += item.Total
		}
	}
	return counts, nil
}

// ActiveName 返回未删除作品的名称；作品不存在或已软删除统一 ErrNotFound。
func (r *Works) ActiveName(ctx context.Context, id string) (string, error) {
	var work model.Work
	err := r.DB.WithContext(ctx).Select("id", "name").Where("id = ? AND deleted_at IS NULL", id).Take(&work).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", model.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return work.Name, nil
}

// Name 返回作品名称（含回收站记录），用于类别回收站展示父级。
func (r *Works) Name(ctx context.Context, id string) (string, error) {
	var work model.Work
	err := r.DB.WithContext(ctx).Select("id", "name").Where("id = ?", id).Take(&work).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", model.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return work.Name, nil
}

// PurgeGroup 物理删除作品根；先清引用、别名、人物、图片与大事记，再删节点、类别与作品。
// 返回待删除的图片相对路径，由调用方在提交后清理文件。
func (r *Works) PurgeGroup(ctx context.Context, id string) (int64, []string, error) {
	var affected int64
	var paths []string
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		childrenPaths, err := purgeWorkChildren(ctx, tx, id)
		if err != nil {
			return err
		}
		paths = childrenPaths
		res := tx.Where("id = ? AND deleted_at IS NOT NULL", id).Delete(&model.Work{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, paths, err
}

// PurgeAllDeleted 物理删除全部回收站作品根，顺序同上；返回待删除的图片相对路径。
func (r *Works) PurgeAllDeleted(ctx context.Context) (int64, []string, error) {
	var affected int64
	var paths []string
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		scope := "work_id IN (SELECT id FROM works WHERE deleted_at IS NOT NULL)"
		if err := tx.Exec("DELETE FROM character_system_refs WHERE " + scope).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM character_names WHERE " + scope).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM characters WHERE " + scope).Error; err != nil {
			return err
		}
		mediaPaths, err := NewMedia(tx).PurgeByDeletedWorks(ctx)
		if err != nil {
			return err
		}
		paths = mediaPaths
		if err := tx.Exec("DELETE FROM chronicles WHERE " + scope).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM system_nodes WHERE " + scope).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM system_categories WHERE " + scope).Error; err != nil {
			return err
		}
		res := tx.Where("deleted_at IS NOT NULL").Delete(&model.Work{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, paths, err
}

func purgeWorkChildren(ctx context.Context, tx *gorm.DB, workID string) ([]string, error) {
	// 与单条人物 Purge 保持一致的「人物行 → 引用 → 别名 → 媒体」加锁顺序，避免并发死锁。
	var characterIDs []string
	if err := tx.WithContext(ctx).
		Raw("SELECT id FROM characters WHERE work_id = ? FOR UPDATE", workID).
		Scan(&characterIDs).Error; err != nil {
		return nil, err
	}
	if err := tx.WithContext(ctx).Exec("DELETE FROM character_system_refs WHERE work_id = ?", workID).Error; err != nil {
		return nil, err
	}
	if err := tx.WithContext(ctx).Exec("DELETE FROM character_names WHERE work_id = ?", workID).Error; err != nil {
		return nil, err
	}
	if err := tx.WithContext(ctx).Exec("DELETE FROM characters WHERE work_id = ?", workID).Error; err != nil {
		return nil, err
	}
	paths, err := NewMedia(tx).PurgeByWork(ctx, workID)
	if err != nil {
		return nil, err
	}
	if err := NewChronicles(tx).PurgeByWork(ctx, workID); err != nil {
		return nil, err
	}
	if err := tx.WithContext(ctx).Exec("DELETE FROM system_nodes WHERE work_id = ?", workID).Error; err != nil {
		return nil, err
	}
	if err := tx.WithContext(ctx).Where("work_id = ?", workID).Delete(&model.SystemCategory{}).Error; err != nil {
		return nil, err
	}
	return paths, nil
}

// NameTaken 判断未删除作品中是否已有同名（用于恢复冲突预检）。
func (r *Works) NameTaken(ctx context.Context, nameKey string) (bool, error) {
	var total int64
	err := r.DB.WithContext(ctx).Model(&model.Work{}).
		Where("name_key = ? AND deleted_at IS NULL", nameKey).
		Count(&total).Error
	return total > 0, err
}

// InsertCategory 写入内置类别（创建作品事务内调用）。
func (r *Works) InsertCategory(ctx context.Context, category *model.SystemCategory) error {
	return r.DB.WithContext(ctx).Create(category).Error
}

// likePattern 转义 LIKE 通配符，空查询返回空串表示不过滤。
func likePattern(q string) string {
	trimmed := strings.ToLower(strings.TrimSpace(q))
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(trimmed) + "%"
}

package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
)

// System 是体系类别、节点与人物引用的数据访问层。
type System struct {
	DB *gorm.DB
}

func NewSystem(db *gorm.DB) *System { return &System{DB: db} }

type CountRow struct {
	Key   string `gorm:"column:key"`
	Total int64  `gorm:"column:total"`
}

type ReferenceRow struct {
	CharacterID string `gorm:"column:character_id"`
	NodeID      string `gorm:"column:node_id"`
	NodeName    string `gorm:"column:node_name"`
}

// ListCategories 返回未删除类别；内置置顶，其余按创建时间稳定排序。
func (r *System) ListCategories(ctx context.Context, workID string, includeDisabled bool) ([]model.SystemCategory, error) {
	var items []model.SystemCategory
	tx := r.DB.WithContext(ctx).
		Where("work_id = ? AND deleted_at IS NULL", workID)
	if !includeDisabled {
		tx = tx.Where("is_enabled = ?", true)
	}
	if err := tx.Order("is_builtin DESC, created_at ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	nodeCounts, err := r.nodeCountsByCategory(ctx, workID, ids)
	if err != nil {
		return nil, err
	}
	refCounts, err := r.referenceCountsByCategory(ctx, workID, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].NodeCount = nodeCounts[items[i].ID]
		items[i].ReferenceCount = refCounts[items[i].ID]
	}
	return items, nil
}

func (r *System) nodeCountsByCategory(ctx context.Context, workID string, categoryIDs []string) (map[string]int64, error) {
	if len(categoryIDs) == 0 {
		return map[string]int64{}, nil
	}
	var rows []CountRow
	err := r.DB.WithContext(ctx).Table("system_nodes").
		Select("category_id AS key, count(*) AS total").
		Where("work_id = ? AND deleted_at IS NULL AND category_id IN ?", workID, categoryIDs).
		Group("category_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Total
	}
	return out, nil
}

func (r *System) referenceCountsByCategory(ctx context.Context, workID string, categoryIDs []string) (map[string]int64, error) {
	if len(categoryIDs) == 0 {
		return map[string]int64{}, nil
	}
	var rows []CountRow
	err := r.DB.WithContext(ctx).Table("character_system_refs").
		Select("category_id AS key, count(DISTINCT character_id) AS total").
		Where("work_id = ? AND category_id IN ?", workID, categoryIDs).
		Group("category_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Total
	}
	return out, nil
}

// GetCategory 返回未删除类别。
func (r *System) GetCategory(ctx context.Context, id string) (*model.SystemCategory, error) {
	var item model.SystemCategory
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CountActiveCategories 统计作品下未删除类别（含内置）。
func (r *System) CountActiveCategories(ctx context.Context, workID string) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Table("system_categories").
		Where("work_id = ? AND deleted_at IS NULL", workID).
		Count(&total).Error
	return total, err
}

// NextCategorySortOrder 返回当前最大 sort_order + 1。
func (r *System) NextCategorySortOrder(ctx context.Context, workID string) (int, error) {
	var row struct {
		MaxValue int `gorm:"column:max_value"`
	}
	err := r.DB.WithContext(ctx).Table("system_categories").
		Where("work_id = ? AND deleted_at IS NULL", workID).
		Select("COALESCE(MAX(sort_order), -1) AS max_value").
		Scan(&row).Error
	if err != nil {
		return 0, err
	}
	return row.MaxValue + 1, nil
}

// CategoryNameTaken 判断作品下是否有同名未删除类别。
func (r *System) CategoryNameTaken(ctx context.Context, workID, nameKey, excludeID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Table("system_categories").
		Where("work_id = ? AND name_key = ? AND deleted_at IS NULL", workID, nameKey)
	if excludeID != "" {
		tx = tx.Where("id <> ?", excludeID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *System) InsertCategory(ctx context.Context, item *model.SystemCategory) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// UpdateCategory 一次写入名称与启用状态，使用 version 做乐观锁。
func (r *System) UpdateCategory(ctx context.Context, id, name, nameKey string, enabled bool, version int, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("system_categories").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, version).
		Updates(map[string]any{
			"name":       name,
			"name_key":   nameKey,
			"is_enabled": enabled,
			"version":    version + 1,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// SoftDeleteCategoryGroup 在同一事务内软删除类别与全部未删除节点，返回节点数量。
func (r *System) SoftDeleteCategoryGroup(ctx context.Context, id string, version int, action string, now time.Time) (int64, error) {
	var affectedNodes int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		nodeRes := tx.Table("system_nodes").
			Where("category_id = ? AND deleted_at IS NULL", id).
			Updates(map[string]any{
				"deleted_at":        now,
				"deleted_root_id":   id,
				"deleted_by_action": action,
				"version":           gorm.Expr("version + 1"),
				"updated_at":        now,
			})
		if nodeRes.Error != nil {
			return nodeRes.Error
		}
		affectedNodes = nodeRes.RowsAffected

		res := tx.Table("system_categories").
			Where("id = ? AND deleted_at IS NULL AND version = ?", id, version).
			Updates(map[string]any{
				"deleted_at":        now,
				"deleted_root_id":   id,
				"deleted_by_action": action,
				"version":           version + 1,
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return model.ErrVersionConflict
		}
		return nil
	})
	return affectedNodes, err
}

// GetDeletedCategoryRoot 返回直接删除进回收站的类别根（不含随作品删除的类别）。
func (r *System) GetDeletedCategoryRoot(ctx context.Context, id string) (*model.SystemCategory, error) {
	var item model.SystemCategory
	err := r.DB.WithContext(ctx).
		Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *System) RenameDeletedCategory(ctx context.Context, id, name, nameKey string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("system_categories").
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

// RestoreCategoryGroup 恢复类别与整组节点。
func (r *System) RestoreCategoryGroup(ctx context.Context, id string, now time.Time) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Table("system_categories").
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
		return tx.Table("system_nodes").
			Where("deleted_root_id = ?", id).
			Updates(map[string]any{
				"deleted_at":        nil,
				"deleted_root_id":   nil,
				"deleted_by_action": nil,
				"version":           gorm.Expr("version + 1"),
				"updated_at":        now,
			}).Error
	})
}

// ListDeletedCategoryRoots 返回全部类别回收站根。
func (r *System) ListDeletedCategoryRoots(ctx context.Context) ([]model.SystemCategory, error) {
	var items []model.SystemCategory
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_root_id = id").
		Order("deleted_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

// CountDeletedNodesByCategoryRoot 一次性统计类别根下的节点数量。
func (r *System) CountDeletedNodesByCategoryRoot(ctx context.Context) (map[string]int64, error) {
	var rows []CountRow
	err := r.DB.WithContext(ctx).Table("system_nodes").
		Select("deleted_root_id AS key, count(*) AS total").
		Where("deleted_root_id IS NOT NULL").
		Group("deleted_root_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Total
	}
	return out, nil
}

// PurgeCategoryGroup 物理删除类别组；调用方必须先确认无人物引用。
func (r *System) PurgeCategoryGroup(ctx context.Context, id string) (int64, error) {
	var deleted int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("system_nodes").Where("category_id = ?", id).Delete(&model.SystemNode{}).Error; err != nil {
			return err
		}
		res := tx.Table("system_categories").Where("id = ?", id).Delete(&model.SystemCategory{})
		if res.Error != nil {
			return res.Error
		}
		deleted = res.RowsAffected
		return nil
	})
	return deleted, err
}

// ListNodes 返回类别下未删除节点；reviewDB 为空表示不过滤。
func (r *System) ListNodes(ctx context.Context, categoryID, reviewDB string) ([]model.SystemNode, error) {
	var items []model.SystemNode
	tx := r.DB.WithContext(ctx).
		Where("category_id = ? AND deleted_at IS NULL", categoryID)
	if reviewDB != "" {
		tx = tx.Where("review_status = ?", reviewDB)
	}
	if err := tx.Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	refCounts, err := r.referenceCountsByNode(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].ReferenceCount = refCounts[items[i].ID]
	}
	return items, nil
}

func (r *System) referenceCountsByNode(ctx context.Context, nodeIDs []string) (map[string]int64, error) {
	if len(nodeIDs) == 0 {
		return map[string]int64{}, nil
	}
	var rows []CountRow
	err := r.DB.WithContext(ctx).Table("character_system_refs").
		Select("node_id AS key, count(DISTINCT character_id) AS total").
		Where("node_id IN ?", nodeIDs).
		Group("node_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Total
	}
	return out, nil
}

// GetNode 返回未删除节点。
func (r *System) GetNode(ctx context.Context, id string) (*model.SystemNode, error) {
	var item model.SystemNode
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// NodeNameTaken 判断同类别同父级是否已有同名未删除节点。
func (r *System) NodeNameTaken(ctx context.Context, categoryID string, parentID *string, nameKey, excludeID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Table("system_nodes").
		Where("category_id = ? AND name_key = ? AND deleted_at IS NULL", categoryID, nameKey)
	if parentID == nil {
		tx = tx.Where("parent_id IS NULL")
	} else {
		tx = tx.Where("parent_id = ?", *parentID)
	}
	if excludeID != "" {
		tx = tx.Where("id <> ?", excludeID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *System) CountNodeChildren(ctx context.Context, nodeID string) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Table("system_nodes").
		Where("parent_id = ? AND deleted_at IS NULL", nodeID).
		Count(&total).Error
	return total, err
}

func (r *System) NextNodeSortOrder(ctx context.Context, categoryID string, parentID *string) (int, error) {
	tx := r.DB.WithContext(ctx).Table("system_nodes").
		Where("category_id = ? AND deleted_at IS NULL", categoryID)
	if parentID == nil {
		tx = tx.Where("parent_id IS NULL")
	} else {
		tx = tx.Where("parent_id = ?", *parentID)
	}
	var row struct {
		MaxValue int `gorm:"column:max_value"`
	}
	if err := tx.Select("COALESCE(MAX(sort_order), -1) AS max_value").Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.MaxValue + 1, nil
}

func (r *System) InsertNode(ctx context.Context, item *model.SystemNode) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

type NodeUpdate struct {
	Name        string
	NameKey     string
	SortOrder   int
	EraStart    *string
	EraEnd      *string
	Version     int
}

func (r *System) UpdateNode(ctx context.Context, id string, update NodeUpdate, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("system_nodes").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, update.Version).
		Updates(map[string]any{
			"name":           update.Name,
			"name_key":       update.NameKey,
			"sort_order":     update.SortOrder,
			"era_start_text": update.EraStart,
			"era_end_text":   update.EraEnd,
			"version":        update.Version + 1,
			"updated_at":     now,
		})
	return res.RowsAffected, res.Error
}

func (r *System) SoftDeleteNode(ctx context.Context, id string, version int, action string, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("system_nodes").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, version).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   id,
			"deleted_by_action": action,
			"version":           version + 1,
			"updated_at":        now,
		})
	return res.RowsAffected, res.Error
}

func (r *System) GetDeletedNodeRoot(ctx context.Context, id string) (*model.SystemNode, error) {
	var item model.SystemNode
	err := r.DB.WithContext(ctx).
		Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *System) RenameDeletedNode(ctx context.Context, id, name, nameKey string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("system_nodes").
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

func (r *System) RestoreNode(ctx context.Context, id string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("system_nodes").
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
	return nil
}

func (r *System) ListDeletedNodeRoots(ctx context.Context) ([]model.SystemNode, error) {
	var items []model.SystemNode
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_root_id = id").
		Order("deleted_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

// PurgeNode 物理删除节点；调用方必须先确认无人物引用。
func (r *System) PurgeNode(ctx context.Context, id string) (int64, error) {
	res := r.DB.WithContext(ctx).Where("id = ?", id).Delete(&model.SystemNode{})
	return res.RowsAffected, res.Error
}

// SiblingIDs 返回同类别同父级的未删除节点 ID（按当前顺序）。
func (r *System) SiblingIDs(ctx context.Context, categoryID string, parentID *string) ([]string, error) {
	tx := r.DB.WithContext(ctx).Table("system_nodes").
		Where("category_id = ? AND deleted_at IS NULL", categoryID)
	if parentID == nil {
		tx = tx.Where("parent_id IS NULL")
	} else {
		tx = tx.Where("parent_id = ?", *parentID)
	}
	var ids []string
	err := tx.Order("sort_order ASC, id ASC").Pluck("id", &ids).Error
	return ids, err
}

// ReorderNodes 按给定顺序写 sort_order（0..n-1）。
func (r *System) ReorderNodes(ctx context.Context, ids []string, now time.Time) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, id := range ids {
			if err := tx.Table("system_nodes").
				Where("id = ? AND deleted_at IS NULL", id).
				Updates(map[string]any{"sort_order": index, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListReferencesByCategory 返回类别下全部人物引用（按节点、人物排序）。
func (r *System) ListReferencesByCategory(ctx context.Context, categoryID string) ([]ReferenceRow, error) {
	var rows []ReferenceRow
	err := r.DB.WithContext(ctx).Table("character_system_refs AS ref").
		Select("ref.character_id, ref.node_id, node.name AS node_name").
		Joins("JOIN system_nodes node ON node.id = ref.node_id").
		Where("ref.category_id = ?", categoryID).
		Order("ref.node_id ASC, ref.character_id ASC").
		Scan(&rows).Error
	return rows, err
}

// ListReferencesByNode 返回单个节点的全部人物引用。
func (r *System) ListReferencesByNode(ctx context.Context, nodeID string) ([]ReferenceRow, error) {
	var rows []ReferenceRow
	err := r.DB.WithContext(ctx).Table("character_system_refs AS ref").
		Select("ref.character_id, ref.node_id, node.name AS node_name").
		Joins("JOIN system_nodes node ON node.id = ref.node_id").
		Where("ref.node_id = ?", nodeID).
		Order("ref.character_id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *System) AnyReferenceForCategory(ctx context.Context, categoryID string) (bool, error) {
	var total int64
	err := r.DB.WithContext(ctx).Table("character_system_refs").
		Where("category_id = ?", categoryID).
		Count(&total).Error
	return total > 0, err
}

func (r *System) AnyReferenceForNode(ctx context.Context, nodeID string) (bool, error) {
	var total int64
	err := r.DB.WithContext(ctx).Table("character_system_refs").
		Where("node_id = ?", nodeID).
		Count(&total).Error
	return total > 0, err
}

// CategoryName 返回类别名称（含已删除），用于回收站父级展示。
func (r *System) CategoryName(ctx context.Context, id string) (string, error) {
	var item model.SystemCategory
	err := r.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", model.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return item.Name, nil
}

// NodeName 返回节点名称（含已删除），用于回收站展示。
func (r *System) NodeName(ctx context.Context, id string) (string, error) {
	var item model.SystemNode
	err := r.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", model.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return item.Name, nil
}

// CategoryEnabled 返回类别启用状态；不存在返 ErrNotFound。
func (r *System) CategoryEnabled(ctx context.Context, id string) (bool, error) {
	var item model.SystemCategory
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, model.ErrNotFound
	}
	if err != nil {
		return false, err
	}
	return item.IsEnabled, nil
}

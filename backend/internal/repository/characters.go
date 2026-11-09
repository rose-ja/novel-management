package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
)

// Characters 是人物、别名与体系引用的数据访问层；事务场景通过 NewCharacters(tx) 重建实例。
type Characters struct {
	DB *gorm.DB
}

func NewCharacters(db *gorm.DB) *Characters { return &Characters{DB: db} }

// CharacterFilter 是列表查询条件；空值表示不过滤。
type CharacterFilter struct {
	WorkID       string
	Query        string
	SystemNodeID string
	FactionID    string
	ReviewDB     string
	Sort         string
	Page         int
	PageSize     int
}

// CharacterSystemRefRow 是详情/列表共用的体系引用行，含节点与父层级路径。
type CharacterSystemRefRow struct {
	CharacterID     string  `gorm:"column:character_id"`
	CategoryID      string  `gorm:"column:category_id"`
	CategoryName    string  `gorm:"column:category_name"`
	CategoryType    string  `gorm:"column:category_type"`
	CategoryBuiltin bool    `gorm:"column:category_builtin"`
	NodeID          string  `gorm:"column:node_id"`
	NodeName        string  `gorm:"column:node_name"`
	ParentID        *string `gorm:"column:parent_id"`
	ParentName      *string `gorm:"column:parent_name"`
}

// characterOrder 将白名单排序值映射为固定列，避免字符串拼接注入。
func characterOrder(sort string) string {
	switch sort {
	case "name":
		return "characters.name ASC, characters.id ASC"
	case "-name":
		return "characters.name DESC, characters.id DESC"
	case "updated_at":
		return "characters.updated_at ASC, characters.id ASC"
	default:
		return "characters.updated_at DESC, characters.id DESC"
	}
}

func (r *Characters) buildList(ctx context.Context, filter CharacterFilter) *gorm.DB {
	tx := r.DB.WithContext(ctx).Model(&model.Character{}).
		Where("characters.work_id = ? AND characters.deleted_at IS NULL", filter.WorkID)
	if pattern := likePattern(filter.Query); pattern != "" {
		tx = tx.Where(
			`(characters.name_key LIKE ? ESCAPE '\' OR EXISTS (
				SELECT 1 FROM character_names n
				WHERE n.character_id = characters.id AND n.value_key LIKE ? ESCAPE '\'
			))`,
			pattern, pattern,
		)
	}
	if filter.SystemNodeID != "" {
		tx = tx.Where(
			"EXISTS (SELECT 1 FROM character_system_refs r WHERE r.character_id = characters.id AND r.node_id = ?)",
			filter.SystemNodeID,
		)
	}
	if filter.FactionID != "" {
		tx = tx.Where("characters.faction_id = ?", filter.FactionID)
	}
	if filter.ReviewDB != "" {
		tx = tx.Where("characters.review_status = ?", filter.ReviewDB)
	}
	return tx
}

// List 返回当前作品未删除人物与筛选总数。
func (r *Characters) List(ctx context.Context, filter CharacterFilter) ([]model.Character, int64, error) {
	var total int64
	if err := r.buildList(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Character
	err := r.buildList(ctx, filter).
		Order(characterOrder(filter.Sort)).
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get 返回未删除人物；不存在或已软删除统一 ErrNotFound。
func (r *Characters) Get(ctx context.Context, id string) (*model.Character, error) {
	var item model.Character
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetActiveByWork 返回指定作品下未删除人物。
func (r *Characters) GetActiveByWork(ctx context.Context, workID, id string) (*model.Character, error) {
	var item model.Character
	err := r.DB.WithContext(ctx).
		Where("id = ? AND work_id = ? AND deleted_at IS NULL", id, workID).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetDeleted 返回直接删除进回收站的人物根（不含随作品删除的人物）。
func (r *Characters) GetDeleted(ctx context.Context, id string) (*model.Character, error) {
	var item model.Character
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

func (r *Characters) Insert(ctx context.Context, item *model.Character) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// NameTaken 判断作品下是否有同名未删除人物。
func (r *Characters) NameTaken(ctx context.Context, workID, nameKey, excludeID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Table("characters").
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

// ActiveByName 返回同名活跃人物，用于 NAME_CONFLICT details。
func (r *Characters) ActiveByName(ctx context.Context, workID, nameKey, excludeID string) (*model.Character, error) {
	tx := r.DB.WithContext(ctx).
		Where("work_id = ? AND name_key = ? AND deleted_at IS NULL", workID, nameKey)
	if excludeID != "" {
		tx = tx.Where("id <> ?", excludeID)
	}
	var item model.Character
	err := tx.First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CharacterUpdate 是一次完整字段写入，version 由调用方保证与 If-Match 一致。
type CharacterUpdate struct {
	Name        string
	NameKey     string
	Description *string
	FactionID   *string
	Version     int
}

// UpdateWithVersion 使用 version 做乐观锁；返回受影响行数（0 表示版本冲突或已删除）。
func (r *Characters) UpdateWithVersion(ctx context.Context, id string, update CharacterUpdate, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("characters").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, update.Version).
		Updates(map[string]any{
			"name":        update.Name,
			"name_key":    update.NameKey,
			"description": update.Description,
			"faction_id":  update.FactionID,
			"version":     update.Version + 1,
			"updated_at":  now,
		})
	return res.RowsAffected, res.Error
}

// ReplaceNames 整组替换别名/称号（调用方负责事务）。
func (r *Characters) ReplaceNames(ctx context.Context, characterID string, names []model.CharacterName) error {
	if err := r.DB.WithContext(ctx).
		Where("character_id = ?", characterID).
		Delete(&model.CharacterName{}).Error; err != nil {
		return err
	}
	if len(names) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Create(&names).Error
}

// ListNames 返回人物全部别名/称号，按提交顺序排序。
func (r *Characters) ListNames(ctx context.Context, characterID string) ([]model.CharacterName, error) {
	var items []model.CharacterName
	err := r.DB.WithContext(ctx).
		Where("character_id = ?", characterID).
		Order("sort_order ASC, created_at ASC, id ASC").
		Find(&items).Error
	return items, err
}

// ListNamesByCharacters 一次批量读取多个人物的别名/称号，避免列表 N+1。
func (r *Characters) ListNamesByCharacters(ctx context.Context, characterIDs []string) ([]model.CharacterName, error) {
	if len(characterIDs) == 0 {
		return []model.CharacterName{}, nil
	}
	var items []model.CharacterName
	err := r.DB.WithContext(ctx).
		Where("character_id IN ?", characterIDs).
		Order("character_id ASC, sort_order ASC, created_at ASC, id ASC").
		Find(&items).Error
	return items, err
}

// ReplaceSystemRefs 整组替换体系引用（调用方负责事务）。
func (r *Characters) ReplaceSystemRefs(ctx context.Context, characterID string, refs []model.CharacterSystemRef) error {
	if err := r.DB.WithContext(ctx).
		Where("character_id = ?", characterID).
		Delete(&model.CharacterSystemRef{}).Error; err != nil {
		return err
	}
	if len(refs) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Create(&refs).Error
}

// ListSystemRefRows 返回人物的体系引用路径（含父层级）；按内置类别优先、类别创建顺序排序。
func (r *Characters) ListSystemRefRows(ctx context.Context, characterIDs []string) ([]CharacterSystemRefRow, error) {
	if len(characterIDs) == 0 {
		return []CharacterSystemRefRow{}, nil
	}
	var rows []CharacterSystemRefRow
	err := r.DB.WithContext(ctx).Table("character_system_refs AS ref").
		Select(`ref.character_id, ref.category_id, cat.name AS category_name,
			cat.category_type, cat.is_builtin AS category_builtin,
			node.id AS node_id, node.name AS node_name,
			node.parent_id, parent.name AS parent_name`).
		Joins("JOIN system_categories cat ON cat.id = ref.category_id").
		Joins("JOIN system_nodes node ON node.id = ref.node_id").
		Joins("LEFT JOIN system_nodes parent ON parent.id = node.parent_id").
		Where("ref.character_id IN ?", characterIDs).
		Order("ref.character_id ASC, cat.is_builtin DESC, cat.created_at ASC, cat.id ASC, node.sort_order ASC, node.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []CharacterSystemRefRow{}
	}
	return rows, nil
}

// SoftDelete 软删除人物自身根；version 不匹配返回 0 行。
func (r *Characters) SoftDelete(ctx context.Context, id, action string, version int, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("characters").
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

// Restore 恢复人物根并递增版本；体系引用与别名未删除，无需重建。
func (r *Characters) Restore(ctx context.Context, id string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("characters").
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

// RestoreGroupByWork 恢复随作品整组删除的人物（deleted_root_id = workID）。
func (r *Characters) RestoreGroupByWork(ctx context.Context, workID string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("characters").
		Where("deleted_root_id = ?", workID).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"version":           gorm.Expr("version + 1"),
			"updated_at":        now,
		}).Error
}

// ListDeletedRoots 返回全部人物回收站根（按删除时间倒序）。
func (r *Characters) ListDeletedRoots(ctx context.Context) ([]model.Character, error) {
	var items []model.Character
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_root_id = id").
		Order("deleted_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

// Purge 物理删除人物；顺序为体系引用 → 别名 → 人物，失败整体回滚。
func (r *Characters) Purge(ctx context.Context, id string) (int64, error) {
	var affected int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Exec("DELETE FROM character_system_refs WHERE character_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("character_id = ?", id).Delete(&model.CharacterName{}).Error; err != nil {
			return err
		}
		res := tx.WithContext(ctx).Where("id = ?", id).Delete(&model.Character{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, err
}

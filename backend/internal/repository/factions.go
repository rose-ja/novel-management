package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Factions 是势力树的数据访问层；事务场景通过 NewFactions(tx) 重建实例。
type Factions struct {
	DB *gorm.DB
}

func NewFactions(db *gorm.DB) *Factions { return &Factions{DB: db} }

// FactionUpdate 是一次完整字段写入，version 由调用方保证与 If-Match 一致。
type FactionUpdate struct {
	Name              string
	NameKey           string
	ParentID          *string
	LeaderCharacterID *string
	Description       *string
	Version           int
}

// FactionMemberRef 是删除阻断时列出的成员人物摘要。
type FactionMemberRef struct {
	ID   string `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

// Briefs 批量读取势力摘要（含回收站），供人物列表/详情展示。
func (r *Factions) Briefs(ctx context.Context, ids []string) (map[string]model.FactionBrief, error) {
	out := make(map[string]model.FactionBrief, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		ID   string
		Name string
	}
	var rows []row
	if err := r.DB.WithContext(ctx).Table("factions").
		Select("id, name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.ID] = model.FactionBrief{ID: item.ID, Name: item.Name}
	}
	return out, nil
}

// ListByLeader 返回以该人物为首领的势力（含回收站），用于删除人物入向阻断。
func (r *Factions) ListByLeader(ctx context.Context, characterID string) ([]model.Faction, error) {
	var items []model.Faction
	err := r.DB.WithContext(ctx).
		Where("leader_character_id = ?", characterID).
		Order("name_key ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Faction{}
	}
	return items, nil
}

// ListActive 返回作品下全部未删除势力，按同级稳定顺序（name_key, id）。
func (r *Factions) ListActive(ctx context.Context, workID, reviewDB string) ([]model.Faction, error) {
	tx := r.DB.WithContext(ctx).
		Where("work_id = ? AND deleted_at IS NULL", workID)
	if reviewDB != "" {
		tx = tx.Where("review_status = ?", reviewDB)
	}
	var items []model.Faction
	err := tx.Order("name_key ASC, id ASC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Faction{}
	}
	return items, nil
}

// Get 返回未删除势力；不存在或已软删除统一 ErrNotFound。
func (r *Factions) Get(ctx context.Context, id string) (*model.Faction, error) {
	return r.get(ctx, "id = ? AND deleted_at IS NULL", id)
}

// GetActiveByWork 返回指定作品下未删除势力。
func (r *Factions) GetActiveByWork(ctx context.Context, workID, id string) (*model.Faction, error) {
	return r.get(ctx, "id = ? AND work_id = ? AND deleted_at IS NULL", id, workID)
}

func (r *Factions) get(ctx context.Context, where string, args ...any) (*model.Faction, error) {
	var item model.Faction
	err := r.DB.WithContext(ctx).Where(where, args...).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetForUpdate 在事务内锁定未删除势力行，供乐观锁读取。
func (r *Factions) GetForUpdate(ctx context.Context, id string) (*model.Faction, error) {
	var item model.Faction
	err := r.DB.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetDeleted 返回直接删除进回收站的势力根（不含随作品删除的势力）。
func (r *Factions) GetDeleted(ctx context.Context, id string) (*model.Faction, error) {
	return r.get(ctx, "id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id)
}

// GetAny 返回任意状态（含软删）的势力，用于区分跨作品与已删引用。
func (r *Factions) GetAny(ctx context.Context, id string) (*model.Faction, error) {
	return r.get(ctx, "id = ?", id)
}

func (r *Factions) Insert(ctx context.Context, item *model.Faction) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// NameTaken 判断作品下是否有同名未删除势力。
func (r *Factions) NameTaken(ctx context.Context, workID, nameKey, excludeID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Table("factions").
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

// ActiveByName 返回同名活跃势力，用于 NAME_CONFLICT details。
func (r *Factions) ActiveByName(ctx context.Context, workID, nameKey, excludeID string) (*model.Faction, error) {
	tx := r.DB.WithContext(ctx).
		Where("work_id = ? AND name_key = ? AND deleted_at IS NULL", workID, nameKey)
	if excludeID != "" {
		tx = tx.Where("id <> ?", excludeID)
	}
	var item model.Faction
	err := tx.First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateWithVersion 使用 version 做乐观锁；返回受影响行数（0 表示版本冲突或已删除）。
func (r *Factions) UpdateWithVersion(ctx context.Context, id string, update FactionUpdate, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("factions").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, update.Version).
		Updates(map[string]any{
			"name":                update.Name,
			"name_key":            update.NameKey,
			"parent_id":           update.ParentID,
			"leader_character_id": update.LeaderCharacterID,
			"description":         update.Description,
			"version":             update.Version + 1,
			"updated_at":          now,
		})
	return res.RowsAffected, res.Error
}

// IsAncestor 判断 nodeID 是否为 candidateID 自身或祖先（含深层），用于移动环检测。
func (r *Factions) IsAncestor(ctx context.Context, nodeID, candidateID string) (bool, error) {
	if nodeID == "" || candidateID == "" {
		return false, nil
	}
	var matched bool
	err := r.DB.WithContext(ctx).Raw(`
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM factions WHERE id = ?
			UNION
			SELECT f.id, f.parent_id FROM factions f JOIN ancestors a ON f.id = a.parent_id
		)
		SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = ?)`,
		candidateID, nodeID,
	).Scan(&matched).Error
	if err != nil {
		return false, err
	}
	return matched, nil
}

// SubtreeIDs 返回势力自身及全部活跃下级 ID（递归去重）。
func (r *Factions) SubtreeIDs(ctx context.Context, workID, factionID string) ([]string, error) {
	var ids []string
	err := r.DB.WithContext(ctx).Raw(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM factions WHERE id = ? AND work_id = ? AND deleted_at IS NULL
			UNION
			SELECT f.id FROM factions f JOIN subtree s ON f.parent_id = s.id
			 WHERE f.deleted_at IS NULL
		)
		SELECT id FROM subtree`, factionID, workID).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

// ListActiveChildren 返回直接下级（仅活跃），按 name_key, id 稳定排序。
func (r *Factions) ListActiveChildren(ctx context.Context, parentID string) ([]model.Faction, error) {
	var items []model.Faction
	err := r.DB.WithContext(ctx).
		Where("parent_id = ? AND deleted_at IS NULL", parentID).
		Order("name_key ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Faction{}
	}
	return items, nil
}

// ListChildRows 返回任意状态（含软删）的直接下级，用于删除阻断（设计 D7）。
func (r *Factions) ListChildRows(ctx context.Context, parentID string) ([]model.Faction, error) {
	var items []model.Faction
	err := r.DB.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("name_key ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Faction{}
	}
	return items, nil
}

// CountActiveChildren 返回作品下各势力的活跃直接下级数量（parent_id → count）。
func (r *Factions) CountActiveChildren(ctx context.Context, workID string) (map[string]int64, error) {
	type row struct {
		Key   string
		Total int64
	}
	var rows []row
	err := r.DB.WithContext(ctx).Model(&model.Faction{}).
		Select("parent_id AS key, count(*) AS total").
		Where("work_id = ? AND deleted_at IS NULL AND parent_id IS NOT NULL", workID).
		Group("parent_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, item := range rows {
		out[item.Key] = item.Total
	}
	return out, nil
}

// CountActiveMembers 返回作品下各势力的活跃直属成员数量（faction_id → count）。
func (r *Factions) CountActiveMembers(ctx context.Context, workID string) (map[string]int64, error) {
	type row struct {
		Key   string
		Total int64
	}
	var rows []row
	err := r.DB.WithContext(ctx).Model(&model.Character{}).
		Select("faction_id AS key, count(*) AS total").
		Where("work_id = ? AND deleted_at IS NULL AND faction_id IS NOT NULL", workID).
		Group("faction_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, item := range rows {
		out[item.Key] = item.Total
	}
	return out, nil
}

// CountActiveMembersOne 返回单个势力的活跃直属成员数量。
func (r *Factions) CountActiveMembersOne(ctx context.Context, factionID string) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Model(&model.Character{}).
		Where("faction_id = ? AND deleted_at IS NULL", factionID).
		Count(&total).Error
	return total, err
}

// CountSubtreeMembers 返回子树内活跃成员总数（递归去重）。
func (r *Factions) CountSubtreeMembers(ctx context.Context, workID, factionID string) (int64, error) {
	ids, err := r.SubtreeIDs(ctx, workID, factionID)
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	var total int64
	err = r.DB.WithContext(ctx).Model(&model.Character{}).
		Where("work_id = ? AND deleted_at IS NULL AND faction_id IN ?", workID, ids).
		Count(&total).Error
	return total, err
}

// ActiveMembers 返回仍直接归属该势力的活跃人物，用于删除阻断。
func (r *Factions) ActiveMembers(ctx context.Context, factionID string) ([]FactionMemberRef, error) {
	var items []FactionMemberRef
	err := r.DB.WithContext(ctx).Table("characters").
		Select("id, name").
		Where("faction_id = ? AND deleted_at IS NULL", factionID).
		Order("name_key ASC, id ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []FactionMemberRef{}
	}
	return items, nil
}

// LeaderNames 返回首领人物名称（含回收站），供列表批量装配，避免 N+1。
func (r *Factions) LeaderNames(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		ID   string
		Name string
	}
	var rows []row
	if err := r.DB.WithContext(ctx).Table("characters").
		Select("id, name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.ID] = item.Name
	}
	return out, nil
}

// ListActiveMembers 返回直属成员分页；排序与人物列表一致（name_key, id）。
func (r *Factions) ListActiveMembers(ctx context.Context, workID, factionID string, page, pageSize int) ([]model.Character, int64, error) {
	build := func() *gorm.DB {
		return r.DB.WithContext(ctx).Model(&model.Character{}).
			Where("work_id = ? AND faction_id = ? AND deleted_at IS NULL", workID, factionID)
	}
	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Character
	err := build().
		Order("name_key ASC, id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListSubtreeMembers 返回子树成员分页（递归含下级并去重）。
func (r *Factions) ListSubtreeMembers(ctx context.Context, workID, factionID string, page, pageSize int) ([]model.Character, int64, error) {
	ids, err := r.SubtreeIDs(ctx, workID, factionID)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []model.Character{}, 0, nil
	}
	build := func() *gorm.DB {
		return r.DB.WithContext(ctx).Model(&model.Character{}).
			Where("work_id = ? AND deleted_at IS NULL AND faction_id IN ?", workID, ids)
	}
	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Character
	err = build().
		Order("name_key ASC, id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// SoftDelete 软删除势力自身根；同事务隐藏势力大事记（子内容跟随根）。
func (r *Factions) SoftDelete(ctx context.Context, id, action string, version int, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("factions").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, version).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   id,
			"deleted_by_action": action,
			"version":           version + 1,
			"updated_at":        now,
		})
	if res.Error != nil || res.RowsAffected == 0 {
		return res.RowsAffected, res.Error
	}
	if err := NewChronicles(r.DB).MarkOwnerDeletedByType(ctx, model.ChronicleOwnerFaction, id, id, action, now); err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// RenameDeleted 在回收站内改名，供恢复冲突时用 new_name 规避。
func (r *Factions) RenameDeleted(ctx context.Context, id, name, nameKey string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("factions").
		Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id).
		Updates(map[string]any{
			"name":       name,
			"name_key":   nameKey,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

// Restore 恢复势力根并递增版本；同时恢复随根删除的势力大事记。
func (r *Factions) Restore(ctx context.Context, id string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("factions").
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
	return NewChronicles(r.DB).RestoreByRoot(ctx, id, model.FactionActionDelete, now)
}

// RestoreGroupByWork 恢复随作品整组删除的势力。
func (r *Factions) RestoreGroupByWork(ctx context.Context, workID string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("factions").
		Where("deleted_root_id = ?", workID).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"version":           gorm.Expr("version + 1"),
			"updated_at":        now,
		}).Error
}

// ListDeletedRoots 返回全部势力回收站根（按删除时间倒序）。
func (r *Factions) ListDeletedRoots(ctx context.Context) ([]model.Faction, error) {
	var items []model.Faction
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_root_id = id").
		Order("deleted_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

// Purge 物理删除势力及其大事记；事务内先锁定并复核回收站根。
func (r *Factions) Purge(ctx context.Context, id string) (int64, error) {
	var affected int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked model.Faction
		err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id).
			First(&locked).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			affected = 0
			return nil
		}
		if err != nil {
			return err
		}
		if err := purgeFactionChronicles(ctx, tx, id); err != nil {
			return err
		}
		// 已软删除的人物仍可能引用该势力（外键 RESTRICT），永久删除前解除出向引用。
		if err := tx.WithContext(ctx).Exec(
			"UPDATE characters SET faction_id = NULL WHERE faction_id = ? AND deleted_at IS NOT NULL", id,
		).Error; err != nil {
			return err
		}
		res := tx.WithContext(ctx).Where("id = ?", id).Delete(&model.Faction{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, err
}

// PurgeByWork 物理删除作品下全部势力：先删势力大事记，再按深度倒序删行，
// 避免自引用 FK RESTRICT 与 characters.faction_id 的约束顺序问题（设计 D9）。
func (r *Factions) PurgeByWork(ctx context.Context, workID string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Where("owner_type = ? AND work_id = ?", model.ChronicleOwnerFaction, workID).
			Delete(&model.Chronicle{}).Error; err != nil {
			return err
		}
		return purgeFactionsByWork(ctx, tx, workID)
	})
}

// PurgeByDeletedWorks 物理删除全部回收站作品的势力，供清空回收站使用。
func (r *Factions) PurgeByDeletedWorks(ctx context.Context) error {
	scope := "work_id IN (SELECT id FROM works WHERE deleted_at IS NOT NULL)"
	if err := r.DB.WithContext(ctx).
		Exec("DELETE FROM chronicles WHERE owner_type = 'faction' AND " + scope).Error; err != nil {
		return err
	}
	var workIDs []string
	if err := r.DB.WithContext(ctx).
		Raw("SELECT id FROM works WHERE deleted_at IS NOT NULL").Scan(&workIDs).Error; err != nil {
		return err
	}
	for _, workID := range workIDs {
		if err := purgeFactionsByWork(ctx, r.DB.WithContext(ctx), workID); err != nil {
			return err
		}
	}
	return nil
}

func purgeFactionChronicles(ctx context.Context, tx *gorm.DB, factionID string) error {
	return tx.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", model.ChronicleOwnerFaction, factionID).
		Delete(&model.Chronicle{}).Error
}

func purgeFactionsByWork(ctx context.Context, tx *gorm.DB, workID string) error {
	var rows []factionParentRow
	if err := tx.WithContext(ctx).Raw(
		"SELECT id, parent_id FROM factions WHERE work_id = ?", workID,
	).Scan(&rows).Error; err != nil {
		return err
	}
	for _, id := range orderFactionsForPurge(rows) {
		if err := tx.WithContext(ctx).Where("id = ?", id).Delete(&model.Faction{}).Error; err != nil {
			return err
		}
	}
	return nil
}

type factionParentRow struct {
	ID       string
	ParentID *string
}

// orderFactionsForPurge 保证子势力先于父势力删除，避免自引用 FK RESTRICT。
func orderFactionsForPurge(rows []factionParentRow) []string {
	if len(rows) < 2 {
		out := make([]string, 0, len(rows))
		for _, item := range rows {
			out = append(out, item.ID)
		}
		return out
	}
	children := make(map[string][]string, len(rows))
	for _, item := range rows {
		if item.ParentID == nil {
			continue
		}
		children[*item.ParentID] = append(children[*item.ParentID], item.ID)
	}
	visited := make(map[string]bool, len(rows))
	ordered := make([]string, 0, len(rows))
	var visit func(id string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, childID := range children[id] {
			visit(childID)
		}
		ordered = append(ordered, id)
	}
	for _, item := range rows {
		visit(item.ID)
	}
	return ordered
}

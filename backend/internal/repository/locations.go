package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Locations 是地点树与人物/地点/势力关系的数据访问层。
type Locations struct {
	DB *gorm.DB
}

func NewLocations(db *gorm.DB) *Locations { return &Locations{DB: db} }

// LocationUpdate 是一次完整字段写入，version 由调用方保证与 If-Match 一致。
type LocationUpdate struct {
	Name        string
	NameKey     string
	ParentID    *string
	TypeText    *string
	Description *string
	Version     int
}

// LocationCharacterRow 是地点详情/删除阻断使用的人物关系行。
type LocationCharacterRow struct {
	CharacterID string  `gorm:"column:character_id"`
	Name        string  `gorm:"column:name"`
	FactionID   *string `gorm:"column:faction_id"`
	InTrash     bool    `gorm:"column:in_trash"`
	Note        *string `gorm:"column:note"`
}

// LocationFactionRow 是地点详情/删除阻断使用的势力关系行。
type LocationFactionRow struct {
	FactionID string  `gorm:"column:faction_id"`
	Name      string  `gorm:"column:name"`
	InTrash   bool    `gorm:"column:in_trash"`
	Note      *string `gorm:"column:note"`
}

// FactionLocationRow 是势力详情使用的地点关系行。
type FactionLocationRow struct {
	LocationID string  `gorm:"column:location_id"`
	Name       string  `gorm:"column:name"`
	InTrash    bool    `gorm:"column:in_trash"`
	Note       *string `gorm:"column:note"`
}

// ListActive 返回作品下全部未删除地点，按 name_key, id 稳定排序。
func (r *Locations) ListActive(ctx context.Context, workID, reviewDB string) ([]model.Location, error) {
	tx := r.DB.WithContext(ctx).
		Where("work_id = ? AND deleted_at IS NULL", workID)
	if reviewDB != "" {
		tx = tx.Where("review_status = ?", reviewDB)
	}
	var items []model.Location
	err := tx.Order("name_key ASC, id ASC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Location{}
	}
	return items, nil
}

// Get 返回未删除地点。
func (r *Locations) Get(ctx context.Context, id string) (*model.Location, error) {
	return r.get(ctx, "id = ? AND deleted_at IS NULL", id)
}

// GetActiveByWork 返回指定作品下未删除地点。
func (r *Locations) GetActiveByWork(ctx context.Context, workID, id string) (*model.Location, error) {
	return r.get(ctx, "id = ? AND work_id = ? AND deleted_at IS NULL", id, workID)
}

func (r *Locations) get(ctx context.Context, where string, args ...any) (*model.Location, error) {
	var item model.Location
	err := r.DB.WithContext(ctx).Where(where, args...).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetActiveByWorkForShare 以 SHARE 锁读取活跃地点，与删除的 UPDATE 锁互斥，避免引用写入已删地点。
func (r *Locations) GetActiveByWorkForShare(ctx context.Context, workID, id string) (*model.Location, error) {
	var item model.Location
	err := r.DB.WithContext(ctx).
		Clauses(clause.Locking{Strength: "SHARE"}).
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

// GetForShare 以 SHARE 锁读取活跃地点（不限定作品），供关系写入前与删除互斥。
func (r *Locations) GetForShare(ctx context.Context, id string) (*model.Location, error) {
	var item model.Location
	err := r.DB.WithContext(ctx).
		Clauses(clause.Locking{Strength: "SHARE"}).
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

// GetForUpdate 在事务内锁定未删除地点行，供乐观锁读取。
func (r *Locations) GetForUpdate(ctx context.Context, id string) (*model.Location, error) {
	var item model.Location
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

// GetDeleted 返回直接删除进回收站的地点根（不含随作品删除的地点）。
func (r *Locations) GetDeleted(ctx context.Context, id string) (*model.Location, error) {
	return r.get(ctx, "id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id)
}

// GetDeletedForUpdate 在事务内锁定回收站地点根，供永久删除前复核。
func (r *Locations) GetDeletedForUpdate(ctx context.Context, id string) (*model.Location, error) {
	var item model.Location
	err := r.DB.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
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

// GetAny 返回任意状态（含软删）的地点，用于区分跨作品与已删引用。
func (r *Locations) GetAny(ctx context.Context, id string) (*model.Location, error) {
	return r.get(ctx, "id = ?", id)
}

func (r *Locations) Insert(ctx context.Context, item *model.Location) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// NameTaken 判断作品下是否有同名未删除地点。
func (r *Locations) NameTaken(ctx context.Context, workID, nameKey, excludeID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Table("locations").
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

// ActiveByName 返回同名活跃地点，用于 NAME_CONFLICT details。
func (r *Locations) ActiveByName(ctx context.Context, workID, nameKey, excludeID string) (*model.Location, error) {
	tx := r.DB.WithContext(ctx).
		Where("work_id = ? AND name_key = ? AND deleted_at IS NULL", workID, nameKey)
	if excludeID != "" {
		tx = tx.Where("id <> ?", excludeID)
	}
	var item model.Location
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
func (r *Locations) UpdateWithVersion(ctx context.Context, id string, update LocationUpdate, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("locations").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, update.Version).
		Updates(map[string]any{
			"name":        update.Name,
			"name_key":    update.NameKey,
			"parent_id":   update.ParentID,
			"type_text":   update.TypeText,
			"description": update.Description,
			"version":     update.Version + 1,
			"updated_at":  now,
		})
	return res.RowsAffected, res.Error
}

// IsAncestor 判断 nodeID 是否为 candidateID 自身或祖先（含深层），用于移动环检测。
func (r *Locations) IsAncestor(ctx context.Context, nodeID, candidateID string) (bool, error) {
	if nodeID == "" || candidateID == "" {
		return false, nil
	}
	var matched bool
	err := r.DB.WithContext(ctx).Raw(`
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM locations WHERE id = ?
			UNION
			SELECT l.id, l.parent_id FROM locations l JOIN ancestors a ON l.id = a.parent_id
		)
		SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = ?)`,
		candidateID, nodeID,
	).Scan(&matched).Error
	if err != nil {
		return false, err
	}
	return matched, nil
}

// ListActiveChildren 返回直接下级（仅活跃），按 name_key, id 稳定排序。
func (r *Locations) ListActiveChildren(ctx context.Context, parentID string) ([]model.Location, error) {
	var items []model.Location
	err := r.DB.WithContext(ctx).
		Where("parent_id = ? AND deleted_at IS NULL", parentID).
		Order("name_key ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Location{}
	}
	return items, nil
}

// ListChildRows 返回任意状态（含软删）的直接下级，用于删除阻断。
func (r *Locations) ListChildRows(ctx context.Context, parentID string) ([]model.Location, error) {
	var items []model.Location
	err := r.DB.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("name_key ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Location{}
	}
	return items, nil
}

// CountActiveChildren 返回作品下各地点的活跃直接下级数量（parent_id → count）。
func (r *Locations) CountActiveChildren(ctx context.Context, workID string) (map[string]int64, error) {
	return r.countBy(ctx, "locations", workID,
		"parent_id IS NOT NULL AND deleted_at IS NULL", "parent_id")
}

// CountActiveCharacters 返回作品下各地点的活跃人物关系数（location_id → count）。
func (r *Locations) CountActiveCharacters(ctx context.Context, workID string) (map[string]int64, error) {
	type row struct {
		Key   string
		Total int64
	}
	var rows []row
	err := r.DB.WithContext(ctx).Table("character_locations AS cl").
		Select("cl.location_id AS key, count(*) AS total").
		Joins("JOIN characters c ON c.id = cl.character_id AND c.deleted_at IS NULL").
		Where("cl.work_id = ?", workID).
		Group("cl.location_id").
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

// CountActiveFactions 返回作品下各地点的活跃势力关系数（location_id → count）。
func (r *Locations) CountActiveFactions(ctx context.Context, workID string) (map[string]int64, error) {
	type row struct {
		Key   string
		Total int64
	}
	var rows []row
	err := r.DB.WithContext(ctx).Table("location_factions AS lf").
		Select("lf.location_id AS key, count(*) AS total").
		Joins("JOIN factions f ON f.id = lf.faction_id AND f.deleted_at IS NULL").
		Where("lf.work_id = ?", workID).
		Group("lf.location_id").
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

func (r *Locations) countBy(ctx context.Context, table, workID, where, key string) (map[string]int64, error) {
	type row struct {
		Key   string
		Total int64
	}
	var rows []row
	err := r.DB.WithContext(ctx).Table(table).
		Select(key+" AS key, count(*) AS total").
		Where("work_id = ? AND "+where, workID).
		Group(key).
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

// ParentBriefs 返回父级摘要（含已删），供列表/详情装配。
func (r *Locations) ParentBriefs(ctx context.Context, ids []string) (map[string]model.LocationBrief, error) {
	out := make(map[string]model.LocationBrief, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		ID   string
		Name string
	}
	var rows []row
	if err := r.DB.WithContext(ctx).Table("locations").
		Select("id, name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.ID] = model.LocationBrief{ID: item.ID, Name: item.Name}
	}
	return out, nil
}

// Briefs 批量读取地点摘要（含回收站），供人物详情展示。
func (r *Locations) Briefs(ctx context.Context, ids []string) (map[string]model.LocationBrief, error) {
	return r.ParentBriefs(ctx, ids)
}

// ListCharacterLocations 返回人物全部地点关系（含地点状态），按地点 name_key 排序。
func (r *Locations) ListCharacterLocations(ctx context.Context, characterID string) ([]model.CharacterLocation, error) {
	var items []model.CharacterLocation
	err := r.DB.WithContext(ctx).
		Where("character_id = ?", characterID).
		Order("location_id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.CharacterLocation{}
	}
	return items, nil
}

// ReplaceCharacterLocations 整组替换人物地点关系（调用方负责事务）。
func (r *Locations) ReplaceCharacterLocations(ctx context.Context, characterID string, links []model.CharacterLocation) error {
	if err := r.DB.WithContext(ctx).
		Where("character_id = ?", characterID).
		Delete(&model.CharacterLocation{}).Error; err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Create(&links).Error
}

// CharacterRowsForLocation 返回地点的人物关系行（含回收站人物），按姓名稳定排序。
func (r *Locations) CharacterRowsForLocation(ctx context.Context, locationID string) ([]LocationCharacterRow, error) {
	var rows []LocationCharacterRow
	err := r.DB.WithContext(ctx).Table("character_locations AS cl").
		Select("c.id AS character_id, c.name, c.faction_id, (c.deleted_at IS NOT NULL) AS in_trash, cl.note").
		Joins("JOIN characters c ON c.id = cl.character_id").
		Where("cl.location_id = ?", locationID).
		Order("c.name_key ASC, c.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []LocationCharacterRow{}
	}
	return rows, nil
}

// FactionRowsForLocation 返回地点的势力关系行（含回收站势力）。
func (r *Locations) FactionRowsForLocation(ctx context.Context, locationID string) ([]LocationFactionRow, error) {
	var rows []LocationFactionRow
	err := r.DB.WithContext(ctx).Table("location_factions AS lf").
		Select("f.id AS faction_id, f.name, (f.deleted_at IS NOT NULL) AS in_trash, lf.note").
		Joins("JOIN factions f ON f.id = lf.faction_id").
		Where("lf.location_id = ?", locationID).
		Order("f.name_key ASC, f.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []LocationFactionRow{}
	}
	return rows, nil
}

// LocationRowsForFaction 返回势力的地点关系行（含回收站地点）。
func (r *Locations) LocationRowsForFaction(ctx context.Context, factionID string) ([]FactionLocationRow, error) {
	var rows []FactionLocationRow
	err := r.DB.WithContext(ctx).Table("location_factions AS lf").
		Select("l.id AS location_id, l.name, (l.deleted_at IS NOT NULL) AS in_trash, lf.note").
		Joins("JOIN locations l ON l.id = lf.location_id").
		Where("lf.faction_id = ?", factionID).
		Order("l.name_key ASC, l.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []FactionLocationRow{}
	}
	return rows, nil
}

// InsertLocationFaction 新增地点-势力关系。
func (r *Locations) InsertLocationFaction(ctx context.Context, item *model.LocationFaction) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// HasLocationFaction 判断地点-势力关系是否存在。
func (r *Locations) HasLocationFaction(ctx context.Context, locationID, factionID string) (bool, error) {
	var total int64
	err := r.DB.WithContext(ctx).Model(&model.LocationFaction{}).
		Where("location_id = ? AND faction_id = ?", locationID, factionID).
		Count(&total).Error
	return total > 0, err
}

// UpdateLocationFactionNote 仅更新关系备注。
func (r *Locations) UpdateLocationFactionNote(ctx context.Context, locationID, factionID string, note *string, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Model(&model.LocationFaction{}).
		Where("location_id = ? AND faction_id = ?", locationID, factionID).
		Updates(map[string]any{"note": note, "updated_at": now})
	return res.RowsAffected, res.Error
}

// DeleteLocationFaction 删除单条地点-势力关系。
func (r *Locations) DeleteLocationFaction(ctx context.Context, locationID, factionID string) (int64, error) {
	res := r.DB.WithContext(ctx).
		Where("location_id = ? AND faction_id = ?", locationID, factionID).
		Delete(&model.LocationFaction{})
	return res.RowsAffected, res.Error
}

// SoftDelete 软删除地点自身根。
func (r *Locations) SoftDelete(ctx context.Context, id, action string, version int, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("locations").
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

// RenameDeleted 在回收站内改名，供恢复冲突时用 new_name 规避。
func (r *Locations) RenameDeleted(ctx context.Context, id, name, nameKey string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("locations").
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

// Restore 恢复地点根并递增版本。
func (r *Locations) Restore(ctx context.Context, id string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("locations").
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

// RestoreGroupByWork 恢复随作品整组删除的地点。
func (r *Locations) RestoreGroupByWork(ctx context.Context, workID string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("locations").
		Where("deleted_root_id = ?", workID).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"version":           gorm.Expr("version + 1"),
			"updated_at":        now,
		}).Error
}

// ListDeletedRoots 返回全部地点回收站根（按删除时间倒序）。
func (r *Locations) ListDeletedRoots(ctx context.Context) ([]model.Location, error) {
	var items []model.Location
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_root_id = id").
		Order("deleted_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

// Purge 物理删除地点及其势力关系；调用方需先确认无人物关系。
func (r *Locations) Purge(ctx context.Context, id string) (int64, error) {
	var affected int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked model.Location
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
		if err := tx.WithContext(ctx).Where("location_id = ?", id).
			Delete(&model.LocationFaction{}).Error; err != nil {
			return err
		}
		res := tx.WithContext(ctx).Where("id = ?", id).Delete(&model.Location{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, err
}

// PurgeByWork 物理删除作品下全部地点：先删势力关系，再按深度倒序删行。
func (r *Locations) PurgeByWork(ctx context.Context, workID string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("work_id = ?", workID).
			Delete(&model.LocationFaction{}).Error; err != nil {
			return err
		}
		return purgeLocationsByWork(ctx, tx, workID)
	})
}

// PurgeByDeletedWorks 物理删除全部回收站作品的地点，供清空回收站使用。
func (r *Locations) PurgeByDeletedWorks(ctx context.Context) error {
	var workIDs []string
	if err := r.DB.WithContext(ctx).
		Raw("SELECT id FROM works WHERE deleted_at IS NOT NULL").Scan(&workIDs).Error; err != nil {
		return err
	}
	for _, workID := range workIDs {
		if err := r.PurgeByWork(ctx, workID); err != nil {
			return err
		}
	}
	return nil
}

func purgeLocationsByWork(ctx context.Context, tx *gorm.DB, workID string) error {
	var rows []locationParentRow
	if err := tx.WithContext(ctx).Raw(
		"SELECT id, parent_id FROM locations WHERE work_id = ?", workID,
	).Scan(&rows).Error; err != nil {
		return err
	}
	for _, id := range orderLocationsForPurge(rows) {
		if err := tx.WithContext(ctx).Where("id = ?", id).Delete(&model.Location{}).Error; err != nil {
			return err
		}
	}
	return nil
}

type locationParentRow struct {
	ID       string
	ParentID *string
}

// orderLocationsForPurge 保证子地点先于父地点删除，避免自引用 FK RESTRICT。
func orderLocationsForPurge(rows []locationParentRow) []string {
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

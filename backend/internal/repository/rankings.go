package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Rankings 是榜单与固定席位的数据访问层。
type Rankings struct {
	DB *gorm.DB
}

func NewRankings(db *gorm.DB) *Rankings { return &Rankings{DB: db} }

// RankingUpdate 是一次整体字段写入，version 由调用方保证与 If-Match 一致。
type RankingUpdate struct {
	Name      string
	NameKey   string
	SeatCount int
	Version   int
}

// SeatOccupantRow 是缩容阻断使用的占用行。
type SeatOccupantRow struct {
	SeatNo      int    `gorm:"column:seat_no"`
	CharacterID string `gorm:"column:character_id"`
	Name        string `gorm:"column:name"`
}

// CharacterRankingRow 是人物席位引用行（含回收站榜单）。
type CharacterRankingRow struct {
	RankingID string  `gorm:"column:ranking_id"`
	Name      string  `gorm:"column:name"`
	SeatNo    int     `gorm:"column:seat_no"`
	Note      *string `gorm:"column:note"`
	InTrash   bool    `gorm:"column:in_trash"`
}

// CharacterRankingBrief 是人物列表/详情名次摘要行。
type CharacterRankingBrief struct {
	CharacterID string  `gorm:"column:character_id"`
	RankingID   string  `gorm:"column:ranking_id"`
	Name        string  `gorm:"column:name"`
	SeatNo      int     `gorm:"column:seat_no"`
	Note        *string `gorm:"column:note"`
	Total       int     `gorm:"column:total"`
}

// Get 返回未删除榜单。
func (r *Rankings) Get(ctx context.Context, id string) (*model.Ranking, error) {
	return r.get(ctx, "id = ? AND deleted_at IS NULL", id)
}

// GetForUpdate 在事务内锁定未删除榜单行。
func (r *Rankings) GetForUpdate(ctx context.Context, id string) (*model.Ranking, error) {
	return r.getLocked(ctx, "id = ? AND deleted_at IS NULL", id)
}

// GetDeleted 返回直接删除进回收站的榜单根（不含随作品删除的榜单）。
func (r *Rankings) GetDeleted(ctx context.Context, id string) (*model.Ranking, error) {
	return r.get(ctx, "id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id)
}

// GetDeletedForUpdate 在事务内锁定回收站榜单根，供永久删除复核。
func (r *Rankings) GetDeletedForUpdate(ctx context.Context, id string) (*model.Ranking, error) {
	return r.getLocked(ctx, "id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", id)
}

func (r *Rankings) get(ctx context.Context, where string, args ...any) (*model.Ranking, error) {
	var item model.Ranking
	err := r.DB.WithContext(ctx).Where(where, args...).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Rankings) getLocked(ctx context.Context, where string, args ...any) (*model.Ranking, error) {
	var item model.Ranking
	err := r.DB.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(where, args...).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ListActive 返回作品下未删除榜单，默认按更新时间倒序。
func (r *Rankings) ListActive(ctx context.Context, workID, q, reviewStatus, sort string, page, pageSize int) ([]model.Ranking, error) {
	tx := r.DB.WithContext(ctx).
		Where("work_id = ? AND deleted_at IS NULL", workID)
	if q != "" {
		tx = tx.Where("name ILIKE ?", "%"+q+"%")
	}
	if reviewStatus != "" {
		tx = tx.Where("review_status = ?", reviewStatus)
	}
	switch sort {
	case "name:asc":
		tx = tx.Order("name_key ASC, id ASC")
	default:
		tx = tx.Order("updated_at DESC, id DESC")
	}
	var items []model.Ranking
	if err := tx.Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Ranking{}
	}
	return items, nil
}

// CountActive 返回与 ListActive 同条件的总数。
func (r *Rankings) CountActive(ctx context.Context, workID, q, reviewStatus string) (int64, error) {
	tx := r.DB.WithContext(ctx).Table("rankings").
		Where("work_id = ? AND deleted_at IS NULL", workID)
	if q != "" {
		tx = tx.Where("name ILIKE ?", "%"+q+"%")
	}
	if reviewStatus != "" {
		tx = tx.Where("review_status = ?", reviewStatus)
	}
	var total int64
	err := tx.Count(&total).Error
	return total, err
}

// NameTaken 判断作品下是否有同名未删除榜单。
func (r *Rankings) NameTaken(ctx context.Context, workID, nameKey, excludeID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Table("rankings").
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

func (r *Rankings) Insert(ctx context.Context, item *model.Ranking) error {
	return r.DB.WithContext(ctx).Create(item).Error
}

// UpdateWithVersion 使用 version 做乐观锁；返回受影响行数（0 表示版本冲突或已删除）。
func (r *Rankings) UpdateWithVersion(ctx context.Context, id string, update RankingUpdate, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("rankings").
		Where("id = ? AND deleted_at IS NULL AND version = ?", id, update.Version).
		Updates(map[string]any{
			"name":       update.Name,
			"name_key":   update.NameKey,
			"seat_count": update.SeatCount,
			"version":    update.Version + 1,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// SoftDelete 软删除榜单根；version > 0 时附带乐观锁，0 表示不带版本条件。
func (r *Rankings) SoftDelete(ctx context.Context, id, action string, version int, now time.Time) (int64, error) {
	tx := r.DB.WithContext(ctx).Table("rankings").
		Where("id = ? AND deleted_at IS NULL", id)
	if version > 0 {
		tx = tx.Where("version = ?", version)
	}
	res := tx.Updates(map[string]any{
		"deleted_at":        now,
		"deleted_root_id":   id,
		"deleted_by_action": action,
		"version":           gorm.Expr("version + 1"),
		"updated_at":        now,
	})
	return res.RowsAffected, res.Error
}

// Restore 恢复榜单根并递增版本；席位行原样保留。
func (r *Rankings) Restore(ctx context.Context, id string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("rankings").
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

// RenameDeleted 在回收站内改名，供恢复冲突时用 new_name 规避。
func (r *Rankings) RenameDeleted(ctx context.Context, id, name, nameKey string, now time.Time) error {
	res := r.DB.WithContext(ctx).Table("rankings").
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

// ListDeletedRoots 返回全部榜单回收站根（按删除时间倒序）。
func (r *Rankings) ListDeletedRoots(ctx context.Context) ([]model.Ranking, error) {
	var items []model.Ranking
	err := r.DB.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_root_id = id").
		Order("deleted_at DESC, id DESC").
		Find(&items).Error
	return items, err
}

// SeatsPage 返回席位分页（seat_no 升序）；occupied 为 nil 时不过滤。
func (r *Rankings) SeatsPage(ctx context.Context, rankingID string, page, pageSize int, occupied *bool) ([]model.RankingSeat, error) {
	tx := r.DB.WithContext(ctx).Where("ranking_id = ?", rankingID)
	if occupied != nil {
		if *occupied {
			tx = tx.Where("character_id IS NOT NULL")
		} else {
			tx = tx.Where("character_id IS NULL")
		}
	}
	var items []model.RankingSeat
	if err := tx.Order("seat_no ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.RankingSeat{}
	}
	return items, nil
}

// CountSeats 统计席位总数；occupied 为 nil 时返回全部席位。
func (r *Rankings) CountSeats(ctx context.Context, rankingID string, occupied *bool) (int64, error) {
	tx := r.DB.WithContext(ctx).Table("ranking_seats").Where("ranking_id = ?", rankingID)
	if occupied != nil {
		if *occupied {
			tx = tx.Where("character_id IS NOT NULL")
		} else {
			tx = tx.Where("character_id IS NULL")
		}
	}
	var total int64
	err := tx.Count(&total).Error
	return total, err
}

// SeatByNo 返回指定席位。
func (r *Rankings) SeatByNo(ctx context.Context, rankingID string, seatNo int) (*model.RankingSeat, error) {
	return r.seat(ctx, false, rankingID, seatNo)
}

// SeatByNoForUpdate 在事务内锁定指定席位。
func (r *Rankings) SeatByNoForUpdate(ctx context.Context, rankingID string, seatNo int) (*model.RankingSeat, error) {
	return r.seat(ctx, true, rankingID, seatNo)
}

func (r *Rankings) seat(ctx context.Context, lock bool, rankingID string, seatNo int) (*model.RankingSeat, error) {
	tx := r.DB.WithContext(ctx)
	if lock {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var item model.RankingSeat
	err := tx.Where("ranking_id = ? AND seat_no = ?", rankingID, seatNo).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// InsertSeatsRange 幂等追加 [from, to] 空席；已存在的席位号跳过。
func (r *Rankings) InsertSeatsRange(ctx context.Context, rankingID, workID string, from, to int) error {
	if from > to {
		return nil
	}
	seats := make([]model.RankingSeat, 0, to-from+1)
	for seatNo := from; seatNo <= to; seatNo++ {
		seats = append(seats, model.RankingSeat{
			ID:        model.NewID(),
			WorkID:    workID,
			RankingID: rankingID,
			SeatNo:    seatNo,
			Version:   1,
		})
	}
	return r.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(seats, 100).Error
}

// UpdateSeatWithVersion 使用席位 version 做乐观锁；返回 0 表示版本冲突或席位不存在。
func (r *Rankings) UpdateSeatWithVersion(ctx context.Context, rankingID string, seatNo, version int, characterID *string, note *string, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).Table("ranking_seats").
		Where("ranking_id = ? AND seat_no = ? AND version = ?", rankingID, seatNo, version).
		Updates(map[string]any{
			"character_id": characterID,
			"note":         note,
			"version":      version + 1,
			"updated_at":   now,
		})
	return res.RowsAffected, res.Error
}

// ClearSeatCharacter 清空席位人物与备注（调用方负责 version 校验）。
func (r *Rankings) ClearSeatCharacter(ctx context.Context, rankingID string, seatNo, version int, now time.Time) (int64, error) {
	return r.UpdateSeatWithVersion(ctx, rankingID, seatNo, version, nil, nil, now)
}

// SetSeatCharacter 无条件回写席位人物，供交换第二阶段使用。
func (r *Rankings) SetSeatCharacter(ctx context.Context, rankingID string, seatNo int, characterID string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("ranking_seats").
		Where("ranking_id = ? AND seat_no = ?", rankingID, seatNo).
		Updates(map[string]any{
			"character_id": characterID,
			"version":      gorm.Expr("version + 1"),
			"updated_at":   now,
		}).Error
}

// ClearSeatCharacters 先清空多个席位的人物（保留备注），供交换规避部分唯一索引。
func (r *Rankings) ClearSeatCharacters(ctx context.Context, rankingID string, seatNos []int, now time.Time) error {
	if len(seatNos) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Table("ranking_seats").
		Where("ranking_id = ? AND seat_no IN ?", rankingID, seatNos).
		Updates(map[string]any{
			"character_id": nil,
			"version":      gorm.Expr("version + 1"),
			"updated_at":   now,
		}).Error
}

// DeleteSeatsAbove 物理删除席位号大于 target 的尾部座椅（调用方先确认均为空席）。
func (r *Rankings) DeleteSeatsAbove(ctx context.Context, rankingID string, target int) error {
	return r.DB.WithContext(ctx).
		Where("ranking_id = ? AND seat_no > ?", rankingID, target).
		Delete(&model.RankingSeat{}).Error
}

// OccupiedAbove 返回席位号大于 target 的占用明细，按席位号升序。
func (r *Rankings) OccupiedAbove(ctx context.Context, rankingID string, target int) ([]SeatOccupantRow, error) {
	var rows []SeatOccupantRow
	err := r.DB.WithContext(ctx).Table("ranking_seats AS rs").
		Select("rs.seat_no, rs.character_id, c.name").
		Joins("JOIN characters c ON c.id = rs.character_id").
		Where("rs.ranking_id = ? AND rs.seat_no > ? AND rs.character_id IS NOT NULL", rankingID, target).
		Order("rs.seat_no ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []SeatOccupantRow{}
	}
	return rows, nil
}

// ConflictSeatNo 返回人物在本榜已占用的席位号（排除 excludeSeat）；无冲突返回 nil。
func (r *Rankings) ConflictSeatNo(ctx context.Context, rankingID, characterID string, excludeSeat int) (*int, error) {
	var seatNo int
	err := r.DB.WithContext(ctx).Table("ranking_seats").
		Select("seat_no").
		Where("ranking_id = ? AND character_id = ? AND seat_no <> ?", rankingID, characterID, excludeSeat).
		Limit(1).
		Scan(&seatNo).Error
	if err != nil {
		return nil, err
	}
	if seatNo == 0 {
		return nil, nil
	}
	return &seatNo, nil
}

// SeatsForCharacter 返回人物全部席位引用（含回收站榜单），供删除阻断。
func (r *Rankings) SeatsForCharacter(ctx context.Context, characterID string) ([]CharacterRankingRow, error) {
	var rows []CharacterRankingRow
	err := r.DB.WithContext(ctx).Table("ranking_seats AS rs").
		Select("r.id AS ranking_id, r.name, rs.seat_no, rs.note, (r.deleted_at IS NOT NULL) AS in_trash").
		Joins("JOIN rankings r ON r.id = rs.ranking_id").
		Where("rs.character_id = ?", characterID).
		Order("r.name_key ASC, r.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []CharacterRankingRow{}
	}
	return rows, nil
}

// OccupiedCounts 批量统计各榜单已占席位数。
func (r *Rankings) OccupiedCounts(ctx context.Context, rankingIDs []string) (map[string]int64, error) {
	out := make(map[string]int64, len(rankingIDs))
	if len(rankingIDs) == 0 {
		return out, nil
	}
	type row struct {
		RankingID string
		Total     int64
	}
	var rows []row
	err := r.DB.WithContext(ctx).Table("ranking_seats").
		Select("ranking_id, count(*) AS total").
		Where("ranking_id IN ? AND character_id IS NOT NULL", rankingIDs).
		Group("ranking_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.RankingID] = item.Total
	}
	return out, nil
}

// TopSeats 批量返回各榜单前 limit 个席位（seat_no 升序，空席 character 为 null）。
func (r *Rankings) TopSeats(ctx context.Context, rankingIDs []string, limit int) (map[string][]model.RankingTopSeat, error) {
	out := make(map[string][]model.RankingTopSeat, len(rankingIDs))
	if len(rankingIDs) == 0 {
		return out, nil
	}
	type row struct {
		RankingID   string
		SeatNo      int
		CharacterID *string
		Name        *string
	}
	var rows []row
	err := r.DB.WithContext(ctx).Raw(`
		SELECT ranking_id, seat_no, character_id, name FROM (
			SELECT rs.ranking_id, rs.seat_no, rs.character_id, c.name,
			       row_number() OVER (PARTITION BY rs.ranking_id ORDER BY rs.seat_no ASC) AS rn
			FROM ranking_seats rs
			LEFT JOIN characters c ON c.id = rs.character_id
			WHERE rs.ranking_id IN ?
		) ranked
		WHERE rn <= ?
		ORDER BY ranking_id ASC, seat_no ASC`, rankingIDs, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		top := model.RankingTopSeat{SeatNo: item.SeatNo}
		if item.CharacterID != nil && item.Name != nil {
			top.Character = &model.RankingTopCharacter{ID: *item.CharacterID, Name: *item.Name}
		}
		out[item.RankingID] = append(out[item.RankingID], top)
	}
	return out, nil
}

// RankingBriefsByCharacters 批量返回人物名次摘要（每人物最多 limit 条）与总榜单数。
func (r *Rankings) RankingBriefsByCharacters(ctx context.Context, characterIDs []string, limit int) ([]CharacterRankingBrief, error) {
	if len(characterIDs) == 0 {
		return []CharacterRankingBrief{}, nil
	}
	var rows []CharacterRankingBrief
	err := r.DB.WithContext(ctx).Raw(`
		SELECT character_id, ranking_id, name, seat_no, note, total FROM (
			SELECT rs.character_id, r.id AS ranking_id, r.name, rs.seat_no, rs.note,
			       row_number() OVER (PARTITION BY rs.character_id ORDER BY r.name_key ASC, r.id ASC) AS rn,
			       count(*) OVER (PARTITION BY rs.character_id) AS total
			FROM ranking_seats rs
			JOIN rankings r ON r.id = rs.ranking_id
			WHERE rs.character_id IN ? AND r.deleted_at IS NULL
		) ranked
		WHERE rn <= ?
		ORDER BY character_id ASC, ranking_id ASC`, characterIDs, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []CharacterRankingBrief{}
	}
	return rows, nil
}

// MarkWorkDeleted 随作品整组软删除榜单。
func (r *Rankings) MarkWorkDeleted(ctx context.Context, workID, rootID, action string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("rankings").
		Where("work_id = ? AND deleted_at IS NULL", workID).
		Updates(map[string]any{
			"deleted_at":        now,
			"deleted_root_id":   rootID,
			"deleted_by_action": action,
			"version":           gorm.Expr("version + 1"),
			"updated_at":        now,
		}).Error
}

// RestoreByWork 恢复随作品整组删除的榜单。
func (r *Rankings) RestoreByWork(ctx context.Context, workID string, now time.Time) error {
	return r.DB.WithContext(ctx).Table("rankings").
		Where("work_id = ? AND deleted_root_id = ?", workID, workID).
		Updates(map[string]any{
			"deleted_at":        nil,
			"deleted_root_id":   nil,
			"deleted_by_action": nil,
			"version":           gorm.Expr("version + 1"),
			"updated_at":        now,
		}).Error
}

// Purge 物理删除榜单及全部席位；仅处理直接进回收站的根。
func (r *Rankings) Purge(ctx context.Context, id string) (int64, error) {
	var affected int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked model.Ranking
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
		if err := tx.WithContext(ctx).Where("ranking_id = ?", id).
			Delete(&model.RankingSeat{}).Error; err != nil {
			return err
		}
		res := tx.WithContext(ctx).Where("id = ?", id).Delete(&model.Ranking{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, err
}

// PurgeByWork 物理删除作品下全部榜单与席位。
func (r *Rankings) PurgeByWork(ctx context.Context, workID string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("work_id = ?", workID).
			Delete(&model.RankingSeat{}).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Where("work_id = ?", workID).
			Delete(&model.Ranking{}).Error
	})
}

// PurgeByDeletedWorks 物理删除全部回收站作品的榜单与席位。
func (r *Rankings) PurgeByDeletedWorks(ctx context.Context) error {
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

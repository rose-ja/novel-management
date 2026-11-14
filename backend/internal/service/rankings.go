package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

// RootTypeRanking 是回收站中榜单根的标识。
const RootTypeRanking = "ranking"

const (
	// RankingsDefaultSeatPageSize / RankingsMaxSeatPageSize 是席位分页只允许的档位。
	RankingsDefaultSeatPageSize = 50
	RankingsMaxSeatPageSize     = 100
	// RankingTopSeatLimit 是列表页前三席展示数量。
	RankingTopSeatLimit = 3
	// RankingDisplayLimit 是人物列表/成员表最多回显的榜单名次数量。
	RankingDisplayLimit = 2
)

// Rankings 负责榜单 CRUD、固定席位与容量调整。
type Rankings struct {
	db         *gorm.DB
	characters *Characters
	now        func() time.Time
}

func NewRankings(db *gorm.DB, characters *Characters) *Rankings {
	return &Rankings{db: db, characters: characters, now: time.Now}
}

type ListRankingsInput struct {
	WorkID       string
	Query        string
	ReviewStatus string
	Sort         string
	Page         int
	PageSize     int
}

type CreateRankingInput struct {
	WorkID    string
	Name      string
	SeatCount *int
}

type UpdateRankingInput struct {
	Name      *string
	SeatCount *int
	IfMatch   string
}

type DeleteRankingResult struct {
	Deleted   bool      `json:"deleted"`
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
	UndoToken string    `json:"undo_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PutSeatInput struct {
	CharacterID  *string
	CharacterSet bool
	Note         *string
	NoteSet      bool
	IfMatch      string
}

type MoveSeatInput struct {
	TargetSeatNo int
	Mode         string
	IfMatch      string
}

type MoveSeatResult struct {
	From model.SeatView `json:"from"`
	To   model.SeatView `json:"to"`
}

// RankingList 是列表响应。
type RankingList struct {
	Items    []model.RankingSummary `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// List 返回作品下的榜单分页，并装配已占/空席与前三席。
func (s *Rankings) List(ctx context.Context, in ListRankingsInput) (*RankingList, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	query := strings.TrimSpace(in.Query)
	if utf8.RuneCountInString(query) > 100 {
		return nil, &model.ValidationError{Field: "q", Message: "搜索词不能超过 100 个字符"}
	}
	reviewDB, err := model.DBReviewStatus(in.ReviewStatus)
	if err != nil {
		return nil, err
	}
	sort := strings.TrimSpace(in.Sort)
	switch sort {
	case "", "updated_at:desc", "name:asc":
	default:
		return nil, &model.ValidationError{Field: "sort", Message: "sort 仅支持 updated_at:desc 或 name:asc"}
	}
	page, pageSize := normalizePage(in.Page, in.PageSize)

	repo := repository.NewRankings(s.db)
	items, err := repo.ListActive(ctx, workID, query, reviewDB, sort, page, pageSize)
	if err != nil {
		return nil, err
	}
	total, err := repo.CountActive(ctx, workID, query, reviewDB)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	occupied, err := repo.OccupiedCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	tops, err := repo.TopSeats(ctx, ids, RankingTopSeatLimit)
	if err != nil {
		return nil, err
	}
	out := make([]model.RankingSummary, 0, len(items))
	for _, item := range items {
		out = append(out, rankingSummary(item, occupied[item.ID], tops[item.ID]))
	}
	return &RankingList{Items: out, Total: total, Page: page, PageSize: pageSize}, nil
}

// Create 新建榜单并在同一事务内生成 1..N 空席。
func (s *Rankings) Create(ctx context.Context, in CreateRankingInput) (*model.RankingDetail, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	name, nameKey, err := model.NormalizeRankingName(in.Name)
	if err != nil {
		return nil, err
	}
	seatCount := model.DefaultRankingSeats
	if in.SeatCount != nil {
		seatCount = *in.SeatCount
	}
	if err := model.NormalizeSeatCount(seatCount); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	item := &model.Ranking{
		ID:           model.NewID(),
		WorkID:       workID,
		Name:         name,
		NameKey:      nameKey,
		SeatCount:    seatCount,
		ReviewStatus: model.ReviewStatusReviewed,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := repository.NewWorks(tx).GetForUpdate(ctx, workID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrCharacterWorkUnavailable
			}
			return err
		}
		repo := repository.NewRankings(tx)
		taken, err := repo.NameTaken(ctx, workID, nameKey, "")
		if err != nil {
			return err
		}
		if taken {
			return newRankingNameConflict(nil, name)
		}
		if err := repo.Insert(ctx, item); err != nil {
			if repository.IsUniqueViolation(err, "rankings_active_name_uq") {
				return newRankingNameConflict(nil, name)
			}
			return err
		}
		return repo.InsertSeatsRange(ctx, item.ID, workID, 1, seatCount)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, item.ID, 1, RankingsDefaultSeatPageSize)
}

// Get 返回榜单摘要与席位分页；page_size 仅允许 50/100。
func (s *Rankings) Get(ctx context.Context, id string, page, pageSize int) (*model.RankingDetail, error) {
	page, pageSize, err := normalizeSeatPage(page, pageSize)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRankings(s.db)
	item, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	occupied, err := repo.CountSeats(ctx, id, boolPtr(true))
	if err != nil {
		return nil, err
	}
	seats, err := repo.SeatsPage(ctx, id, page, pageSize, nil)
	if err != nil {
		return nil, err
	}
	views, err := s.seatViews(ctx, seats)
	if err != nil {
		return nil, err
	}
	total, err := repo.CountSeats(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	return &model.RankingDetail{
		RankingSummary: rankingSummary(*item, occupied, nil),
		Seats: model.SeatPage{
			Items:    views,
			Page:     page,
			PageSize: pageSize,
			Total:    int(total),
		},
	}, nil
}

// Update 以内联字段子集更新名称与席位数；扩容追加空席，缩容仅允许尾部空席。
func (s *Rankings) Update(ctx context.Context, id string, in UpdateRankingInput) (*model.RankingDetail, error) {
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	now := s.now().UTC()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewRankings(tx)
		item, err := lockRanking(ctx, tx, id)
		if err != nil {
			return err
		}
		if item.Version != version {
			return &model.VersionConflictError{CurrentVersion: item.Version}
		}

		name, nameKey := item.Name, item.NameKey
		if in.Name != nil {
			if name, nameKey, err = model.NormalizeRankingName(*in.Name); err != nil {
				return err
			}
		}
		seatCount := item.SeatCount
		if in.SeatCount != nil {
			seatCount = *in.SeatCount
			if err := model.NormalizeSeatCount(seatCount); err != nil {
				return err
			}
		}
		if in.Name == nil && in.SeatCount == nil {
			return nil
		}
		if nameKey != item.NameKey {
			taken, err := repo.NameTaken(ctx, item.WorkID, nameKey, id)
			if err != nil {
				return err
			}
			if taken {
				return newRankingNameConflict(nil, name)
			}
		}
		if seatCount < item.SeatCount {
			occupants, err := repo.OccupiedAbove(ctx, id, seatCount)
			if err != nil {
				return err
			}
			if len(occupants) > 0 {
				items := make([]model.SeatOccupant, 0, len(occupants))
				for _, row := range occupants {
					items = append(items, model.SeatOccupant{
						SeatNo:      row.SeatNo,
						CharacterID: row.CharacterID,
						Name:        row.Name,
					})
				}
				return &model.SeatShrinkOccupiedError{
					FromSeatCount: item.SeatCount,
					ToSeatCount:   seatCount,
					Items:         items,
				}
			}
		}

		rows, err := repo.UpdateWithVersion(ctx, id, repository.RankingUpdate{
			Name:      name,
			NameKey:   nameKey,
			SeatCount: seatCount,
			Version:   version,
		}, now)
		if err != nil {
			if repository.IsUniqueViolation(err, "rankings_active_name_uq") {
				return newRankingNameConflict(nil, name)
			}
			return err
		}
		if rows == 0 {
			return model.ErrVersionConflict
		}
		if seatCount > item.SeatCount {
			return repo.InsertSeatsRange(ctx, id, item.WorkID, item.SeatCount+1, seatCount)
		}
		if seatCount < item.SeatCount {
			return repo.DeleteSeatsAbove(ctx, id, seatCount)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id, 1, RankingsDefaultSeatPageSize)
}

// Delete 软删除整榜并发放 8 秒撤销令牌；席位行保留。
func (s *Rankings) Delete(ctx context.Context, id, ifMatch string) (*DeleteRankingResult, error) {
	version := 0
	if strings.TrimSpace(ifMatch) != "" {
		parsed, ok := ParseIfMatch(ifMatch)
		if !ok {
			return nil, model.ErrPreconditionRequired
		}
		version = parsed
	}
	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err := lockRanking(ctx, tx, id)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrAlreadyDeleted
			}
			return err
		}
		if version > 0 && version != item.Version {
			return &model.VersionConflictError{CurrentVersion: item.Version}
		}
		rows, err := repository.NewRankings(tx).SoftDelete(ctx, id, model.RankingActionDelete, item.Version, now)
		if err != nil {
			return err
		}
		if rows == 0 {
			return model.ErrAlreadyDeleted
		}
		return repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeRanking,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		})
	})
	if err != nil {
		return nil, err
	}
	return &DeleteRankingResult{Deleted: true, ID: id, DeletedAt: now, UndoToken: token, ExpiresAt: expires}, nil
}

// ListSeats 返回席位分页，支持 occupied 过滤。
func (s *Rankings) ListSeats(ctx context.Context, id string, page, pageSize int, occupied *bool) (*model.SeatPage, error) {
	page, pageSize, err := normalizeSeatPage(page, pageSize)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRankings(s.db)
	if _, err := repo.Get(ctx, id); err != nil {
		return nil, err
	}
	seats, err := repo.SeatsPage(ctx, id, page, pageSize, occupied)
	if err != nil {
		return nil, err
	}
	views, err := s.seatViews(ctx, seats)
	if err != nil {
		return nil, err
	}
	total, err := repo.CountSeats(ctx, id, occupied)
	if err != nil {
		return nil, err
	}
	return &model.SeatPage{Items: views, Page: page, PageSize: pageSize, Total: int(total)}, nil
}

// PutSeat 指派/清空席位人物并更新备注；If-Match 为席位 version。
func (s *Rankings) PutSeat(ctx context.Context, rankingID string, seatNo int, in PutSeatInput) (*model.SeatView, error) {
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	now := s.now().UTC()

	var updated *model.RankingSeat
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err := lockRanking(ctx, tx, rankingID)
		if err != nil {
			return err
		}
		if seatNo < 1 || seatNo > item.SeatCount {
			return &model.SeatOutOfRangeError{SeatNo: seatNo, SeatCount: item.SeatCount}
		}
		repo := repository.NewRankings(tx)
		seat, err := repo.SeatByNoForUpdate(ctx, rankingID, seatNo)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return &model.SeatOutOfRangeError{SeatNo: seatNo, SeatCount: item.SeatCount}
			}
			return err
		}
		if seat.Version != version {
			return &model.VersionConflictError{CurrentVersion: seat.Version}
		}

		characterID := seat.CharacterID
		note := seat.Note
		switch {
		case in.CharacterSet && in.CharacterID == nil:
			// 清空人物时一并清空备注。
			characterID = nil
			note = nil
		case in.CharacterSet:
			cid := strings.TrimSpace(*in.CharacterID)
			if cid == "" {
				return &model.ValidationError{Field: "character_id", Message: "character_id 不能为空"}
			}
			characterRepo := repository.NewCharacters(tx)
			if _, err := characterRepo.GetActiveByWork(ctx, item.WorkID, cid); err != nil {
				if !errors.Is(err, model.ErrNotFound) {
					return err
				}
				any, anyErr := characterRepo.GetAny(ctx, cid)
				if anyErr != nil {
					if errors.Is(anyErr, model.ErrNotFound) {
						return model.ErrParentDeleted
					}
					return anyErr
				}
				if any.WorkID != item.WorkID {
					return &model.CrossWorkReferenceError{Field: "character_id", Message: "人物不属于当前作品"}
				}
				return model.ErrParentDeleted
			}
			existing, err := repo.ConflictSeatNo(ctx, rankingID, cid, seatNo)
			if err != nil {
				return err
			}
			if existing != nil {
				return &model.SeatConflictError{SeatNo: seatNo, CharacterID: cid, ExistingSeatNo: *existing}
			}
			characterID = &cid
			if in.NoteSet {
				if note, err = model.NormalizeSeatNote(in.Note); err != nil {
					return err
				}
			}
		default:
			if in.NoteSet {
				if note, err = model.NormalizeSeatNote(in.Note); err != nil {
					return err
				}
			}
		}

		rows, err := repo.UpdateSeatWithVersion(ctx, rankingID, seatNo, version, characterID, note, now)
		if err != nil {
			if repository.IsUniqueViolation(err, "seats_character_uq") {
				existing := seatNo
				if characterID != nil {
					if conflict, lookupErr := repo.ConflictSeatNo(ctx, rankingID, *characterID, seatNo); lookupErr == nil && conflict != nil {
						existing = *conflict
					}
					return &model.SeatConflictError{
						SeatNo:         seatNo,
						CharacterID:    *characterID,
						ExistingSeatNo: existing,
					}
				}
				return model.ErrSeatConflict
			}
			return err
		}
		if rows == 0 {
			return &model.VersionConflictError{CurrentVersion: seat.Version}
		}
		updated, err = repo.SeatByNo(ctx, rankingID, seatNo)
		return err
	})
	if err != nil {
		return nil, err
	}
	views, err := s.seatViews(ctx, []model.RankingSeat{*updated})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

// MoveSeat 将源席位人物移动到空席或与目标席位交换。
func (s *Rankings) MoveSeat(ctx context.Context, rankingID string, seatNo int, in MoveSeatInput) (*MoveSeatResult, error) {
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	mode := strings.TrimSpace(in.Mode)
	if mode != model.SeatModeEmpty && mode != model.SeatModeSwap {
		return nil, &model.ValidationError{Field: "mode", Message: "mode 仅支持 empty 或 swap"}
	}
	if in.TargetSeatNo == seatNo {
		return nil, &model.ValidationError{Field: "target_seat_no", Message: "目标席位不能与当前席位相同"}
	}
	now := s.now().UTC()

	var from, to model.RankingSeat
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err := lockRanking(ctx, tx, rankingID)
		if err != nil {
			return err
		}
		targetNo := in.TargetSeatNo
		if targetNo < 1 || targetNo > item.SeatCount {
			return &model.SeatOutOfRangeError{SeatNo: targetNo, SeatCount: item.SeatCount}
		}
		if seatNo < 1 || seatNo > item.SeatCount {
			return &model.SeatOutOfRangeError{SeatNo: seatNo, SeatCount: item.SeatCount}
		}
		repo := repository.NewRankings(tx)
		// 固定按席位号升序加锁，避免并发相交移动互相等待。
		firstNo, secondNo := seatNo, targetNo
		if firstNo > secondNo {
			firstNo, secondNo = secondNo, firstNo
		}
		if _, err := repo.SeatByNoForUpdate(ctx, rankingID, firstNo); err != nil {
			return &model.SeatOutOfRangeError{SeatNo: firstNo, SeatCount: item.SeatCount}
		}
		if _, err := repo.SeatByNoForUpdate(ctx, rankingID, secondNo); err != nil {
			return &model.SeatOutOfRangeError{SeatNo: secondNo, SeatCount: item.SeatCount}
		}
		source, err := repo.SeatByNo(ctx, rankingID, seatNo)
		if err != nil {
			return err
		}
		target, err := repo.SeatByNo(ctx, rankingID, targetNo)
		if err != nil {
			return err
		}
		if source.Version != version {
			return &model.VersionConflictError{CurrentVersion: source.Version}
		}
		if source.CharacterID == nil {
			return &model.ValidationError{Field: "seat_no", Message: "源席位为空，无需移动"}
		}

		if mode == model.SeatModeEmpty {
			if target.CharacterID != nil {
				return &model.SeatConflictError{
					SeatNo:         targetNo,
					CharacterID:    *target.CharacterID,
					ExistingSeatNo: targetNo,
				}
			}
			if _, err := repo.ClearSeatCharacter(ctx, rankingID, seatNo, source.Version, now); err != nil {
				return err
			}
			if _, err := repo.UpdateSeatWithVersion(ctx, rankingID, targetNo, target.Version, source.CharacterID, source.Note, now); err != nil {
				return err
			}
		} else {
			if err := repo.ClearSeatCharacters(ctx, rankingID, []int{seatNo, targetNo}, now); err != nil {
				return err
			}
			if target.CharacterID != nil {
				if err := repo.SetSeatCharacter(ctx, rankingID, seatNo, *target.CharacterID, now); err != nil {
					return err
				}
			}
			if source.CharacterID != nil {
				if err := repo.SetSeatCharacter(ctx, rankingID, targetNo, *source.CharacterID, now); err != nil {
					return err
				}
			}
		}

		fromSeat, err := repo.SeatByNo(ctx, rankingID, seatNo)
		if err != nil {
			return err
		}
		toSeat, err := repo.SeatByNo(ctx, rankingID, targetNo)
		if err != nil {
			return err
		}
		from, to = *fromSeat, *toSeat
		return nil
	})
	if err != nil {
		return nil, err
	}
	views, err := s.seatViews(ctx, []model.RankingSeat{from, to})
	if err != nil {
		return nil, err
	}
	return &MoveSeatResult{From: views[0], To: views[1]}, nil
}

// seatViews 批量装配席位人物摘要；人物缺失时 character 为 null。
func (s *Rankings) seatViews(ctx context.Context, seats []model.RankingSeat) ([]model.SeatView, error) {
	out := make([]model.SeatView, 0, len(seats))
	ids := make([]string, 0, len(seats))
	seen := make(map[string]struct{}, len(seats))
	for _, seat := range seats {
		if seat.CharacterID == nil {
			continue
		}
		if _, ok := seen[*seat.CharacterID]; ok {
			continue
		}
		seen[*seat.CharacterID] = struct{}{}
		ids = append(ids, *seat.CharacterID)
	}
	summaries := make(map[string]model.CharacterSummary, len(ids))
	if len(ids) > 0 && s.characters != nil {
		items, err := repository.NewCharacters(s.db).ListByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		built, err := s.characters.Summaries(ctx, items)
		if err != nil {
			return nil, err
		}
		for _, summary := range built {
			summaries[summary.ID] = summary
		}
	}
	for _, seat := range seats {
		view := model.SeatView{SeatNo: seat.SeatNo, Note: seat.Note, Version: seat.Version}
		if seat.CharacterID != nil {
			if summary, ok := summaries[*seat.CharacterID]; ok {
				value := summary
				view.Character = &value
			}
		}
		out = append(out, view)
	}
	return out, nil
}

// lockRanking 按「作品行锁 → 榜单 advisory → 榜单行锁」顺序串行化席位变更。
func lockRanking(ctx context.Context, tx *gorm.DB, id string) (*model.Ranking, error) {
	repo := repository.NewRankings(tx)
	probe, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := repository.NewWorks(tx).GetForUpdate(ctx, probe.WorkID); err != nil {
		return nil, err
	}
	if err := tx.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtext(?))", id).Error; err != nil {
		return nil, err
	}
	return repo.GetForUpdate(ctx, id)
}

func rankingSummary(item model.Ranking, occupied int64, tops []model.RankingTopSeat) model.RankingSummary {
	if tops == nil {
		tops = []model.RankingTopSeat{}
	}
	return model.RankingSummary{
		ID:            item.ID,
		WorkID:        item.WorkID,
		Name:          item.Name,
		SeatCount:     item.SeatCount,
		OccupiedCount: int(occupied),
		EmptyCount:    item.SeatCount - int(occupied),
		TopSeats:      tops,
		ReviewStatus:  model.APIReviewStatus(item.ReviewStatus),
		ReviewedAt:    item.ReviewedAt,
		UpdatedAt:     item.UpdatedAt,
		Version:       item.Version,
	}
}

func newRankingNameConflict(existing *model.Ranking, name string) error {
	item := model.ConflictItem{Name: name, EntityType: RootTypeRanking}
	if existing != nil {
		item.ID = existing.ID
	}
	return &model.NameConflictError{Items: []model.ConflictItem{item}}
}

// normalizeSeatPage 校验席位分页：默认 50，仅允许 50/100。
func normalizeSeatPage(page, pageSize int) (int, int, error) {
	if page < 1 {
		page = 1
	}
	switch pageSize {
	case 0:
		return page, RankingsDefaultSeatPageSize, nil
	case RankingsDefaultSeatPageSize, RankingsMaxSeatPageSize:
		return page, pageSize, nil
	default:
		return 0, 0, &model.ValidationError{Field: "page_size", Message: "page_size 仅支持 50 或 100"}
	}
}

func boolPtr(value bool) *bool { return &value }

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
	"novel-management/backend/internal/search"
)

// RootTypeLocation 是回收站中地点根的标识。
const RootTypeLocation = "location"

// Locations 负责地点树的创建、查询、移动、关系维护与删除。
type Locations struct {
	db         *gorm.DB
	characters *Characters
	now        func() time.Time
}

func NewLocations(db *gorm.DB, characters *Characters) *Locations {
	return &Locations{db: db, characters: characters, now: time.Now}
}

type ListLocationsInput struct {
	WorkID       string
	Query        string
	ParentID     string
	TypeText     string
	ReviewStatus string
}

type CreateLocationInput struct {
	WorkID      string
	Name        string
	ParentID    *string
	TypeText    *string
	Description *string
}

type UpdateLocationInput struct {
	Name             *string
	ParentID         *string
	ClearParent      bool
	TypeText         *string
	ClearType        bool
	Description      *string
	ClearDescription bool
	IfMatch          string
}

type DeleteLocationResult struct {
	Deleted   bool      `json:"deleted"`
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
	UndoToken string    `json:"undo_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AddLocationFactionInput struct {
	FactionID string
	Note      *string
}

// LocationList 是树接口响应；前端一次取全量后本地组树。
type LocationList struct {
	Items []model.LocationSummary `json:"items"`
	Total int                     `json:"total"`
}

// List 返回作品下扁平全量地点；命中筛选时补充全部祖先，便于前端保持树结构。
func (s *Locations) List(ctx context.Context, in ListLocationsInput) (*LocationList, error) {
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
	parentFilter := strings.TrimSpace(in.ParentID)
	typeText := strings.TrimSpace(in.TypeText)

	repo := repository.NewLocations(s.db)
	all, err := repo.ListActive(ctx, workID, "")
	if err != nil {
		return nil, err
	}
	byID := make(map[string]model.Location, len(all))
	for _, item := range all {
		byID[item.ID] = item
	}
	selected := selectLocations(all, byID, query, typeText, reviewDB, parentFilter)
	if len(selected) == 0 {
		return &LocationList{Items: []model.LocationSummary{}, Total: 0}, nil
	}

	childCounts, err := repo.CountActiveChildren(ctx, workID)
	if err != nil {
		return nil, err
	}
	charCounts, err := repo.CountActiveCharacters(ctx, workID)
	if err != nil {
		return nil, err
	}
	factionCounts, err := repo.CountActiveFactions(ctx, workID)
	if err != nil {
		return nil, err
	}

	out := make([]model.LocationSummary, 0, len(selected))
	for _, item := range selected {
		out = append(out, locationSummary(item, byID, childCounts, charCounts, factionCounts))
	}
	return &LocationList{Items: out, Total: len(out)}, nil
}

func selectLocations(all []model.Location, byID map[string]model.Location, query, typeText, reviewDB, parentFilter string) []model.Location {
	if query == "" && typeText == "" && reviewDB == "" && parentFilter == "" {
		return all
	}
	lowerQuery := strings.ToLower(query)
	matches := func(item model.Location) bool {
		if lowerQuery != "" && !strings.Contains(item.NameKey, lowerQuery) {
			return false
		}
		if typeText != "" {
			if item.TypeText == nil || !strings.EqualFold(*item.TypeText, typeText) {
				return false
			}
		}
		if reviewDB != "" && item.ReviewStatus != reviewDB {
			return false
		}
		if parentFilter != "" {
			if parentFilter == "root" {
				if item.ParentID != nil {
					return false
				}
			} else if item.ParentID == nil || *item.ParentID != parentFilter {
				return false
			}
		}
		return true
	}

	selected := make(map[string]struct{}, len(all))
	for _, item := range all {
		if !matches(item) {
			continue
		}
		selected[item.ID] = struct{}{}
		// 搜索/类型/待校对命中时补齐全部祖先，保证层级不断裂。
		parentID := item.ParentID
		for parentID != nil {
			parent, ok := byID[*parentID]
			if !ok {
				break
			}
			selected[parent.ID] = struct{}{}
			parentID = parent.ParentID
		}
	}

	out := make([]model.Location, 0, len(selected))
	for _, item := range all {
		if _, ok := selected[item.ID]; ok {
			out = append(out, item)
		}
	}
	return out
}

func locationSummary(item model.Location, byID map[string]model.Location, childCounts, charCounts, factionCounts map[string]int64) model.LocationSummary {
	var parent *model.LocationBrief
	if item.ParentID != nil {
		if parentItem, ok := byID[*item.ParentID]; ok {
			parent = &model.LocationBrief{ID: parentItem.ID, Name: parentItem.Name}
		} else {
			parent = &model.LocationBrief{ID: *item.ParentID}
		}
	}
	return model.LocationSummary{
		ID:              item.ID,
		WorkID:          item.WorkID,
		Name:            item.Name,
		Parent:          parent,
		TypeText:        item.TypeText,
		Description:     item.Description,
		ReviewStatus:    model.APIReviewStatus(item.ReviewStatus),
		ReviewedAt:      item.ReviewedAt,
		ChildrenCount:   childCounts[item.ID],
		CharactersCount: charCounts[item.ID],
		FactionsCount:   factionCounts[item.ID],
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		Version:         item.Version,
		DeletedAt:       item.DeletedAt,
	}
}

// Get 返回地点详情（含下级、关联人物与关联势力）。
func (s *Locations) Get(ctx context.Context, id string) (*model.LocationDetail, error) {
	repo := repository.NewLocations(s.db)
	item, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	parentName := make(map[string]string)
	if item.ParentID != nil {
		briefs, err := repo.ParentBriefs(ctx, []string{*item.ParentID})
		if err != nil {
			return nil, err
		}
		for _, brief := range briefs {
			parentName[brief.ID] = brief.Name
		}
	}
	childCounts, err := repo.CountActiveChildren(ctx, item.WorkID)
	if err != nil {
		return nil, err
	}
	charCounts, err := repo.CountActiveCharacters(ctx, item.WorkID)
	if err != nil {
		return nil, err
	}
	factionCounts, err := repo.CountActiveFactions(ctx, item.WorkID)
	if err != nil {
		return nil, err
	}
	byID := map[string]model.Location{item.ID: *item}
	sum := locationSummary(*item, byID, childCounts, charCounts, factionCounts)
	if item.ParentID != nil {
		if name, ok := parentName[*item.ParentID]; ok {
			sum.Parent = &model.LocationBrief{ID: *item.ParentID, Name: name}
		}
	}

	children, err := repo.ListActiveChildren(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	childViews := make([]model.LocationChildView, 0, len(children))
	for _, child := range children {
		childViews = append(childViews, model.LocationChildView{
			ID:              child.ID,
			Name:            child.Name,
			TypeText:        child.TypeText,
			CharactersCount: charCounts[child.ID],
			FactionsCount:   factionCounts[child.ID],
		})
	}

	characters, err := s.characterViews(ctx, repo, item.ID)
	if err != nil {
		return nil, err
	}
	factions, err := s.factionViews(ctx, repo, item.ID)
	if err != nil {
		return nil, err
	}

	return &model.LocationDetail{
		LocationSummary: sum,
		Relations: model.LocationRelations{
			Children:   childViews,
			Characters: characters,
			Factions:   factions,
		},
	}, nil
}

// characterViews 装配地点详情关联人物；回收站人物保留并标记 in_trash，便于解释删除阻断。
func (s *Locations) characterViews(ctx context.Context, repo *repository.Locations, locationID string) ([]model.LocationCharacterView, error) {
	rows, err := repo.CharacterRowsForLocation(ctx, locationID)
	if err != nil {
		return nil, err
	}
	out := make([]model.LocationCharacterView, 0, len(rows))
	active := make([]repository.LocationCharacterRow, 0, len(rows))
	for _, row := range rows {
		if row.InTrash {
			out = append(out, model.LocationCharacterView{
				CharacterID: row.CharacterID,
				Name:        row.Name,
				Note:        row.Note,
				InTrash:     true,
			})
			continue
		}
		active = append(active, row)
	}
	if len(active) == 0 || s.characters == nil {
		return out, nil
	}
	items := make([]model.Character, 0, len(active))
	for _, row := range active {
		items = append(items, model.Character{ID: row.CharacterID, Name: row.Name, FactionID: row.FactionID})
	}
	summaries, err := s.characters.Summaries(ctx, items)
	if err != nil {
		return nil, err
	}
	notes := make(map[string]*string, len(active))
	for _, row := range active {
		notes[row.CharacterID] = row.Note
	}
	for _, summary := range summaries {
		out = append(out, model.LocationCharacterView{
			CharacterID:   summary.ID,
			Name:          summary.Name,
			Faction:       summary.Faction,
			SystemSummary: summary.SystemSummary,
			ImageThumbURL: summary.ImageThumbURL,
			Note:          notes[summary.ID],
			InTrash:       false,
		})
	}
	return out, nil
}

// factionViews 装配地点详情关联势力；回收站势力保留并标记 in_trash，便于解释删除阻断。
func (s *Locations) factionViews(ctx context.Context, repo *repository.Locations, locationID string) ([]model.LocationFactionView, error) {
	rows, err := repo.FactionRowsForLocation(ctx, locationID)
	if err != nil {
		return nil, err
	}
	out := make([]model.LocationFactionView, 0, len(rows))
	for _, row := range rows {
		out = append(out, model.LocationFactionView{
			FactionID: row.FactionID,
			Name:      row.Name,
			Note:      row.Note,
			InTrash:   row.InTrash,
		})
	}
	return out, nil
}

// Create 新建地点节点；名称同作品唯一，父级必须同作品活跃。
func (s *Locations) Create(ctx context.Context, in CreateLocationInput) (*model.LocationDetail, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	name, nameKey, err := model.NormalizeLocationName(in.Name)
	if err != nil {
		return nil, err
	}
	typeText, err := model.NormalizeLocationType(in.TypeText)
	if err != nil {
		return nil, err
	}
	description, err := model.NormalizeLocationDescription(in.Description)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	item := &model.Location{
		ID:           model.NewID(),
		WorkID:       workID,
		Name:         name,
		NameKey:      nameKey,
		TypeText:     typeText,
		Description:  description,
		ReviewStatus: model.ReviewStatusReviewed,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewLocations(tx)
		if _, err := repository.NewWorks(tx).GetForUpdate(ctx, workID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrCharacterWorkUnavailable
			}
			return err
		}
		if in.ParentID != nil {
			parentID, err := resolveLocationParent(ctx, tx, workID, *in.ParentID)
			if err != nil {
				return err
			}
			item.ParentID = parentID
		}
		taken, err := repo.NameTaken(ctx, workID, nameKey, "")
		if err != nil {
			return err
		}
		if taken {
			existing, lookupErr := repo.ActiveByName(ctx, workID, nameKey, "")
			if lookupErr != nil {
				return lookupErr
			}
			return newLocationNameConflict(existing, name)
		}
		if err := repo.Insert(ctx, item); err != nil {
			if repository.IsUniqueViolation(err, "locations_active_name_uq") {
				return newLocationNameConflict(nil, name)
			}
			return err
		}
		return search.NewRefresher(tx).RefreshWork(ctx, item.WorkID)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, item.ID)
}

// Update 以内联字段子集更新；移动时串行化作品内变更并禁止成环。
func (s *Locations) Update(ctx context.Context, id string, in UpdateLocationInput) (*model.LocationDetail, error) {
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	now := s.now().UTC()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewLocations(tx)
		item, err := repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if version != item.Version {
			return &model.VersionConflictError{CurrentVersion: item.Version}
		}

		name, nameKey := item.Name, item.NameKey
		if in.Name != nil {
			if name, nameKey, err = model.NormalizeLocationName(*in.Name); err != nil {
				return err
			}
		}
		typeText := item.TypeText
		if in.ClearType {
			typeText = nil
		} else if in.TypeText != nil {
			if typeText, err = model.NormalizeLocationType(in.TypeText); err != nil {
				return err
			}
		}
		description := item.Description
		if in.ClearDescription {
			description = nil
		} else if in.Description != nil {
			if description, err = model.NormalizeLocationDescription(in.Description); err != nil {
				return err
			}
		}
		parentID := item.ParentID
		if in.ClearParent {
			parentID = nil
		} else if in.ParentID != nil {
			if parentID, err = resolveLocationParent(ctx, tx, item.WorkID, *in.ParentID); err != nil {
				return err
			}
		}
		if parentID != nil {
			if err := lockLocationWorkMoves(ctx, tx, item.WorkID); err != nil {
				return err
			}
			cycle, err := repo.IsAncestor(ctx, id, *parentID)
			if err != nil {
				return err
			}
			if cycle {
				return &model.CycleDetectedError{Field: "parent_id"}
			}
		}
		if nameKey != item.NameKey {
			taken, err := repo.NameTaken(ctx, item.WorkID, nameKey, id)
			if err != nil {
				return err
			}
			if taken {
				existing, err := repo.ActiveByName(ctx, item.WorkID, nameKey, id)
				if err != nil {
					return err
				}
				return newLocationNameConflict(existing, name)
			}
		}

		rows, err := repo.UpdateWithVersion(ctx, id, repository.LocationUpdate{
			Name:        name,
			NameKey:     nameKey,
			ParentID:    parentID,
			TypeText:    typeText,
			Description: description,
			Version:     version,
		}, now)
		if err != nil {
			if repository.IsUniqueViolation(err, "locations_active_name_uq") {
				return newLocationNameConflict(nil, name)
			}
			return err
		}
		if rows == 0 {
			return model.ErrVersionConflict
		}
		return search.NewRefresher(tx).RefreshWork(ctx, item.WorkID)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Delete 在入向引用检查通过后软删除地点并发放 8 秒撤销令牌。
func (s *Locations) Delete(ctx context.Context, id, ifMatch string) (*DeleteLocationResult, error) {
	version, ok := ParseIfMatch(ifMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewLocations(tx)
		item, err := repo.GetForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrAlreadyDeleted
			}
			return err
		}
		if version != item.Version {
			return &model.VersionConflictError{CurrentVersion: item.Version}
		}
		children, err := repo.ListChildRows(ctx, id)
		if err != nil {
			return err
		}
		characterRows, err := repo.CharacterRowsForLocation(ctx, id)
		if err != nil {
			return err
		}
		factionRows, err := repo.FactionRowsForLocation(ctx, id)
		if err != nil {
			return err
		}
		if len(children) > 0 {
			// 一次返回全部三类引用，避免用户先清下级、再被关系阻断。
			items := make([]model.ConflictItem, 0, len(children)+len(characterRows)+len(factionRows))
			for _, child := range children {
				items = append(items, model.ConflictItem{
					ID:         child.ID,
					Name:       child.Name,
					EntityType: "location_child",
					InTrash:    child.DeletedAt != nil,
				})
			}
			for _, row := range characterRows {
				if len(items) >= model.ReferenceListLimit {
					break
				}
				items = append(items, model.ConflictItem{
					ID:         row.CharacterID,
					Name:       row.Name,
					EntityType: "character_location",
					InTrash:    row.InTrash,
				})
			}
			for _, row := range factionRows {
				if len(items) >= model.ReferenceListLimit {
					break
				}
				items = append(items, model.ConflictItem{
					ID:         row.FactionID,
					Name:       row.Name,
					EntityType: "location_faction",
					InTrash:    row.InTrash,
				})
			}
			return &model.HasChildrenError{Items: items}
		}
		if err := locationReferenceConflict(item.Name, characterRows, factionRows); err != nil {
			return err
		}
		rows, err := repo.SoftDelete(ctx, id, model.LocationActionDelete, version, now)
		if err != nil {
			return err
		}
		if rows == 0 {
			return model.ErrAlreadyDeleted
		}
		if err := repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeLocation,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		}); err != nil {
			return err
		}
		return search.NewRefresher(tx).RefreshWork(ctx, item.WorkID)
	})
	if err != nil {
		return nil, err
	}
	return &DeleteLocationResult{Deleted: true, ID: id, DeletedAt: now, UndoToken: token, ExpiresAt: expires}, nil
}

// ListCharacters 返回地点的人物关系（含回收站人物标记）。
func (s *Locations) ListCharacters(ctx context.Context, locationID string) ([]model.LocationCharacterView, error) {
	repo := repository.NewLocations(s.db)
	if _, err := repo.Get(ctx, locationID); err != nil {
		return nil, err
	}
	rows, err := repo.CharacterRowsForLocation(ctx, locationID)
	if err != nil {
		return nil, err
	}
	out := make([]model.LocationCharacterView, 0, len(rows))
	if s.characters == nil {
		for _, row := range rows {
			out = append(out, model.LocationCharacterView{
				CharacterID: row.CharacterID,
				Name:        row.Name,
				Note:        row.Note,
				InTrash:     row.InTrash,
			})
		}
		return out, nil
	}
	items := make([]model.Character, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.Character{ID: row.CharacterID, Name: row.Name, FactionID: row.FactionID})
	}
	summaries, err := s.characters.Summaries(ctx, items)
	if err != nil {
		return nil, err
	}
	notes := make(map[string]*string, len(rows))
	trashed := make(map[string]bool, len(rows))
	for _, row := range rows {
		notes[row.CharacterID] = row.Note
		trashed[row.CharacterID] = row.InTrash
	}
	for _, summary := range summaries {
		out = append(out, model.LocationCharacterView{
			CharacterID:   summary.ID,
			Name:          summary.Name,
			Faction:       summary.Faction,
			SystemSummary: summary.SystemSummary,
			ImageThumbURL: summary.ImageThumbURL,
			Note:          notes[summary.ID],
			InTrash:       trashed[summary.ID],
		})
	}
	return out, nil
}

// ListFactions 返回地点的势力关系（含回收站势力标记）。
func (s *Locations) ListFactions(ctx context.Context, locationID string) ([]model.LocationFactionView, error) {
	repo := repository.NewLocations(s.db)
	if _, err := repo.Get(ctx, locationID); err != nil {
		return nil, err
	}
	rows, err := repo.FactionRowsForLocation(ctx, locationID)
	if err != nil {
		return nil, err
	}
	out := make([]model.LocationFactionView, 0, len(rows))
	for _, row := range rows {
		out = append(out, model.LocationFactionView{
			FactionID: row.FactionID,
			Name:      row.Name,
			Note:      row.Note,
			InTrash:   row.InTrash,
		})
	}
	return out, nil
}

// AddFaction 新增地点-势力关系；地点与势力必须同作品活跃。
func (s *Locations) AddFaction(ctx context.Context, locationID string, in AddLocationFactionInput) (*model.LocationFactionView, error) {
	factionID := strings.TrimSpace(in.FactionID)
	if factionID == "" {
		return nil, &model.ValidationError{Field: "faction_id", Message: "faction_id 不能为空"}
	}
	note, err := model.NormalizeRelationNote(in.Note)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	var view *model.LocationFactionView
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewLocations(tx)
		// SHARE 锁与地点删除的 UPDATE 锁互斥，避免关系写入已删地点。
		location, err := repo.GetForShare(ctx, locationID)
		if err != nil {
			return err
		}
		factionRepo := repository.NewFactions(tx)
		faction, err := factionRepo.GetActiveByWork(ctx, location.WorkID, factionID)
		if err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return err
			}
			any, anyErr := factionRepo.GetAny(ctx, factionID)
			if anyErr != nil {
				if errors.Is(anyErr, model.ErrNotFound) {
					return &model.ValidationError{Field: "faction_id", Message: "势力不存在"}
				}
				return anyErr
			}
			if any.WorkID != location.WorkID {
				return &model.CrossWorkReferenceError{Field: "faction_id", Message: "势力不属于当前作品"}
			}
			return model.ErrParentDeleted
		}
		exists, err := repo.HasLocationFaction(ctx, locationID, factionID)
		if err != nil {
			return err
		}
		if exists {
			return model.ErrDuplicateRelation
		}
		item := &model.LocationFaction{
			ID:         model.NewID(),
			WorkID:     location.WorkID,
			LocationID: locationID,
			FactionID:  factionID,
			Note:       note,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := repo.InsertLocationFaction(ctx, item); err != nil {
			if repository.IsUniqueViolation(err, "loc_fac_uq") {
				return model.ErrDuplicateRelation
			}
			return err
		}
		view = &model.LocationFactionView{FactionID: faction.ID, Name: faction.Name, Note: note}
		return search.NewRefresher(tx).RefreshWork(ctx, location.WorkID)
	})
	if err != nil {
		return nil, err
	}
	return view, nil
}

// UpdateFactionNote 仅更新地点-势力关系备注。
func (s *Locations) UpdateFactionNote(ctx context.Context, locationID, factionID string, note *string) (*model.LocationFactionView, error) {
	normalized, err := model.NormalizeRelationNote(note)
	if err != nil {
		return nil, err
	}
	repo := repository.NewLocations(s.db)
	if _, err := repo.Get(ctx, locationID); err != nil {
		return nil, err
	}
	rows, err := repo.UpdateLocationFactionNote(ctx, locationID, factionID, normalized, s.now().UTC())
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, model.ErrNotFound
	}
	rowsView, err := repo.FactionRowsForLocation(ctx, locationID)
	if err != nil {
		return nil, err
	}
	for _, row := range rowsView {
		if row.FactionID == factionID {
			return &model.LocationFactionView{
				FactionID: row.FactionID,
				Name:      row.Name,
				Note:      row.Note,
				InTrash:   row.InTrash,
			}, nil
		}
	}
	return nil, model.ErrNotFound
}

// RemoveFaction 删除单条地点-势力关系，不删除势力。
func (s *Locations) RemoveFaction(ctx context.Context, locationID, factionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewLocations(tx)
		location, err := repo.Get(ctx, locationID)
		if err != nil {
			return err
		}
		rows, err := repo.DeleteLocationFaction(ctx, locationID, factionID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return model.ErrNotFound
		}
		return search.NewRefresher(tx).RefreshWork(ctx, location.WorkID)
	})
}

// resolveLocationParent 校验父级同作品且活跃，返回去空白后的 ID。
func resolveLocationParent(ctx context.Context, tx *gorm.DB, workID, raw string) (*string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return nil, &model.ValidationError{Field: "parent_id", Message: "parent_id 不能为空"}
	}
	repo := repository.NewLocations(tx)
	if _, err := repo.GetActiveByWorkForShare(ctx, workID, id); err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return nil, err
		}
		any, anyErr := repo.GetAny(ctx, id)
		if anyErr != nil {
			if errors.Is(anyErr, model.ErrNotFound) {
				return nil, model.ErrParentDeleted
			}
			return nil, anyErr
		}
		if any.WorkID != workID {
			return nil, &model.CrossWorkReferenceError{Field: "parent_id", Message: "上级地点不属于当前作品"}
		}
		return nil, model.ErrParentDeleted
	}
	return &id, nil
}

// lockLocationWorkMoves 以作品为粒度串行化上级变更，避免并发移动互相不可见而成环。
func lockLocationWorkMoves(ctx context.Context, tx *gorm.DB, workID string) error {
	return tx.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtext(?))", workID).Error
}

func newLocationNameConflict(existing *model.Location, name string) error {
	item := model.ConflictItem{Name: name, EntityType: RootTypeLocation}
	if existing != nil {
		item.ID = existing.ID
	}
	return &model.NameConflictError{Items: []model.ConflictItem{item}}
}

// locationReferenceConflict 构造「人物/势力仍引用地点」的删除阻断明细。
func locationReferenceConflict(name string, characters []repository.LocationCharacterRow, factions []repository.LocationFactionRow) error {
	items := make([]model.ReferenceItem, 0, len(characters)+len(factions))
	for _, row := range characters {
		if len(items) >= model.ReferenceListLimit {
			break
		}
		characterName := row.Name
		items = append(items, model.ReferenceItem{
			Type:     "character_location",
			EntityID: row.CharacterID,
			Name:     &characterName,
			Field:    "location_id",
			Value:    name,
			InTrash:  row.InTrash,
		})
	}
	for _, row := range factions {
		if len(items) >= model.ReferenceListLimit {
			break
		}
		factionName := row.Name
		items = append(items, model.ReferenceItem{
			Type:     "location_faction",
			EntityID: row.FactionID,
			Name:     &factionName,
			Field:    "location_id",
			Value:    name,
			InTrash:  row.InTrash,
		})
	}
	if len(items) == 0 {
		return nil
	}
	return &model.ReferenceConflictError{Message: "地点仍被引用，无法删除", Items: items}
}

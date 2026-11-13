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

// RootTypeFaction 是回收站中势力根的标识。
const RootTypeFaction = "faction"

// Factions 负责势力树的创建、查询、移动、成员推导与删除。
type Factions struct {
	db  *gorm.DB
	now func() time.Time
}

func NewFactions(db *gorm.DB) *Factions {
	return &Factions{db: db, now: time.Now}
}

type ListFactionsInput struct {
	WorkID       string
	Query        string
	ParentID     string
	ReviewStatus string
}

type CreateFactionInput struct {
	WorkID            string
	Name              string
	ParentID          *string
	LeaderCharacterID *string
	Description       *string
}

type UpdateFactionInput struct {
	Name              *string
	ParentID          *string
	ClearParent       bool
	LeaderCharacterID *string
	ClearLeader       bool
	Description       *string
	ClearDescription  bool
	IfMatch           string
}

type DeleteFactionResult struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
	UndoToken string    `json:"undo_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// FactionList 是树接口响应；前端一次取全量后本地组树（设计 D3）。
type FactionList struct {
	Items []model.FactionSummary `json:"items"`
	Total int                    `json:"total"`
}

// MemberPage 是成员分页；items 复用人物摘要 DTO。
type MemberPage struct {
	Items    []model.CharacterSummary `json:"items"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
	Total    int64                    `json:"total"`
}

// List 返回作品下扁平全量势力；q 命中节点时补充其全部祖先，便于前端保持树结构。
func (s *Factions) List(ctx context.Context, in ListFactionsInput) (*FactionList, error) {
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

	repo := repository.NewFactions(s.db)
	all, err := repo.ListActive(ctx, workID, "")
	if err != nil {
		return nil, err
	}
	byID := make(map[string]model.Faction, len(all))
	for _, item := range all {
		byID[item.ID] = item
	}

	selected, err := selectFactions(all, byID, query, reviewDB, parentFilter)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return &FactionList{Items: []model.FactionSummary{}, Total: 0}, nil
	}

	childCounts, err := repo.CountActiveChildren(ctx, workID)
	if err != nil {
		return nil, err
	}
	memberCounts, err := repo.CountActiveMembers(ctx, workID)
	if err != nil {
		return nil, err
	}
	leaderIDs := make([]string, 0, len(selected))
	for _, item := range selected {
		if item.LeaderCharacterID != nil {
			leaderIDs = append(leaderIDs, *item.LeaderCharacterID)
		}
	}
	leaderNames, err := repo.LeaderNames(ctx, leaderIDs)
	if err != nil {
		return nil, err
	}

	views := make([]model.FactionSummary, 0, len(selected))
	for _, item := range selected {
		views = append(views, s.summary(item, byID, leaderNames, childCounts, memberCounts))
	}
	return &FactionList{Items: views, Total: len(views)}, nil
}

// selectFactions 应用 q / review_status / parent_id 过滤；q 命中时补齐祖先。
func selectFactions(all []model.Faction, byID map[string]model.Faction, query, reviewDB, parentFilter string) ([]model.Faction, error) {
	selected := make(map[string]struct{}, len(all))
	matches := func(item model.Faction) bool {
		if query != "" && !strings.Contains(item.NameKey, strings.ToLower(query)) {
			return false
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

	hit := make(map[string]struct{}, len(all))
	for _, item := range all {
		if matches(item) {
			selected[item.ID] = struct{}{}
			hit[item.ID] = struct{}{}
		}
	}
	// 搜索命中时补齐全部祖先，保证前端组树不丢父链。
	if query != "" {
		for id := range hit {
			parentID := byID[id].ParentID
			for parentID != nil {
				parent, ok := byID[*parentID]
				if !ok {
					break
				}
				selected[parent.ID] = struct{}{}
				parentID = parent.ParentID
			}
		}
	}

	out := make([]model.Faction, 0, len(selected))
	for _, item := range all {
		if _, ok := selected[item.ID]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *Factions) summary(item model.Faction, byID map[string]model.Faction, leaderNames map[string]string, childCounts, memberCounts map[string]int64) model.FactionSummary {
	var parent *model.FactionBrief
	if item.ParentID != nil {
		if parentItem, ok := byID[*item.ParentID]; ok {
			parent = &model.FactionBrief{ID: parentItem.ID, Name: parentItem.Name}
		} else {
			parent = &model.FactionBrief{ID: *item.ParentID}
		}
	}
	var leader *model.FactionBrief
	if item.LeaderCharacterID != nil {
		leader = &model.FactionBrief{ID: *item.LeaderCharacterID, Name: leaderNames[*item.LeaderCharacterID]}
	}
	return model.FactionSummary{
		ID:            item.ID,
		WorkID:        item.WorkID,
		Name:          item.Name,
		Parent:        parent,
		Leader:        leader,
		Description:   item.Description,
		ReviewStatus:  model.APIReviewStatus(item.ReviewStatus),
		ReviewedAt:    item.ReviewedAt,
		ChildrenCount: childCounts[item.ID],
		MemberCount:   memberCounts[item.ID],
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
		Version:       item.Version,
		DeletedAt:     item.DeletedAt,
	}
}

// Get 返回势力详情（含下级、地点占位、大事记计数与含下级成员数）。
func (s *Factions) Get(ctx context.Context, id string) (*model.FactionDetail, error) {
	repo := repository.NewFactions(s.db)
	item, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	parentName := make(map[string]string)
	leaderNames := make(map[string]string)
	if item.ParentID != nil {
		if parent, err := repo.Get(ctx, *item.ParentID); err == nil {
			parentName[parent.ID] = parent.Name
		}
	}
	if item.LeaderCharacterID != nil {
		names, err := repo.LeaderNames(ctx, []string{*item.LeaderCharacterID})
		if err != nil {
			return nil, err
		}
		leaderNames = names
	}
	childCounts, err := repo.CountActiveChildren(ctx, item.WorkID)
	if err != nil {
		return nil, err
	}
	memberCounts, err := repo.CountActiveMembers(ctx, item.WorkID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]model.Faction)
	sum := s.summary(*item, byID, leaderNames, childCounts, memberCounts)
	if item.ParentID != nil {
		if name, ok := parentName[*item.ParentID]; ok {
			sum.Parent = &model.FactionBrief{ID: *item.ParentID, Name: name}
		}
	}

	children, err := repo.ListActiveChildren(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	childViews := make([]model.FactionBrief, 0, len(children))
	for _, child := range children {
		childViews = append(childViews, model.FactionBrief{ID: child.ID, Name: child.Name})
	}
	chronicleCount, err := repository.NewChronicles(s.db).CountActive(ctx, model.ChronicleOwnerFaction, item.ID)
	if err != nil {
		return nil, err
	}
	subtreeMembers, err := repo.CountSubtreeMembers(ctx, item.WorkID, item.ID)
	if err != nil {
		return nil, err
	}

	return &model.FactionDetail{
		FactionSummary: sum,
		Relations: model.FactionRelations{
			Children:           childViews,
			Locations:          []model.LocationBrief{},
			ChronicleCount:     chronicleCount,
			SubtreeMemberCount: subtreeMembers,
		},
	}, nil
}

// Create 在事务内校验作品、父级、首领与名称唯一后写入势力。
func (s *Factions) Create(ctx context.Context, in CreateFactionInput) (*model.FactionDetail, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	name, nameKey, err := model.NormalizeFactionName(in.Name)
	if err != nil {
		return nil, err
	}
	description, err := model.NormalizeFactionDescription(in.Description)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	item := &model.Faction{
		ID:           model.NewID(),
		WorkID:       workID,
		Name:         name,
		NameKey:      nameKey,
		Description:  description,
		ReviewStatus: model.ReviewStatusReviewed,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewFactions(tx)
		if _, err := repository.NewWorks(tx).GetForUpdate(ctx, workID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrCharacterWorkUnavailable
			}
			return err
		}
		if in.ParentID != nil {
			parentID, err := resolveFactionParent(ctx, tx, workID, *in.ParentID)
			if err != nil {
				return err
			}
			item.ParentID = parentID
		}
		if in.LeaderCharacterID != nil {
			leaderID, err := resolveFactionLeader(ctx, tx, workID, *in.LeaderCharacterID)
			if err != nil {
				return err
			}
			item.LeaderCharacterID = leaderID
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
			return newFactionNameConflict(existing, name)
		}
		if err := repo.Insert(ctx, item); err != nil {
			if repository.IsUniqueViolation(err, "factions_active_name_uq") {
				return newFactionNameConflict(nil, name)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, item.ID)
}

// Update 以内联字段子集更新；If-Match 缺失与版本不匹配分别返回 428/409。
func (s *Factions) Update(ctx context.Context, id string, in UpdateFactionInput) (*model.FactionDetail, error) {
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	now := s.now().UTC()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewFactions(tx)
		item, err := repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if version != item.Version {
			return &model.VersionConflictError{CurrentVersion: item.Version}
		}

		name, nameKey := item.Name, item.NameKey
		if in.Name != nil {
			if name, nameKey, err = model.NormalizeFactionName(*in.Name); err != nil {
				return err
			}
		}
		description := item.Description
		if in.ClearDescription {
			description = nil
		} else if in.Description != nil {
			if description, err = model.NormalizeFactionDescription(in.Description); err != nil {
				return err
			}
		}

		parentID := item.ParentID
		if in.ClearParent {
			parentID = nil
		} else if in.ParentID != nil {
			resolved, err := resolveFactionParent(ctx, tx, item.WorkID, *in.ParentID)
			if err != nil {
				return err
			}
			if resolved != nil {
				cycle, err := repo.IsAncestor(ctx, item.ID, *resolved)
				if err != nil {
					return err
				}
				if cycle {
					return &model.CycleDetectedError{Field: "parent_id"}
				}
			}
			parentID = resolved
		}

		leaderID := item.LeaderCharacterID
		if in.ClearLeader {
			leaderID = nil
		} else if in.LeaderCharacterID != nil {
			resolved, err := resolveFactionLeader(ctx, tx, item.WorkID, *in.LeaderCharacterID)
			if err != nil {
				return err
			}
			leaderID = resolved
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
				return newFactionNameConflict(existing, name)
			}
		}

		rows, err := repo.UpdateWithVersion(ctx, id, repository.FactionUpdate{
			Name:              name,
			NameKey:           nameKey,
			ParentID:          parentID,
			LeaderCharacterID: leaderID,
			Description:       description,
			Version:           version,
		}, now)
		if err != nil {
			if repository.IsUniqueViolation(err, "factions_active_name_uq") {
				return newFactionNameConflict(nil, name)
			}
			return err
		}
		if rows == 0 {
			return model.ErrVersionConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Delete 在入向引用检查通过后软删除势力并发放 8 秒撤销令牌。
func (s *Factions) Delete(ctx context.Context, id, ifMatch string) (*DeleteFactionResult, error) {
	version, ok := ParseIfMatch(ifMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewFactions(tx)
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
		if len(children) > 0 {
			items := make([]model.ConflictItem, 0, len(children))
			for _, child := range children {
				items = append(items, model.ConflictItem{ID: child.ID, Name: child.Name, EntityType: RootTypeFaction})
			}
			return &model.HasChildrenError{Items: items}
		}
		members, err := repo.ActiveMembers(ctx, id)
		if err != nil {
			return err
		}
		if len(members) > 0 {
			items := make([]model.ReferenceItem, 0, len(members))
			for _, member := range members {
				name := member.Name
				items = append(items, model.ReferenceItem{
					Type:     "character_faction",
					EntityID: member.ID,
					Name:     &name,
					Field:    "faction_id",
					Value:    item.Name,
				})
			}
			return &model.ReferenceConflictError{Message: "势力仍有成员，无法删除", Items: items}
		}
		rows, err := repo.SoftDelete(ctx, id, model.FactionActionDelete, version, now)
		if err != nil {
			return err
		}
		if rows == 0 {
			return model.ErrAlreadyDeleted
		}
		return repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeFaction,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		})
	})
	if err != nil {
		return nil, err
	}
	return &DeleteFactionResult{ID: id, DeletedAt: now, UndoToken: token, ExpiresAt: expires}, nil
}

// resolveFactionParent 校验父级同作品且活跃，返回去空白后的 ID。
func resolveFactionParent(ctx context.Context, tx *gorm.DB, workID, raw string) (*string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return nil, &model.ValidationError{Field: "parent_id", Message: "parent_id 不能为空"}
	}
	if _, err := repository.NewFactions(tx).GetActiveByWork(ctx, workID, id); err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return nil, err
		}
		any, anyErr := repository.NewFactions(tx).GetAny(ctx, id)
		if anyErr != nil {
			if errors.Is(anyErr, model.ErrNotFound) {
				return nil, model.ErrParentDeleted
			}
			return nil, anyErr
		}
		if any.WorkID != workID {
			return nil, &model.CrossWorkReferenceError{Field: "parent_id", Message: "上级势力不属于当前作品"}
		}
		return nil, model.ErrParentDeleted
	}
	return &id, nil
}

// resolveFactionLeader 校验首领为当前作品活跃人物，返回去空白后的 ID。
func resolveFactionLeader(ctx context.Context, tx *gorm.DB, workID, raw string) (*string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return nil, &model.ValidationError{Field: "leader_character_id", Message: "leader_character_id 不能为空"}
	}
	if _, err := repository.NewCharacters(tx).GetActiveByWork(ctx, workID, id); err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return nil, err
		}
		any, anyErr := repository.NewCharacters(tx).GetAny(ctx, id)
		if anyErr != nil {
			if errors.Is(anyErr, model.ErrNotFound) {
				return nil, &model.ValidationError{Field: "leader_character_id", Message: "首领人物不存在"}
			}
			return nil, anyErr
		}
		if any.WorkID != workID {
			return nil, &model.CrossWorkReferenceError{Field: "leader_character_id", Message: "首领人物不属于当前作品"}
		}
		return nil, model.ErrParentDeleted
	}
	return &id, nil
}

func newFactionNameConflict(existing *model.Faction, name string) error {
	item := model.ConflictItem{Name: name, EntityType: RootTypeFaction}
	if existing != nil {
		item.ID = existing.ID
		item.Name = existing.Name
	}
	return &model.NameConflictError{Items: []model.ConflictItem{item}}
}

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

// Characters 负责人物档案的创建、查询、内联更新与删除。
type Characters struct {
	db  *gorm.DB
	now func() time.Time
}

const (
	RootTypeCharacter     = "character"
	CharacterActionDelete = "character_delete"
)

// DeleteCharacterResult 是 DELETE /characters/{id} 的响应。
type DeleteCharacterResult struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
	UndoToken string    `json:"undo_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewCharacters(db *gorm.DB) *Characters {
	return &Characters{db: db, now: time.Now}
}

type CharacterNameInput struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type CharacterSystemRefInput struct {
	CategoryID string `json:"category_id"`
	NodeID     string `json:"node_id"`
}

type CharacterLocationLinkInput struct {
	LocationID string  `json:"location_id"`
	Note       *string `json:"note"`
}

type CreateCharacterInput struct {
	WorkID        string
	Name          string
	Names         []CharacterNameInput
	Description   *string
	FactionID     *string
	SystemRefs    []CharacterSystemRefInput
	LocationLinks []CharacterLocationLinkInput
}

type UpdateCharacterInput struct {
	Name             *string
	Names            *[]CharacterNameInput
	Description      *string
	ClearDescription bool
	FactionID        *string
	ClearFaction     bool
	SystemRefs       *[]CharacterSystemRefInput
	LocationLinks    *[]CharacterLocationLinkInput
	IfMatch          string
}

type ListCharactersInput struct {
	WorkID       string
	Query        string
	SystemNodeID string
	FactionID    string
	LocationID   string
	ReviewStatus string
	Sort         string
	Page         int
	PageSize     int
}

type CharacterPage struct {
	Items    []model.CharacterSummary `json:"items"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
	Total    int64                    `json:"total"`
}

// List 返回当前作品的人物摘要；q 命中主名称或别名/称号。
func (s *Characters) List(ctx context.Context, in ListCharactersInput) (*CharacterPage, error) {
	filter, page, pageSize, err := parseCharacterFilter(in)
	if err != nil {
		return nil, err
	}
	repo := repository.NewCharacters(s.db)
	items, total, err := repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	summaries, err := s.summaries(ctx, repo, items)
	if err != nil {
		return nil, err
	}
	return &CharacterPage{Items: summaries, Page: page, PageSize: pageSize, Total: total}, nil
}

func parseCharacterFilter(in ListCharactersInput) (repository.CharacterFilter, int, int, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return repository.CharacterFilter{}, 0, 0, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	query := strings.TrimSpace(in.Query)
	if utf8.RuneCountInString(query) > 100 {
		return repository.CharacterFilter{}, 0, 0, &model.ValidationError{Field: "q", Message: "搜索词不能超过 100 个字符"}
	}
	if strings.TrimSpace(in.LocationID) != "" {
		return repository.CharacterFilter{}, 0, 0, &model.ValidationError{Field: "location_id", Message: "地点功能尚未开放"}
	}
	switch in.Sort {
	case "", "-updated_at", "updated_at", "name", "-name":
	default:
		return repository.CharacterFilter{}, 0, 0, &model.ValidationError{Field: "sort", Message: "sort 仅支持 -updated_at、updated_at、name、-name"}
	}
	reviewDB, err := model.DBReviewStatus(in.ReviewStatus)
	if err != nil {
		return repository.CharacterFilter{}, 0, 0, err
	}

	page, pageSize := normalizePage(in.Page, in.PageSize)
	return repository.CharacterFilter{
		WorkID:       workID,
		Query:        query,
		SystemNodeID: strings.TrimSpace(in.SystemNodeID),
		FactionID:    strings.TrimSpace(in.FactionID),
		ReviewDB:     reviewDB,
		Sort:         in.Sort,
		Page:         page,
		PageSize:     pageSize,
	}, page, pageSize, nil
}

// summaries 批量装配别名与境界摘要，避免列表 N+1。
func (s *Characters) summaries(ctx context.Context, repo *repository.Characters, items []model.Character) ([]model.CharacterSummary, error) {
	out := make([]model.CharacterSummary, 0, len(items))
	if len(items) == 0 {
		return out, nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	names, err := repo.ListNamesByCharacters(ctx, ids)
	if err != nil {
		return nil, err
	}
	refs, err := repo.ListSystemRefRows(ctx, ids)
	if err != nil {
		return nil, err
	}

	namesByCharacter := make(map[string][]model.CharacterNameView, len(items))
	for _, name := range names {
		namesByCharacter[name.CharacterID] = append(namesByCharacter[name.CharacterID], model.CharacterNameView{
			Kind:  name.Kind,
			Value: name.Value,
		})
	}
	summaryByCharacter := make(map[string]*string, len(items))
	for _, row := range refs {
		if !row.CategoryBuiltin {
			continue
		}
		path := row.NodeName
		if row.ParentName != nil && *row.ParentName != "" {
			path = *row.ParentName + "·" + row.NodeName
		}
		summaryByCharacter[row.CharacterID] = &path
	}

	for _, item := range items {
		views := namesByCharacter[item.ID]
		if views == nil {
			views = []model.CharacterNameView{}
		}
		out = append(out, model.CharacterSummary{
			ID:            item.ID,
			Name:          item.Name,
			Names:         views,
			Faction:       nil,
			SystemSummary: summaryByCharacter[item.ID],
			ImageThumbURL: nil,
			ReviewStatus:  model.APIReviewStatus(item.ReviewStatus),
			ReviewedAt:    item.ReviewedAt,
			UpdatedAt:     item.UpdatedAt,
			Version:       item.Version,
		})
	}
	return out, nil
}

// Get 返回人物详情；不存在或已删除统一 404。
func (s *Characters) Get(ctx context.Context, id string) (*model.CharacterDetail, error) {
	repo := repository.NewCharacters(s.db)
	item, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	names, err := repo.ListNames(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := repo.ListSystemRefRows(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return buildCharacterDetail(item, names, rows), nil
}

func buildCharacterDetail(item *model.Character, names []model.CharacterName, rows []repository.CharacterSystemRefRow) *model.CharacterDetail {
	views := make([]model.CharacterNameView, 0, len(names))
	for _, name := range names {
		views = append(views, model.CharacterNameView{Kind: name.Kind, Value: name.Value})
	}
	refs := make([]model.SystemRefView, 0, len(rows))
	for _, row := range rows {
		node := model.CharacterNodeView{ID: row.NodeID, Name: row.NodeName}
		if row.ParentID != nil {
			parentName := ""
			if row.ParentName != nil {
				parentName = *row.ParentName
			}
			node.Parent = &model.CharacterNodeParentView{ID: *row.ParentID, Name: parentName}
		}
		refs = append(refs, model.SystemRefView{
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			CategoryType: row.CategoryType,
			Node:         node,
		})
	}
	return &model.CharacterDetail{
		ID:            item.ID,
		WorkID:        item.WorkID,
		Name:          item.Name,
		Names:         views,
		Description:   item.Description,
		Faction:       nil,
		SystemRefs:    refs,
		LocationLinks: []model.LocationLinkView{},
		Image:         nil,
		ReviewStatus:  model.APIReviewStatus(item.ReviewStatus),
		ReviewedAt:    item.ReviewedAt,
		ReviewedBy:    item.ReviewedBy,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
		Version:       item.Version,
		DeletedAt:     item.DeletedAt,
	}
}

// Create 在事务内校验父作品、名称唯一与体系引用后写入人物。
func (s *Characters) Create(ctx context.Context, in CreateCharacterInput) (*model.CharacterDetail, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	name, nameKey, err := model.NormalizeCharacterName(in.Name)
	if err != nil {
		return nil, err
	}
	names, err := normalizeCharacterNames(in.Names)
	if err != nil {
		return nil, err
	}
	description, err := model.NormalizeCharacterDescription(in.Description)
	if err != nil {
		return nil, err
	}
	if in.FactionID != nil {
		return nil, &model.ValidationError{Field: "faction_id", Message: "势力功能尚未开放"}
	}
	if len(in.LocationLinks) > 0 {
		return nil, &model.ValidationError{Field: "location_links", Message: "地点功能尚未开放"}
	}
	refs, err := normalizeCharacterRefs(in.SystemRefs)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	item := &model.Character{
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
		repo := repository.NewCharacters(tx)
		if _, err := repository.NewWorks(tx).GetForUpdate(ctx, workID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrCharacterWorkUnavailable
			}
			return err
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
			return newNameConflict(existing, name)
		}
		if err := repo.Insert(ctx, item); err != nil {
			if repository.IsUniqueViolation(err, "characters_active_name_uq") {
				return newNameConflict(nil, name)
			}
			return err
		}
		if err := repo.ReplaceNames(ctx, item.ID, buildNameRows(item.ID, workID, names, now)); err != nil {
			return err
		}
		return s.replaceSystemRefs(ctx, tx, repo, item, refs, now)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, item.ID)
}

// Update 以内联字段子集更新；If-Match 缺失与版本不匹配分别返回 428/409。
func (s *Characters) Update(ctx context.Context, id string, in UpdateCharacterInput) (*model.CharacterDetail, error) {
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	if in.LocationLinks != nil && len(*in.LocationLinks) > 0 {
		return nil, &model.ValidationError{Field: "location_links", Message: "地点功能尚未开放"}
	}
	if in.FactionID != nil {
		return nil, &model.ValidationError{Field: "faction_id", Message: "势力功能尚未开放"}
	}

	now := s.now().UTC()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewCharacters(tx)
		item, err := repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if version != item.Version {
			return &model.VersionConflictError{CurrentVersion: item.Version}
		}

		name, nameKey := item.Name, item.NameKey
		if in.Name != nil {
			if name, nameKey, err = model.NormalizeCharacterName(*in.Name); err != nil {
				return err
			}
		}
		description := item.Description
		if in.ClearDescription {
			description = nil
		} else if in.Description != nil {
			if description, err = model.NormalizeCharacterDescription(in.Description); err != nil {
				return err
			}
		}
		factionID := item.FactionID
		if in.ClearFaction {
			factionID = nil
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
				return newNameConflict(existing, name)
			}
		}

		rows, err := repo.UpdateWithVersion(ctx, id, repository.CharacterUpdate{
			Name:        name,
			NameKey:     nameKey,
			Description: description,
			FactionID:   factionID,
			Version:     version,
		}, now)
		if err != nil {
			if repository.IsUniqueViolation(err, "characters_active_name_uq") {
				return newNameConflict(nil, name)
			}
			return err
		}
		if rows == 0 {
			return model.ErrVersionConflict
		}

		if in.Names != nil {
			names, err := normalizeCharacterNames(*in.Names)
			if err != nil {
				return err
			}
			if err := repo.ReplaceNames(ctx, id, buildNameRows(id, item.WorkID, names, now)); err != nil {
				return err
			}
		}
		if in.SystemRefs != nil {
			refs, err := normalizeCharacterRefs(*in.SystemRefs)
			if err != nil {
				return err
			}
			if err := s.replaceSystemRefs(ctx, tx, repo, item, refs, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Delete 软删除人物并发放 8 秒撤销令牌；入向引用命中时整单阻断。
func (s *Characters) Delete(ctx context.Context, id, ifMatch string) (*DeleteCharacterResult, error) {
	version, ok := ParseIfMatch(ifMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewCharacters(tx)
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
		refs, err := characterInboundReferences(ctx, tx, id)
		if err != nil {
			return err
		}
		if len(refs) > 0 {
			return &model.ReferenceConflictError{Message: "人物仍被其他数据引用，无法删除", Items: refs}
		}
		rows, err := repo.SoftDelete(ctx, id, CharacterActionDelete, version, now)
		if err != nil {
			return err
		}
		if rows == 0 {
			return model.ErrAlreadyDeleted
		}
		return repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeCharacter,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		})
	})
	if err != nil {
		return nil, err
	}
	return &DeleteCharacterResult{ID: id, DeletedAt: now, UndoToken: token, ExpiresAt: expires}, nil
}

// characterInboundReferences 是入向引用检查框架：势力首领与榜单席位表尚未创建，
// CHG-006 / CHG-008 接入后在此返回阻断明细。
func characterInboundReferences(_ context.Context, _ *gorm.DB, _ string) ([]model.ReferenceItem, error) {
	return nil, nil
}

// replaceSystemRefs 校验类别/节点归属与启用状态后整组替换引用。
func (s *Characters) replaceSystemRefs(ctx context.Context, tx *gorm.DB, repo *repository.Characters, item *model.Character, refs []CharacterSystemRefInput, now time.Time) error {
	if len(refs) == 0 {
		return repo.ReplaceSystemRefs(ctx, item.ID, nil)
	}
	system := repository.NewSystem(tx)
	rows := make([]model.CharacterSystemRef, 0, len(refs))
	for _, ref := range refs {
		category, err := system.GetCategory(ctx, ref.CategoryID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if category.WorkID != item.WorkID {
			return &model.CrossWorkReferenceError{Field: "system_refs", Message: "类别不属于当前作品"}
		}
		if !category.IsEnabled {
			return &model.CategoryDisabledError{CategoryName: category.Name}
		}
		node, err := system.GetNode(ctx, ref.NodeID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if node.WorkID != item.WorkID {
			return &model.CrossWorkReferenceError{Field: "system_refs", Message: "节点不属于当前作品"}
		}
		if node.CategoryID != category.ID {
			return model.ErrNodeParentMismatch
		}
		rows = append(rows, model.CharacterSystemRef{
			ID:          model.NewID(),
			WorkID:      item.WorkID,
			CharacterID: item.ID,
			CategoryID:  category.ID,
			NodeID:      node.ID,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return repo.ReplaceSystemRefs(ctx, item.ID, rows)
}

func newNameConflict(existing *model.Character, name string) error {
	item := model.ConflictItem{Name: name, EntityType: "character"}
	if existing != nil {
		item.ID = existing.ID
		item.Name = existing.Name
	}
	return &model.NameConflictError{Items: []model.ConflictItem{item}}
}

type normalizedCharacterName struct {
	Kind      string
	Value     string
	ValueKey  string
	SortOrder int
}

// normalizeCharacterNames 校验 kind/长度/条数，并按 (kind, value_key) 去重后保留提交顺序。
func normalizeCharacterNames(inputs []CharacterNameInput) ([]normalizedCharacterName, error) {
	if len(inputs) > model.MaxCharacterNames {
		return nil, &model.ValidationError{Field: "names", Message: "别名与称号合计不能超过 100 条"}
	}
	seen := make(map[string]struct{}, len(inputs))
	out := make([]normalizedCharacterName, 0, len(inputs))
	for _, input := range inputs {
		kind := strings.TrimSpace(input.Kind)
		if kind != model.CharacterNameKindAlias && kind != model.CharacterNameKindTitle {
			return nil, &model.ValidationError{Field: "names", Message: "kind 仅支持 alias 或 title"}
		}
		value, valueKey, err := model.NormalizeCharacterNameValue(input.Value, "names")
		if err != nil {
			return nil, err
		}
		unique := kind + "\x00" + valueKey
		if _, dup := seen[unique]; dup {
			continue
		}
		seen[unique] = struct{}{}
		out = append(out, normalizedCharacterName{
			Kind:      kind,
			Value:     value,
			ValueKey:  valueKey,
			SortOrder: len(out),
		})
	}
	return out, nil
}

func buildNameRows(characterID, workID string, names []normalizedCharacterName, now time.Time) []model.CharacterName {
	rows := make([]model.CharacterName, 0, len(names))
	for _, name := range names {
		rows = append(rows, model.CharacterName{
			ID:          model.NewID(),
			CharacterID: characterID,
			WorkID:      workID,
			Kind:        name.Kind,
			Value:       name.Value,
			ValueKey:    name.ValueKey,
			SortOrder:   name.SortOrder,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return rows
}

// normalizeCharacterRefs 校验必填与类别唯一，返回去空白后的提交顺序。
func normalizeCharacterRefs(inputs []CharacterSystemRefInput) ([]CharacterSystemRefInput, error) {
	out := make([]CharacterSystemRefInput, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		categoryID := strings.TrimSpace(input.CategoryID)
		nodeID := strings.TrimSpace(input.NodeID)
		if categoryID == "" || nodeID == "" {
			return nil, &model.ValidationError{Field: "system_refs", Message: "category_id 与 node_id 不能为空"}
		}
		if _, dup := seen[categoryID]; dup {
			return nil, &model.ValidationError{Field: "system_refs", Message: "同一类别只能提交一个节点"}
		}
		seen[categoryID] = struct{}{}
		out = append(out, CharacterSystemRefInput{CategoryID: categoryID, NodeID: nodeID})
	}
	return out, nil
}

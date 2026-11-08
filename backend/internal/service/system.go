package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

// 体系回收站根类型与软删除动作，与 ADR-004 保持一致。
const (
	RootTypeSystemCategory = "system_category"
	RootTypeSystemNode     = "system_node"

	ActionCategoryDelete = "category_delete"
	ActionNodeDelete     = "delete"
)

// System 负责体系类别与节点的读写、引用阻断与回收站联动。
type System struct {
	db  *gorm.DB
	now func() time.Time
}

func NewSystem(db *gorm.DB) *System {
	return &System{db: db, now: time.Now}
}

type CreateCategoryInput struct {
	WorkID       string
	Name         string
	CategoryType string
	IsEnabled    *bool
}

type UpdateCategoryInput struct {
	Name      *string
	IsEnabled *bool
	IfMatch   string
}

type DeleteCategoryInput struct {
	IfMatch string
}

type DeleteCategoryResult struct {
	Deleted       bool      `json:"deleted"`
	UndoToken     string    `json:"undo_token"`
	AffectedNodes int64     `json:"affected_nodes"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// ListCategories 返回作品下未删除类别；includeDisabled=false 时仅返回启用类别。
func (s *System) ListCategories(ctx context.Context, workID string, includeDisabled bool) ([]model.SystemCategory, error) {
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	if _, err := repository.NewWorks(s.db).Get(ctx, workID); err != nil {
		return nil, err
	}
	items, err := repository.NewSystem(s.db).ListCategories(ctx, workID, includeDisabled)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.SystemCategory{}
	}
	return items, nil
}

// CreateCategory 在事务内校验上限与重名后写入自定义类别。
func (s *System) CreateCategory(ctx context.Context, in CreateCategoryInput) (*model.SystemCategory, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	display, nameKey, err := model.NormalizeKey(in.Name, "name")
	if err != nil {
		return nil, err
	}

	categoryType := in.CategoryType
	if categoryType == "" {
		categoryType = model.CategoryTypeNormal
	}
	if categoryType != model.CategoryTypeNormal && categoryType != model.CategoryTypeEra {
		return nil, &model.ValidationError{Field: "category_type", Message: "category_type 仅支持 normal 或 era"}
	}
	enabled := true
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}

	if _, err := repository.NewWorks(s.db).Get(ctx, workID); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	item := &model.SystemCategory{
		ID:           model.NewID(),
		WorkID:       workID,
		Name:         display,
		NameKey:      nameKey,
		CategoryType: categoryType,
		IsBuiltin:    false,
		IsEnabled:    enabled,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewSystem(tx)
		count, err := repo.CountActiveCategories(ctx, workID)
		if err != nil {
			return err
		}
		if count >= model.MaxSystemCategories {
			return model.ErrCategoryLimit
		}
		taken, err := repo.CategoryNameTaken(ctx, workID, nameKey, "")
		if err != nil {
			return err
		}
		if taken {
			return model.ErrNameConflict
		}
		sortOrder, err := repo.NextCategorySortOrder(ctx, workID)
		if err != nil {
			return err
		}
		item.SortOrder = sortOrder
		if err := repo.InsertCategory(ctx, item); err != nil {
			if repository.IsUniqueViolation(err, "system_categories_work_name_uq") {
				return model.ErrNameConflict
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// UpdateCategory 处理改名与启停；内置类别只允许无操作的 is_enabled=true。
func (s *System) UpdateCategory(ctx context.Context, id string, in UpdateCategoryInput) (*model.SystemCategory, error) {
	repo := repository.NewSystem(s.db)
	item, err := repo.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok || version != item.Version {
		return nil, model.ErrVersionConflict
	}

	if item.IsBuiltin {
		if in.Name != nil {
			return nil, model.ErrCategoryBuiltinImmutable
		}
		if in.IsEnabled != nil && !*in.IsEnabled {
			return nil, model.ErrCategoryBuiltinImmutable
		}
	}

	name, nameKey := item.Name, item.NameKey
	if in.Name != nil {
		name, nameKey, err = model.NormalizeKey(*in.Name, "name")
		if err != nil {
			return nil, err
		}
	}
	enabled := item.IsEnabled
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}

	if name == item.Name && enabled == item.IsEnabled {
		return item, nil
	}
	if name != item.Name {
		taken, err := repo.CategoryNameTaken(ctx, item.WorkID, nameKey, id)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, model.ErrNameConflict
		}
	}
	if item.IsEnabled && !enabled {
		if err := s.assertCategoryUnreferenced(ctx, repo, id, "停用"); err != nil {
			return nil, err
		}
	}

	now := s.now().UTC()
	rows, err := repo.UpdateCategory(ctx, id, name, nameKey, enabled, version, now)
	if err != nil {
		if repository.IsUniqueViolation(err, "system_categories_work_name_uq") {
			return nil, model.ErrNameConflict
		}
		return nil, err
	}
	if rows == 0 {
		return nil, model.ErrVersionConflict
	}
	return repo.GetCategory(ctx, id)
}

// DeleteCategory 引用检查通过后整组软删除类别与节点，并发放撤销令牌。
func (s *System) DeleteCategory(ctx context.Context, id string, in DeleteCategoryInput) (*DeleteCategoryResult, error) {
	repo := repository.NewSystem(s.db)
	item, err := repo.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.IsBuiltin {
		return nil, model.ErrCategoryBuiltinImmutable
	}
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok || version != item.Version {
		return nil, model.ErrVersionConflict
	}
	if err := s.assertCategoryUnreferenced(ctx, repo, id, "删除"); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()
	var affectedNodes int64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		affectedNodes, err = repository.NewSystem(tx).SoftDeleteCategoryGroup(ctx, id, version, ActionCategoryDelete, now)
		if err != nil {
			return err
		}
		return repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeSystemCategory,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		})
	})
	if err != nil {
		return nil, err
	}
	return &DeleteCategoryResult{
		Deleted:       true,
		UndoToken:     token,
		AffectedNodes: affectedNodes,
		ExpiresAt:     expires,
	}, nil
}

// assertCategoryUnreferenced 构造 REFERENCE_CONFLICT details；0003 无人物表，name 固定 null。
func (s *System) assertCategoryUnreferenced(ctx context.Context, repo *repository.System, categoryID, action string) error {
	rows, err := repo.ListReferencesByCategory(ctx, categoryID)
	if err != nil {
		return err
	}
	return referenceConflict(rows, "类别", action)
}

type CreateNodeInput struct {
	CategoryID   string
	ParentID     *string
	Name         string
	EraStartText *string
	EraEndText   *string
}

type UpdateNodeInput struct {
	Name         *string
	SortOrder    *int
	EraStartText *string
	ClearEraStart bool
	EraEndText   *string
	ClearEraEnd  bool
	IfMatch      string
}

type DeleteNodeInput struct {
	IfMatch string
}

type DeleteNodeResult struct {
	Deleted   bool      `json:"deleted"`
	UndoToken string    `json:"undo_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ReorderNodesInput struct {
	CategoryID string
	ParentID   *string
	OrderedIDs []string
}

// ListNodes 默认返回两级树；flat=true 或带 review 筛选时返回平铺列表。
func (s *System) ListNodes(ctx context.Context, categoryID string, flat bool, reviewStatus string) ([]model.SystemNode, error) {
	repo := repository.NewSystem(s.db)
	if _, err := repo.GetCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	reviewDB, err := model.DBReviewStatus(reviewStatus)
	if err != nil {
		return nil, err
	}
	nodes, err := repo.ListNodes(ctx, categoryID, reviewDB)
	if err != nil {
		return nil, err
	}
	for i := range nodes {
		nodes[i].ReviewStatus = model.APIReviewStatus(nodes[i].ReviewStatus)
	}
	if flat || reviewDB != "" {
		model.SortSystemNodes(nodes)
		return nodes, nil
	}
	return model.AssembleNodeTree(nodes)
}

// CreateNode 校验类别启用、两级层级、纪元字段与同级唯一后追加节点。
func (s *System) CreateNode(ctx context.Context, in CreateNodeInput) (*model.SystemNode, error) {
	repo := repository.NewSystem(s.db)
	category, err := repo.GetCategory(ctx, in.CategoryID)
	if err != nil {
		return nil, err
	}
	if !category.IsEnabled {
		return nil, model.ErrCategoryDisabled
	}

	var parent *model.SystemNode
	if in.ParentID != nil {
		parent, err = repo.GetNode(ctx, *in.ParentID)
		if err != nil {
			return nil, model.ErrNodeParentMismatch
		}
		if parent.CategoryID != category.ID || parent.WorkID != category.WorkID {
			return nil, model.ErrNodeParentMismatch
		}
		if parent.ParentID != nil {
			return nil, model.ErrNodeLevelLimit
		}
	}

	display, nameKey, err := model.NormalizeKey(in.Name, "name")
	if err != nil {
		return nil, err
	}
	eraStart, _, err := applyEraField(category, nil, in.EraStartText != nil, false, in.EraStartText, "era_start_text")
	if err != nil {
		return nil, err
	}
	eraEnd, _, err := applyEraField(category, nil, in.EraEndText != nil, false, in.EraEndText, "era_end_text")
	if err != nil {
		return nil, err
	}

	taken, err := repo.NodeNameTaken(ctx, category.ID, in.ParentID, nameKey, "")
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, model.ErrNameConflict
	}
	sortOrder, err := repo.NextNodeSortOrder(ctx, category.ID, in.ParentID)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	node := &model.SystemNode{
		ID:           model.NewID(),
		WorkID:       category.WorkID,
		CategoryID:   category.ID,
		ParentID:     in.ParentID,
		Name:         display,
		NameKey:      nameKey,
		SortOrder:    sortOrder,
		EraStartText: eraStart,
		EraEndText:   eraEnd,
		ReviewStatus: model.ReviewStatusReviewed,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.InsertNode(ctx, node); err != nil {
		if repository.IsUniqueViolation(err, "system_nodes_sibling_name_uq") {
			return nil, model.ErrNameConflict
		}
		return nil, err
	}
	node.ReviewStatus = model.APIReviewStatus(node.ReviewStatus)
	return node, nil
}

// UpdateNode 更新名称、排序或纪元文本；父级与节点类型不可改。
func (s *System) UpdateNode(ctx context.Context, id string, in UpdateNodeInput) (*model.SystemNode, error) {
	repo := repository.NewSystem(s.db)
	node, err := repo.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}
	category, err := repo.GetCategory(ctx, node.CategoryID)
	if err != nil {
		return nil, err
	}
	if !category.IsEnabled {
		return nil, model.ErrCategoryDisabled
	}
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok || version != node.Version {
		return nil, model.ErrVersionConflict
	}

	name, nameKey := node.Name, node.NameKey
	if in.Name != nil {
		name, nameKey, err = model.NormalizeKey(*in.Name, "name")
		if err != nil {
			return nil, err
		}
	}
	sortOrder := node.SortOrder
	if in.SortOrder != nil {
		if *in.SortOrder < 0 {
			return nil, &model.ValidationError{Field: "sort_order", Message: "sort_order 不能为负数"}
		}
		sortOrder = *in.SortOrder
	}
	eraStart, changed, err := applyEraField(category, node.EraStartText, in.EraStartText != nil, in.ClearEraStart, in.EraStartText, "era_start_text")
	if err != nil {
		return nil, err
	}
	if !changed {
		eraStart = node.EraStartText
	}
	eraEnd, changed, err := applyEraField(category, node.EraEndText, in.EraEndText != nil, in.ClearEraEnd, in.EraEndText, "era_end_text")
	if err != nil {
		return nil, err
	}
	if !changed {
		eraEnd = node.EraEndText
	}

	if name != node.Name {
		taken, err := repo.NodeNameTaken(ctx, node.CategoryID, node.ParentID, nameKey, id)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, model.ErrNameConflict
		}
	}

	now := s.now().UTC()
	rows, err := repo.UpdateNode(ctx, id, repository.NodeUpdate{
		Name:      name,
		NameKey:   nameKey,
		SortOrder: sortOrder,
		EraStart:  eraStart,
		EraEnd:    eraEnd,
		Version:   version,
	}, now)
	if err != nil {
		if repository.IsUniqueViolation(err, "system_nodes_sibling_name_uq") {
			return nil, model.ErrNameConflict
		}
		return nil, err
	}
	if rows == 0 {
		return nil, model.ErrVersionConflict
	}
	updated, err := repo.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}
	updated.ReviewStatus = model.APIReviewStatus(updated.ReviewStatus)
	return updated, nil
}

// DeleteNode 有子节点或人物引用时拒绝；通过后软删除自身并发放撤销令牌。
func (s *System) DeleteNode(ctx context.Context, id string, in DeleteNodeInput) (*DeleteNodeResult, error) {
	repo := repository.NewSystem(s.db)
	node, err := repo.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}
	category, err := repo.GetCategory(ctx, node.CategoryID)
	if err != nil {
		return nil, err
	}
	if !category.IsEnabled {
		return nil, model.ErrCategoryDisabled
	}
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok || version != node.Version {
		return nil, model.ErrVersionConflict
	}
	children, err := repo.CountNodeChildren(ctx, id)
	if err != nil {
		return nil, err
	}
	if children > 0 {
		return nil, model.ErrHasChildren
	}
	if err := s.assertNodeUnreferenced(ctx, repo, node, "删除"); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if rows, err := repository.NewSystem(tx).SoftDeleteNode(ctx, id, version, ActionNodeDelete, now); err != nil {
			return err
		} else if rows == 0 {
			return model.ErrVersionConflict
		}
		return repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeSystemNode,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		})
	})
	if err != nil {
		return nil, err
	}
	return &DeleteNodeResult{Deleted: true, UndoToken: token, ExpiresAt: expires}, nil
}

// ReorderNodes 校验同级集合完整后按给定顺序持久化。
func (s *System) ReorderNodes(ctx context.Context, in ReorderNodesInput) error {
	repo := repository.NewSystem(s.db)
	category, err := repo.GetCategory(ctx, in.CategoryID)
	if err != nil {
		return err
	}
	if !category.IsEnabled {
		return model.ErrCategoryDisabled
	}
	if in.ParentID != nil {
		parent, err := repo.GetNode(ctx, *in.ParentID)
		if err != nil {
			return model.ErrNodeParentMismatch
		}
		if parent.CategoryID != category.ID || parent.WorkID != category.WorkID {
			return model.ErrNodeParentMismatch
		}
		if parent.ParentID != nil {
			return model.ErrNodeLevelLimit
		}
	}

	siblings, err := repo.SiblingIDs(ctx, in.CategoryID, in.ParentID)
	if err != nil {
		return err
	}
	if err := model.ValidateSiblingOrder(siblings, in.OrderedIDs); err != nil {
		return err
	}
	return repo.ReorderNodes(ctx, in.OrderedIDs, s.now().UTC())
}

// assertNodeUnreferenced 构造单个节点引用阻断明细。
func (s *System) assertNodeUnreferenced(ctx context.Context, repo *repository.System, node *model.SystemNode, action string) error {
	rows, err := repo.ListReferencesByNode(ctx, node.ID)
	if err != nil {
		return err
	}
	return referenceConflict(rows, "节点", action)
}

// referenceConflict 汇总引用并按人物去重，供停用/删除/永久删除阻断。
func referenceConflict(rows []repository.ReferenceRow, subject, action string) error {
	if len(rows) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rows))
	items := make([]model.ReferenceItem, 0, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.CharacterID]; ok {
			continue
		}
		seen[row.CharacterID] = struct{}{}
		items = append(items, model.ReferenceItem{
			Type:     "character_system_ref",
			EntityID: row.CharacterID,
			Field:    "node",
			Value:    row.NodeName,
			InTrash:  false,
		})
	}
	return &model.ReferenceConflictError{
		Message: fmt.Sprintf("该%s仍有 %d 人引用，无法%s", subject, len(items), action),
		Items:   items,
	}
}

// applyEraField 处理纪元文本的可选更新；普通类别出现非空值视为字段校验错误。
// changed 表示本次是否应写入（false 时沿用当前值）。
func applyEraField(category *model.SystemCategory, current *string, provided, clear bool, raw *string, field string) (*string, bool, error) {
	if !provided && !clear {
		return current, false, nil
	}
	if clear {
		return nil, true, nil
	}
	if category.CategoryType != model.CategoryTypeEra {
		if raw != nil && strings.TrimSpace(*raw) != "" {
			return nil, false, &model.ValidationError{Field: field, Message: "普通类别不支持纪元文本"}
		}
		return nil, true, nil
	}
	value, err := model.NormalizeEraText(raw, field)
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

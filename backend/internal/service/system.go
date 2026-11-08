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
	if len(rows) == 0 {
		return nil
	}
	items := make([]model.ReferenceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.ReferenceItem{
			Type:     "character_system_ref",
			EntityID: row.CharacterID,
			Field:    "node",
			Value:    row.NodeName,
			InTrash:  false,
		})
	}
	return &model.ReferenceConflictError{
		Message: fmt.Sprintf("该类别仍有 %d 人引用，无法%s", len(items), action),
		Items:   items,
	}
}

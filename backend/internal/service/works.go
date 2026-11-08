package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	BuiltinAction   = "work_delete"
	RootTypeWork    = "work"
)

// Works 负责作品生命周期：创建、查询、改名、软删除与撤销令牌发放。
type Works struct {
	db        *gorm.DB
	snapshots *Snapshot
	now       func() time.Time
}

func NewWorks(db *gorm.DB, snapshots *Snapshot) *Works {
	return &Works{db: db, snapshots: snapshots, now: time.Now}
}

// WorkPage 是 GET /works 的分页响应。
type WorkPage struct {
	Items    []model.Work `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Total    int64        `json:"total"`
}

type CreateWorkInput struct {
	Name        string
	Description *string
}

type UpdateWorkInput struct {
	Name             *string
	Description      *string
	ClearDescription bool
	IfMatch          string
}

type DeleteWorkInput struct {
	ConfirmName string
	IfMatch     string
}

type DeleteWorkResult struct {
	Deleted    bool      `json:"deleted"`
	UndoToken  string    `json:"undo_token"`
	SnapshotID string    `json:"snapshot_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (s *Works) List(ctx context.Context, q string, page, pageSize int) (*WorkPage, error) {
	page, pageSize = normalizePage(page, pageSize)
	items, total, err := repository.NewWorks(s.db).List(ctx, q, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &WorkPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Works) Get(ctx context.Context, id string) (*model.Work, error) {
	return repository.NewWorks(s.db).Get(ctx, id)
}

// Create 在同一事务内创建作品与内置「境界」类别。
func (s *Works) Create(ctx context.Context, in CreateWorkInput) (*model.Work, error) {
	name, nameKey, err := model.NormalizeName(in.Name)
	if err != nil {
		return nil, err
	}
	description, err := model.NormalizeDescription(in.Description)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	work := &model.Work{
		ID:          model.NewID(),
		Name:        name,
		NameKey:     nameKey,
		Description: description,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	category := &model.SystemCategory{
		ID:           model.NewID(),
		WorkID:       work.ID,
		Name:         model.BuiltinRealmCategory,
		NameKey:      strings.ToLower(model.BuiltinRealmCategory),
		CategoryType: "normal",
		IsBuiltin:    true,
		IsEnabled:    true,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewWorks(tx)
		if err := repo.Insert(ctx, work); err != nil {
			if repository.IsUniqueViolation(err, "works_name_key_uq") {
				return model.ErrNameConflict
			}
			return err
		}
		if err := repo.InsertCategory(ctx, category); err != nil {
			return err
		}
		// 新建作品自动成为当前作品，避免用户创建后仍显示「未选择作品」。
		return repository.NewSettings(tx).SetCurrentWork(ctx, &work.ID, now)
	})
	if err != nil {
		return nil, err
	}
	return work, nil
}

// Update 使用 If-Match 版本的乐观锁；版本不匹配统一 409。
func (s *Works) Update(ctx context.Context, id string, in UpdateWorkInput) (*model.Work, error) {
	repo := repository.NewWorks(s.db)
	work, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	version, ok := ParseIfMatch(in.IfMatch)
	if !ok || version != work.Version {
		return nil, model.ErrVersionConflict
	}

	name, nameKey := work.Name, work.NameKey
	if in.Name != nil {
		name, nameKey, err = model.NormalizeName(*in.Name)
		if err != nil {
			return nil, err
		}
	}
	description := work.Description
	if in.ClearDescription {
		description = nil
	} else if in.Description != nil {
		description, err = model.NormalizeDescription(in.Description)
		if err != nil {
			return nil, err
		}
	}

	now := s.now().UTC()
	rows, err := repo.UpdateWithVersion(ctx, id, name, nameKey, description, version, now)
	if err != nil {
		if repository.IsUniqueViolation(err, "works_name_key_uq") {
			return nil, model.ErrNameConflict
		}
		return nil, err
	}
	if rows == 0 {
		return nil, model.ErrVersionConflict
	}
	return repo.Get(ctx, id)
}

// Delete 先快照后软删除；快照失败时作品保持原状（EX-23）。
func (s *Works) Delete(ctx context.Context, id string, in DeleteWorkInput) (*DeleteWorkResult, error) {
	repo := repository.NewWorks(s.db)
	work, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.ConfirmName) != work.Name {
		return nil, &model.ValidationError{Field: "confirm_name", Message: "确认名称与作品名不一致"}
	}
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok || version != work.Version {
		return nil, model.ErrVersionConflict
	}

	snapshot, err := s.snapshots.Create(ctx, model.SnapshotReasonWorkDelete)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := repository.NewWorks(tx).SoftDeleteGroup(ctx, id, BuiltinAction, now); err != nil {
			return err
		}
		if err := repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeWork,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		}); err != nil {
			return err
		}
		return repository.NewSettings(tx).ClearCurrentWorkIf(ctx, id, now)
	})
	if err != nil {
		return nil, err
	}

	return &DeleteWorkResult{
		Deleted:    true,
		UndoToken:  token,
		SnapshotID: snapshot.ID,
		ExpiresAt:  expires,
	}, nil
}

// ParseIfMatch 解析 `If-Match: "3"`；缺失、非数字或小于 1 均视为无效。
func ParseIfMatch(raw string) (int, bool) {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, `"`)
	if value == "" {
		return 0, false
	}
	version, err := strconv.Atoi(value)
	if err != nil || version < 1 {
		return 0, false
	}
	return version, true
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

func newUndoToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// errNotFoundAs alreadyDeleted 将「目标已不存在」统一为重复操作的语义。
func alreadyDeleted(err error) error {
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrAlreadyDeleted
	}
	return err
}

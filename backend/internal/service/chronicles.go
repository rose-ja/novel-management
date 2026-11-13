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

// RootTypeChronicle 仅用于撤销令牌；大事记不作为回收站根展示。
const RootTypeChronicle = "chronicle"

// Chronicles 负责人物/势力大事记的校验、排序与轻撤销。
type Chronicles struct {
	db  *gorm.DB
	now func() time.Time
}

func NewChronicles(db *gorm.DB) *Chronicles {
	return &Chronicles{db: db, now: time.Now}
}

type ChronicleInput struct {
	OwnerType string
	OwnerID   string
	Text      string
	TimeLabel *string
}

type UpdateChronicleInput struct {
	Text           *string
	TimeLabel      *string
	ClearTimeLabel bool
	SortOrder      *int
	IfMatch        string
}

type DeleteChronicleResult struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
	UndoToken string    `json:"undo_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// List 返回归属实体未删除的大事记。
func (s *Chronicles) List(ctx context.Context, ownerType, ownerID string) ([]model.Chronicle, error) {
	if err := s.requireOwner(ctx, s.db, ownerType, ownerID); err != nil {
		return nil, err
	}
	items, err := repository.NewChronicles(s.db).ListActive(ctx, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.Chronicle{}
	}
	return items, nil
}

// Create 追加一条大事记到归属实体末尾。
func (s *Chronicles) Create(ctx context.Context, in ChronicleInput) (*model.Chronicle, error) {
	text, timeLabel, err := normalizeChronicle(in.Text, in.TimeLabel)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	var created model.Chronicle
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workID, err := s.ownerWorkID(ctx, tx, in.OwnerType, in.OwnerID)
		if err != nil {
			return err
		}
		repo := repository.NewChronicles(tx)
		order, err := repo.NextSortOrder(ctx, in.OwnerType, in.OwnerID)
		if err != nil {
			return err
		}
		created = model.Chronicle{
			ID:        model.NewID(),
			WorkID:    workID,
			OwnerType: in.OwnerType,
			OwnerID:   in.OwnerID,
			Text:      text,
			TimeLabel: timeLabel,
			SortOrder: order,
			Version:   1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return repo.Insert(ctx, &created)
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// ownerWorkID 锁定归属实体并返回其作品 ID；支持人物与势力。
func (s *Chronicles) ownerWorkID(ctx context.Context, tx *gorm.DB, ownerType, ownerID string) (string, error) {
	switch ownerType {
	case model.ChronicleOwnerCharacter:
		item, err := repository.NewCharacters(tx).GetForUpdate(ctx, ownerID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return "", model.ErrNotFound
			}
			return "", err
		}
		return item.WorkID, nil
	case model.ChronicleOwnerFaction:
		item, err := repository.NewFactions(tx).GetForUpdate(ctx, ownerID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				if _, anyErr := repository.NewFactions(tx).GetAny(ctx, ownerID); anyErr == nil {
					return "", model.ErrParentDeleted
				}
				return "", model.ErrNotFound
			}
			return "", err
		}
		return item.WorkID, nil
	default:
		return "", model.ErrChronicleOwnerUnsupported
	}
}

// Update 编辑文本/时间标签/顺序；If-Match 必填。
func (s *Chronicles) Update(ctx context.Context, id string, in UpdateChronicleInput) (*model.Chronicle, error) {
	version, ok := ParseIfMatch(in.IfMatch)
	if !ok {
		return nil, model.ErrPreconditionRequired
	}
	fields := map[string]any{}
	if in.Text != nil {
		text := strings.TrimSpace(*in.Text)
		if text == "" {
			return nil, &model.ValidationError{Field: "text", Message: "大事记内容不能为空"}
		}
		if utf8.RuneCountInString(text) > model.MaxChronicleRunes {
			return nil, &model.ValidationError{Field: "text", Message: "大事记内容不能超过 2000 个字符"}
		}
		fields["text"] = text
	}
	if in.ClearTimeLabel {
		fields["time_label"] = nil
	} else if in.TimeLabel != nil {
		label := strings.TrimSpace(*in.TimeLabel)
		if utf8.RuneCountInString(label) > model.MaxChronicleTimeLabel {
			return nil, &model.ValidationError{Field: "time_label", Message: "时间标签不能超过 200 个字符"}
		}
		if label == "" {
			fields["time_label"] = nil
		} else {
			fields["time_label"] = label
		}
	}
	if in.SortOrder != nil {
		if *in.SortOrder < 0 {
			return nil, &model.ValidationError{Field: "sort_order", Message: "顺序不能为负数"}
		}
		fields["sort_order"] = *in.SortOrder
	}

	now := s.now().UTC()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewChronicles(tx)
		item, err := repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if version != item.Version {
			return &model.VersionConflictError{CurrentVersion: item.Version}
		}
		if len(fields) == 0 {
			return nil
		}
		return repo.Update(ctx, id, fields, version, now)
	})
	if err != nil {
		return nil, err
	}
	return repository.NewChronicles(s.db).GetActive(ctx, id)
}

// Delete 软删除单条大事记并发放 8 秒撤销令牌。
func (s *Chronicles) Delete(ctx context.Context, id string) (*DeleteChronicleResult, error) {
	now := s.now().UTC()
	expires := now.Add(model.UndoWindowSeconds * time.Second)
	token := newUndoToken()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewChronicles(tx)
		item, err := repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		rows, err := repo.SoftDelete(ctx, id, item.OwnerID, model.ChronicleActionDelete, now)
		if err != nil {
			return err
		}
		if rows == 0 {
			return model.ErrAlreadyDeleted
		}
		return repository.NewUndo(tx).Insert(ctx, &model.UndoToken{
			Token:     token,
			RootType:  RootTypeChronicle,
			RootID:    id,
			CreatedAt: now,
			ExpiresAt: expires,
		})
	})
	if err != nil {
		return nil, err
	}
	return &DeleteChronicleResult{ID: id, DeletedAt: now, UndoToken: token, ExpiresAt: expires}, nil
}

func (s *Chronicles) requireOwner(ctx context.Context, db *gorm.DB, ownerType, ownerID string) error {
	if ownerType != model.ChronicleOwnerCharacter && ownerType != model.ChronicleOwnerFaction {
		return model.ErrChronicleOwnerUnsupported
	}
	if strings.TrimSpace(ownerID) == "" {
		return &model.ValidationError{Field: "owner_id", Message: "owner_id 不能为空"}
	}
	if ownerType == model.ChronicleOwnerFaction {
		if _, err := repository.NewFactions(db).Get(ctx, ownerID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				if _, anyErr := repository.NewFactions(db).GetAny(ctx, ownerID); anyErr == nil {
					return model.ErrParentDeleted
				}
				return model.ErrNotFound
			}
			return err
		}
		return nil
	}
	if _, err := repository.NewCharacters(db).Get(ctx, ownerID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.ErrNotFound
		}
		return err
	}
	return nil
}

func normalizeChronicle(rawText string, rawLabel *string) (string, *string, error) {
	text := strings.TrimSpace(rawText)
	if text == "" {
		return "", nil, &model.ValidationError{Field: "text", Message: "大事记内容不能为空"}
	}
	if utf8.RuneCountInString(text) > model.MaxChronicleRunes {
		return "", nil, &model.ValidationError{Field: "text", Message: "大事记内容不能超过 2000 个字符"}
	}
	if rawLabel == nil {
		return text, nil, nil
	}
	label := strings.TrimSpace(*rawLabel)
	if utf8.RuneCountInString(label) > model.MaxChronicleTimeLabel {
		return "", nil, &model.ValidationError{Field: "time_label", Message: "时间标签不能超过 200 个字符"}
	}
	if label == "" {
		return text, nil, nil
	}
	return text, &label, nil
}

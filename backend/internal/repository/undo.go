package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"novel-management/backend/internal/model"
)

// Undo 管理一次性撤销令牌。
type Undo struct {
	DB *gorm.DB
}

func NewUndo(db *gorm.DB) *Undo { return &Undo{DB: db} }

func (r *Undo) Insert(ctx context.Context, token *model.UndoToken) error {
	return r.DB.WithContext(ctx).Create(token).Error
}

// Consume 原子占用令牌：已用 → ErrAlreadyDeleted；过期 → ErrUndoExpired；不存在 → ErrUndoExpired。
func (r *Undo) Consume(ctx context.Context, token string, now time.Time) (*model.UndoToken, error) {
	var record model.UndoToken
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token = ?", token).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrUndoExpired
		}
		if err != nil {
			return err
		}
		if record.UsedAt != nil {
			return model.ErrAlreadyDeleted
		}
		if now.After(record.ExpiresAt) {
			return model.ErrUndoExpired
		}
		return tx.Table("undo_tokens").Where("token = ?", token).Update("used_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteOlderThan 清理过期令牌，避免表无限增长（无独立定时任务）。
func (r *Undo) DeleteOlderThan(ctx context.Context, before time.Time) error {
	return r.DB.WithContext(ctx).Where("expires_at < ?", before).Delete(&model.UndoToken{}).Error
}

package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

// Settings 读写当前作品等全局设置。
type Settings struct {
	db *gorm.DB
}

func NewSettings(db *gorm.DB) *Settings { return &Settings{db: db} }

func (s *Settings) Get(ctx context.Context) (*model.AppSettings, error) {
	row, err := repository.NewSettings(s.db).Get(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.AppSettings{ID: 1, SnapshotKeep: model.SnapshotKeepDefault}, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// SetCurrentWork 校验作品存在后写入；空值表示清空当前作品。
func (s *Settings) SetCurrentWork(ctx context.Context, workID *string) error {
	repo := repository.NewSettings(s.db)
	now := time.Now().UTC()

	if workID == nil || strings.TrimSpace(*workID) == "" {
		return repo.SetCurrentWork(ctx, nil, now)
	}
	if _, err := repository.NewWorks(s.db).Get(ctx, *workID); err != nil {
		return err
	}
	return repo.SetCurrentWork(ctx, workID, now)
}

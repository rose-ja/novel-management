package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

// Trash 负责回收站列表、整组恢复、永久删除、清空与撤销。
type Trash struct {
	db        *gorm.DB
	snapshots *Snapshot
	now       func() time.Time
}

func NewTrash(db *gorm.DB, snapshots *Snapshot) *Trash {
	return &Trash{db: db, snapshots: snapshots, now: time.Now}
}

type TrashItem struct {
	RootType      string     `json:"root_type"`
	RootID        string     `json:"root_id"`
	Name          string     `json:"name"`
	ParentName    *string    `json:"parent_name"`
	DeletedAt     *time.Time `json:"deleted_at"`
	AffectedCount int64      `json:"affected_count"`
}

type TrashPage struct {
	Items    []TrashItem `json:"items"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Total    int64       `json:"total"`
}

type RenameItem struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	NewName    string `json:"new_name"`
}

type RestoreInput struct {
	NewName *string      `json:"new_name"`
	Renames []RenameItem `json:"renames"`
}

type RestoreResult struct {
	RootType      string `json:"root_type"`
	RootID        string `json:"root_id"`
	Name          string `json:"name"`
	RestoredCount int64  `json:"restored_count"`
}

type UndoResult struct {
	Restored bool   `json:"restored"`
	RootType string `json:"root_type"`
	RootID   string `json:"root_id"`
}

type PurgeAllResult struct {
	Purged     int64  `json:"purged"`
	SnapshotID string `json:"snapshot_id"`
}

// List 只返回回收站根记录；本变更仅 work 根有数据，其余类型返回空列表。
func (s *Trash) List(ctx context.Context, rootType string, page, pageSize int) (*TrashPage, error) {
	page, pageSize = normalizePage(page, pageSize)
	if rootType != "" && rootType != RootTypeWork {
		return &TrashPage{Items: []TrashItem{}, Page: page, PageSize: pageSize}, nil
	}

	works, err := repository.NewWorks(s.db).ListDeletedRoots(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := repository.NewWorks(s.db).CountDeletedByRoot(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]TrashItem, 0, len(works))
	for _, work := range works {
		items = append(items, TrashItem{
			RootType:      RootTypeWork,
			RootID:        work.ID,
			Name:          work.Name,
			DeletedAt:     work.DeletedAt,
			AffectedCount: counts[work.ID],
		})
	}

	total := int64(len(items))
	start := (page - 1) * pageSize
	if start >= len(items) {
		return &TrashPage{Items: []TrashItem{}, Page: page, PageSize: pageSize, Total: total}, nil
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return &TrashPage{Items: items[start:end], Page: page, PageSize: pageSize, Total: total}, nil
}

// Restore 整组原子恢复；同名冲突时不部分恢复。
func (s *Trash) Restore(ctx context.Context, rootType, rootID string, in RestoreInput) (*RestoreResult, error) {
	if rootType != RootTypeWork {
		return nil, model.ErrInvalidRootType
	}

	now := s.now().UTC()
	var result *RestoreResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewWorks(tx)
		work, err := repo.GetDeleted(ctx, rootID)
		if err != nil {
			return alreadyDeleted(err)
		}

		name, nameKey := work.Name, work.NameKey
		if in.NewName != nil {
			name, nameKey, err = model.NormalizeName(*in.NewName)
			if err != nil {
				return err
			}
		}

		taken, err := repo.NameTaken(ctx, nameKey)
		if err != nil {
			return err
		}
		if taken {
			return &model.RestoreConflictError{Items: []model.ConflictItem{{ID: rootID, Name: name}}}
		}

		if in.NewName != nil {
			if err := repo.RenameDeleted(ctx, rootID, name, nameKey, now); err != nil {
				return alreadyDeleted(err)
			}
		}

		children, err := repo.CountDeletedChildren(ctx, rootID)
		if err != nil {
			return err
		}
		if err := repo.RestoreGroup(ctx, rootID, now); err != nil {
			return alreadyDeleted(err)
		}

		restored, err := repo.Get(ctx, rootID)
		if err != nil {
			return err
		}
		result = &RestoreResult{
			RootType:      RootTypeWork,
			RootID:        rootID,
			Name:          restored.Name,
			RestoredCount: children + 1,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Purge 永久删除单个根；先落 trash_purge 快照，失败即阻止。
func (s *Trash) Purge(ctx context.Context, rootType, rootID string) error {
	if rootType != RootTypeWork {
		return model.ErrInvalidRootType
	}
	if _, err := repository.NewWorks(s.db).GetDeleted(ctx, rootID); err != nil {
		return alreadyDeleted(err)
	}
	if _, err := s.snapshots.Create(ctx, model.SnapshotReasonTrashPurge); err != nil {
		return err
	}
	rows, err := repository.NewSystem(s.db).ListReferencesByWork(ctx, rootID)
	if err != nil {
		return err
	}
	if err := workReferenceConflict(rows, "永久删除"); err != nil {
		return err
	}

	now := s.now().UTC()
	_, err = repository.NewWorks(s.db).PurgeGroup(ctx, rootID)
	if err != nil {
		return err
	}
	return repository.NewSettings(s.db).ClearCurrentWorkIf(ctx, rootID, now)
}

// PurgeAll 清空回收站；recycle bin 为空时不生成快照。
func (s *Trash) PurgeAll(ctx context.Context, confirm string) (*PurgeAllResult, error) {
	if confirm != "PURGE_ALL" {
		return nil, &model.ValidationError{Field: "confirm", Message: "清空回收站需要提供 confirm=PURGE_ALL"}
	}

	works, err := repository.NewWorks(s.db).ListDeletedRoots(ctx)
	if err != nil {
		return nil, err
	}
	if len(works) == 0 {
		return &PurgeAllResult{Purged: 0}, nil
	}

	system := repository.NewSystem(s.db)
	var references []repository.ReferenceRow
	for _, work := range works {
		rows, err := system.ListReferencesByWork(ctx, work.ID)
		if err != nil {
			return nil, err
		}
		references = append(references, rows...)
	}
	if err := workReferenceConflict(references, "清空回收站"); err != nil {
		return nil, err
	}

	snapshot, err := s.snapshots.Create(ctx, model.SnapshotReasonPurgeAll)
	if err != nil {
		return nil, err
	}

	purged, err := repository.NewWorks(s.db).PurgeAllDeleted(ctx)
	if err != nil {
		return nil, err
	}
	return &PurgeAllResult{Purged: purged, SnapshotID: snapshot.ID}, nil
}

// Undo 在 8 秒窗口内撤销最近的删除；令牌一次性，重复使用返回 ALREADY_DELETED。
func (s *Trash) Undo(ctx context.Context, token string) (*UndoResult, error) {
	now := s.now().UTC()
	record, err := repository.NewUndo(s.db).Consume(ctx, token, now)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if record.RootType != RootTypeWork {
			return model.ErrAlreadyDeleted
		}
		repo := repository.NewWorks(tx)
		if err := repo.RestoreGroup(ctx, record.RootID, now); err != nil {
			return alreadyDeleted(err)
		}
		// 删除时清空了当前作品，撤销后补回，避免恢复到「未选择作品」。
		settings := repository.NewSettings(tx)
		row, err := settings.Get(ctx)
		if err != nil {
			return err
		}
		if row.CurrentWorkID == nil {
			return settings.SetCurrentWork(ctx, &record.RootID, now)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	_ = repository.NewUndo(s.db).DeleteOlderThan(ctx, now.Add(-24*time.Hour))
	return &UndoResult{Restored: true, RootType: record.RootType, RootID: record.RootID}, nil
}

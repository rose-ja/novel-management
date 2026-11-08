package service

import (
	"context"
	"errors"
	"sort"
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

// List 汇总三类回收站根，按删除时间倒序后在内存分页。
func (s *Trash) List(ctx context.Context, rootType string, page, pageSize int) (*TrashPage, error) {
	page, pageSize = normalizePage(page, pageSize)
	switch rootType {
	case "", RootTypeWork, RootTypeSystemCategory, RootTypeSystemNode:
	default:
		return nil, model.ErrInvalidRootType
	}

	worksRepo := repository.NewWorks(s.db)
	systemRepo := repository.NewSystem(s.db)

	works := []model.Work{}
	categories := []model.SystemCategory{}
	nodes := []model.SystemNode{}
	var err error
	if rootType == "" || rootType == RootTypeWork {
		if works, err = worksRepo.ListDeletedRoots(ctx); err != nil {
			return nil, err
		}
	}
	if rootType == "" || rootType == RootTypeSystemCategory {
		if categories, err = systemRepo.ListDeletedCategoryRoots(ctx); err != nil {
			return nil, err
		}
	}
	if rootType == "" || rootType == RootTypeSystemNode {
		if nodes, err = systemRepo.ListDeletedNodeRoots(ctx); err != nil {
			return nil, err
		}
	}

	workCounts, err := worksRepo.CountDeletedByRoot(ctx)
	if err != nil {
		return nil, err
	}
	nodeCounts, err := systemRepo.CountDeletedNodesByCategoryRoot(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]TrashItem, 0, len(works)+len(categories)+len(nodes))
	for _, work := range works {
		items = append(items, TrashItem{
			RootType:      RootTypeWork,
			RootID:        work.ID,
			Name:          work.Name,
			DeletedAt:     work.DeletedAt,
			AffectedCount: workCounts[work.ID],
		})
	}
	for _, category := range categories {
		var parent *string
		if name, err := worksRepo.Name(ctx, category.WorkID); err == nil {
			parent = &name
		}
		items = append(items, TrashItem{
			RootType:      RootTypeSystemCategory,
			RootID:        category.ID,
			Name:          category.Name,
			ParentName:    parent,
			DeletedAt:     category.DeletedAt,
			AffectedCount: nodeCounts[category.ID],
		})
	}
	for _, node := range nodes {
		var parent *string
		if name, err := systemRepo.CategoryName(ctx, node.CategoryID); err == nil {
			parent = &name
		}
		items = append(items, TrashItem{
			RootType:   RootTypeSystemNode,
			RootID:     node.ID,
			Name:       node.Name,
			ParentName: parent,
			DeletedAt:  node.DeletedAt,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].DeletedAt.Equal(*items[j].DeletedAt) {
			return items[i].RootID > items[j].RootID
		}
		return items[i].DeletedAt.After(*items[j].DeletedAt)
	})

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

// Restore 按根类型整组原子恢复；同名冲突时不部分恢复。
func (s *Trash) Restore(ctx context.Context, rootType, rootID string, in RestoreInput) (*RestoreResult, error) {
	now := s.now().UTC()
	var result *RestoreResult
	var run func(tx *gorm.DB) error
	switch rootType {
	case RootTypeWork:
		run = func(tx *gorm.DB) error {
			var err error
			result, err = s.restoreWork(ctx, tx, rootID, in, now)
			return err
		}
	case RootTypeSystemCategory:
		run = func(tx *gorm.DB) error {
			var err error
			result, err = s.restoreCategory(ctx, tx, rootID, in, now)
			return err
		}
	case RootTypeSystemNode:
		run = func(tx *gorm.DB) error {
			var err error
			result, err = s.restoreNode(ctx, tx, rootID, in, now)
			return err
		}
	default:
		return nil, model.ErrInvalidRootType
	}

	if err := s.db.WithContext(ctx).Transaction(run); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Trash) restoreWork(ctx context.Context, tx *gorm.DB, rootID string, in RestoreInput, now time.Time) (*RestoreResult, error) {
	repo := repository.NewWorks(tx)
	work, err := repo.GetDeleted(ctx, rootID)
	if err != nil {
		return nil, alreadyDeleted(err)
	}

	name, nameKey := work.Name, work.NameKey
	if in.NewName != nil {
		name, nameKey, err = model.NormalizeName(*in.NewName)
		if err != nil {
			return nil, err
		}
	}

	taken, err := repo.NameTaken(ctx, nameKey)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, &model.RestoreConflictError{Items: []model.ConflictItem{{
			ID:         rootID,
			Name:       name,
			EntityType: RootTypeWork,
		}}}
	}

	if in.NewName != nil {
		if err := repo.RenameDeleted(ctx, rootID, name, nameKey, now); err != nil {
			return nil, alreadyDeleted(err)
		}
	}

	children, err := repo.CountDeletedChildren(ctx, rootID)
	if err != nil {
		return nil, err
	}
	if err := repo.RestoreGroup(ctx, rootID, now); err != nil {
		return nil, alreadyDeleted(err)
	}

	restored, err := repo.Get(ctx, rootID)
	if err != nil {
		return nil, err
	}
	return &RestoreResult{
		RootType:      RootTypeWork,
		RootID:        rootID,
		Name:          restored.Name,
		RestoredCount: children + 1,
	}, nil
}

func (s *Trash) restoreCategory(ctx context.Context, tx *gorm.DB, rootID string, in RestoreInput, now time.Time) (*RestoreResult, error) {
	system := repository.NewSystem(tx)
	category, err := system.GetDeletedCategoryRoot(ctx, rootID)
	if err != nil {
		return nil, alreadyDeleted(err)
	}
	if _, err := repository.NewWorks(tx).Name(ctx, category.WorkID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.ErrParentDeleted
		}
		return nil, err
	}

	name, nameKey := category.Name, category.NameKey
	if in.NewName != nil {
		name, nameKey, err = model.NormalizeKey(*in.NewName, "name")
		if err != nil {
			return nil, err
		}
	}

	var conflicts []model.ConflictItem
	taken, err := system.CategoryNameTaken(ctx, category.WorkID, nameKey, "")
	if err != nil {
		return nil, err
	}
	if taken {
		conflicts = append(conflicts, model.ConflictItem{
			ID:         category.ID,
			Name:       name,
			EntityType: RootTypeSystemCategory,
		})
	}

	nodes, err := system.ListDeletedNodesByCategoryRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	nodeByID := make(map[string]model.SystemNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	renameNames := make(map[string]string, len(in.Renames))
	renameKeys := make(map[string]string, len(in.Renames))
	for _, rename := range in.Renames {
		if rename.EntityType != RootTypeSystemNode {
			return nil, &model.ValidationError{Field: "renames", Message: "renames.entity_type 仅支持 system_node"}
		}
		node, ok := nodeByID[rename.EntityID]
		if !ok {
			return nil, &model.ValidationError{Field: "renames", Message: "renames 指向的节点不在该类别回收站组内"}
		}
		display, key, err := model.NormalizeKey(rename.NewName, "new_name")
		if err != nil {
			return nil, err
		}
		if _, exists := renameKeys[siblingGroupKey(node.ParentID)+key]; exists {
			conflicts = append(conflicts, model.ConflictItem{
				ID:         node.ID,
				Name:       display,
				EntityType: RootTypeSystemNode,
			})
		}
		renameKeys[siblingGroupKey(node.ParentID)+key] = node.ID
		taken, err := system.NodeNameTaken(ctx, category.ID, node.ParentID, key, node.ID)
		if err != nil {
			return nil, err
		}
		if taken {
			conflicts = append(conflicts, model.ConflictItem{
				ID:         node.ID,
				Name:       display,
				EntityType: RootTypeSystemNode,
			})
		}
		renameNames[node.ID] = display
	}
	if len(conflicts) > 0 {
		return nil, &model.RestoreConflictError{Items: conflicts}
	}

	if in.NewName != nil {
		if err := system.RenameDeletedCategory(ctx, rootID, name, nameKey, now); err != nil {
			return nil, alreadyDeleted(err)
		}
	}
	for nodeID, display := range renameNames {
		_, key, err := model.NormalizeKey(display, "name")
		if err != nil {
			return nil, err
		}
		if err := system.RenameDeletedNodeInGroup(ctx, nodeID, rootID, display, key, now); err != nil {
			return nil, alreadyDeleted(err)
		}
	}
	if err := system.RestoreCategoryGroup(ctx, rootID, now); err != nil {
		return nil, alreadyDeleted(err)
	}

	return &RestoreResult{
		RootType:      RootTypeSystemCategory,
		RootID:        rootID,
		Name:          name,
		RestoredCount: int64(len(nodes)) + 1,
	}, nil
}

func (s *Trash) restoreNode(ctx context.Context, tx *gorm.DB, rootID string, in RestoreInput, now time.Time) (*RestoreResult, error) {
	system := repository.NewSystem(tx)
	node, err := system.GetDeletedNodeRoot(ctx, rootID)
	if err != nil {
		return nil, alreadyDeleted(err)
	}
	if _, err := system.GetCategory(ctx, node.CategoryID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.ErrParentDeleted
		}
		return nil, err
	}
	if node.ParentID != nil {
		if _, err := system.GetNode(ctx, *node.ParentID); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return nil, model.ErrParentDeleted
			}
			return nil, err
		}
	}

	name, nameKey := node.Name, node.NameKey
	if in.NewName != nil {
		name, nameKey, err = model.NormalizeKey(*in.NewName, "name")
		if err != nil {
			return nil, err
		}
	} else {
		for _, rename := range in.Renames {
			if rename.EntityID != rootID {
				continue
			}
			if rename.EntityType != RootTypeSystemNode {
				return nil, &model.ValidationError{Field: "renames", Message: "renames.entity_type 仅支持 system_node"}
			}
			name, nameKey, err = model.NormalizeKey(rename.NewName, "new_name")
			if err != nil {
				return nil, err
			}
		}
	}

	taken, err := system.NodeNameTaken(ctx, node.CategoryID, node.ParentID, nameKey, node.ID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, &model.RestoreConflictError{Items: []model.ConflictItem{{
			ID:         node.ID,
			Name:       name,
			EntityType: RootTypeSystemNode,
		}}}
	}

	if name != node.Name {
		if err := system.RenameDeletedNode(ctx, node.ID, name, nameKey, now); err != nil {
			return nil, alreadyDeleted(err)
		}
	}
	if err := system.RestoreNode(ctx, node.ID, now); err != nil {
		return nil, alreadyDeleted(err)
	}
	return &RestoreResult{
		RootType:      RootTypeSystemNode,
		RootID:        node.ID,
		Name:          name,
		RestoredCount: 1,
	}, nil
}

// Purge 永久删除单个根；先落 trash_purge 快照，失败即阻止。
func (s *Trash) Purge(ctx context.Context, rootType, rootID string) error {
	system := repository.NewSystem(s.db)
	switch rootType {
	case RootTypeWork:
		if _, err := repository.NewWorks(s.db).GetDeleted(ctx, rootID); err != nil {
			return alreadyDeleted(err)
		}
		if _, err := s.snapshots.Create(ctx, model.SnapshotReasonTrashPurge); err != nil {
			return err
		}
		rows, err := system.ListReferencesByWork(ctx, rootID)
		if err != nil {
			return err
		}
		if err := referenceConflict(rows, "作品", "永久删除"); err != nil {
			return err
		}
		now := s.now().UTC()
		if _, err := repository.NewWorks(s.db).PurgeGroup(ctx, rootID); err != nil {
			return err
		}
		return repository.NewSettings(s.db).ClearCurrentWorkIf(ctx, rootID, now)
	case RootTypeSystemCategory:
		if _, err := system.GetDeletedCategoryRoot(ctx, rootID); err != nil {
			return alreadyDeleted(err)
		}
		if _, err := s.snapshots.Create(ctx, model.SnapshotReasonTrashPurge); err != nil {
			return err
		}
		rows, err := system.ListReferencesByCategory(ctx, rootID)
		if err != nil {
			return err
		}
		if err := referenceConflict(rows, "类别", "永久删除"); err != nil {
			return err
		}
		_, err = system.PurgeCategoryGroup(ctx, rootID)
		return err
	case RootTypeSystemNode:
		if _, err := system.GetDeletedNodeRoot(ctx, rootID); err != nil {
			return alreadyDeleted(err)
		}
		if _, err := s.snapshots.Create(ctx, model.SnapshotReasonTrashPurge); err != nil {
			return err
		}
		rows, err := system.ListReferencesByNode(ctx, rootID)
		if err != nil {
			return err
		}
		if err := referenceConflict(rows, "节点", "永久删除"); err != nil {
			return err
		}
		_, err = system.PurgeNode(ctx, rootID)
		return err
	default:
		return model.ErrInvalidRootType
	}
}

// PurgeAll 清空回收站；先做全量引用检查，再落快照并按顺序物理删除。
func (s *Trash) PurgeAll(ctx context.Context, confirm string) (*PurgeAllResult, error) {
	if confirm != "PURGE_ALL" {
		return nil, &model.ValidationError{Field: "confirm", Message: "清空回收站需要提供 confirm=PURGE_ALL"}
	}

	works, err := repository.NewWorks(s.db).ListDeletedRoots(ctx)
	if err != nil {
		return nil, err
	}
	system := repository.NewSystem(s.db)
	categories, err := system.ListDeletedCategoryRoots(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := system.ListDeletedNodeRoots(ctx)
	if err != nil {
		return nil, err
	}
	if len(works)+len(categories)+len(nodes) == 0 {
		return &PurgeAllResult{Purged: 0}, nil
	}

	var references []repository.ReferenceRow
	for _, work := range works {
		rows, err := system.ListReferencesByWork(ctx, work.ID)
		if err != nil {
			return nil, err
		}
		references = append(references, rows...)
	}
	for _, category := range categories {
		rows, err := system.ListReferencesByCategory(ctx, category.ID)
		if err != nil {
			return nil, err
		}
		references = append(references, rows...)
	}
	for _, node := range nodes {
		rows, err := system.ListReferencesByNode(ctx, node.ID)
		if err != nil {
			return nil, err
		}
		references = append(references, rows...)
	}
	if err := referenceConflict(references, "回收站", "清空"); err != nil {
		return nil, err
	}

	snapshot, err := s.snapshots.Create(ctx, model.SnapshotReasonPurgeAll)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, work := range works {
			if _, err := repository.NewWorks(tx).PurgeGroup(ctx, work.ID); err != nil {
				return err
			}
		}
		for _, category := range categories {
			if _, err := repository.NewSystem(tx).PurgeCategoryGroup(ctx, category.ID); err != nil {
				return err
			}
		}
		for _, node := range nodes {
			if _, err := repository.NewSystem(tx).PurgeNode(ctx, node.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &PurgeAllResult{
		Purged:     int64(len(works) + len(categories) + len(nodes)),
		SnapshotID: snapshot.ID,
	}, nil
}

// Undo 在 8 秒窗口内撤销最近的删除；令牌一次性，重复使用返回 ALREADY_DELETED。
func (s *Trash) Undo(ctx context.Context, token string) (*UndoResult, error) {
	now := s.now().UTC()
	record, err := repository.NewUndo(s.db).Consume(ctx, token, now)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		switch record.RootType {
		case RootTypeWork:
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
		case RootTypeSystemCategory:
			system := repository.NewSystem(tx)
			category, err := system.GetDeletedCategoryRoot(ctx, record.RootID)
			if err != nil {
				return alreadyDeleted(err)
			}
			if _, err := repository.NewWorks(tx).Name(ctx, category.WorkID); err != nil {
				if errors.Is(err, model.ErrNotFound) {
					return model.ErrParentDeleted
				}
				return err
			}
			taken, err := system.CategoryNameTaken(ctx, category.WorkID, category.NameKey, "")
			if err != nil {
				return err
			}
			if taken {
				return &model.RestoreConflictError{Items: []model.ConflictItem{{
					ID:         category.ID,
					Name:       category.Name,
					EntityType: RootTypeSystemCategory,
				}}}
			}
			if err := system.RestoreCategoryGroup(ctx, record.RootID, now); err != nil {
				return alreadyDeleted(err)
			}
			return nil
		case RootTypeSystemNode:
			system := repository.NewSystem(tx)
			node, err := system.GetDeletedNodeRoot(ctx, record.RootID)
			if err != nil {
				return alreadyDeleted(err)
			}
			if _, err := system.GetCategory(ctx, node.CategoryID); err != nil {
				if errors.Is(err, model.ErrNotFound) {
					return model.ErrParentDeleted
				}
				return err
			}
			if node.ParentID != nil {
				if _, err := system.GetNode(ctx, *node.ParentID); err != nil {
					if errors.Is(err, model.ErrNotFound) {
						return model.ErrParentDeleted
					}
					return err
				}
			}
			taken, err := system.NodeNameTaken(ctx, node.CategoryID, node.ParentID, node.NameKey, node.ID)
			if err != nil {
				return err
			}
			if taken {
				return &model.RestoreConflictError{Items: []model.ConflictItem{{
					ID:         node.ID,
					Name:       node.Name,
					EntityType: RootTypeSystemNode,
				}}}
			}
			if err := system.RestoreNode(ctx, record.RootID, now); err != nil {
				return alreadyDeleted(err)
			}
			return nil
		default:
			return model.ErrAlreadyDeleted
		}
	})
	if err != nil {
		return nil, err
	}

	_ = repository.NewUndo(s.db).DeleteOlderThan(ctx, now.Add(-24*time.Hour))
	return &UndoResult{Restored: true, RootType: record.RootType, RootID: record.RootID}, nil
}

func siblingGroupKey(parentID *string) string {
	if parentID == nil {
		return "root:"
	}
	return "node:" + *parentID + ":"
}

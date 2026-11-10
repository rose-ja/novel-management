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

// List 汇总四类回收站根，按删除时间倒序后在内存分页。
func (s *Trash) List(ctx context.Context, rootType string, page, pageSize int) (*TrashPage, error) {
	page, pageSize = normalizePage(page, pageSize)
	switch rootType {
	case "", RootTypeWork, RootTypeSystemCategory, RootTypeSystemNode, RootTypeCharacter:
	default:
		return nil, model.ErrInvalidRootType
	}

	worksRepo := repository.NewWorks(s.db)
	systemRepo := repository.NewSystem(s.db)

	works := []model.Work{}
	categories := []model.SystemCategory{}
	nodes := []model.SystemNode{}
	characters := []model.Character{}
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
	if rootType == "" || rootType == RootTypeCharacter {
		if characters, err = repository.NewCharacters(s.db).ListDeletedRoots(ctx); err != nil {
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

	items := make([]TrashItem, 0, len(works)+len(categories)+len(nodes)+len(characters))
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
	for _, character := range characters {
		var parent *string
		if name, err := worksRepo.Name(ctx, character.WorkID); err == nil {
			parent = &name
		}
		items = append(items, TrashItem{
			RootType:      RootTypeCharacter,
			RootID:        character.ID,
			Name:          character.Name,
			ParentName:    parent,
			DeletedAt:     character.DeletedAt,
			AffectedCount: 1,
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
	case RootTypeCharacter:
		run = func(tx *gorm.DB) error {
			var err error
			result, err = s.restoreCharacter(ctx, tx, rootID, in, now)
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
	if _, err := repository.NewWorks(tx).ActiveName(ctx, category.WorkID); err != nil {
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
	conflictSeen := make(map[string]struct{})
	addConflict := func(item model.ConflictItem) {
		if _, exists := conflictSeen[item.ID]; exists {
			return
		}
		conflictSeen[item.ID] = struct{}{}
		conflicts = append(conflicts, item)
	}
	taken, err := system.CategoryNameTaken(ctx, category.WorkID, nameKey, "")
	if err != nil {
		return nil, err
	}
	if taken {
		addConflict(model.ConflictItem{
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
		if _, ok := nodeByID[rename.EntityID]; !ok {
			return nil, &model.ValidationError{Field: "renames", Message: "renames 指向的节点不在该类别回收站组内"}
		}
		display, key, err := model.NormalizeKey(rename.NewName, "new_name")
		if err != nil {
			return nil, err
		}
		renameNames[rename.EntityID] = display
		renameKeys[rename.EntityID] = key
	}

	// 计算每个节点的最终名称：改名节点用 renames，未改名节点沿用原名；
	// 按父级分组检测最终名称重复，覆盖「改名撞同组未改名兄弟」。
	finalDisplay := make(map[string]string, len(nodes))
	finalKey := make(map[string]string, len(nodes))
	for _, node := range nodes {
		if display, ok := renameNames[node.ID]; ok {
			finalDisplay[node.ID] = display
			finalKey[node.ID] = renameKeys[node.ID]
			continue
		}
		finalDisplay[node.ID] = node.Name
		finalKey[node.ID] = node.NameKey
	}
	seenGroup := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		groupKey := siblingGroupKey(node.ParentID) + finalKey[node.ID]
		if _, exists := seenGroup[groupKey]; exists {
			addConflict(model.ConflictItem{
				ID:         node.ID,
				Name:       finalDisplay[node.ID],
				EntityType: RootTypeSystemNode,
			})
			continue
		}
		seenGroup[groupKey] = struct{}{}
	}

	// 最终名称还需与同类别活跃节点比较（改名与未改名节点都覆盖）。
	for _, node := range nodes {
		taken, err := system.NodeNameTaken(ctx, category.ID, node.ParentID, finalKey[node.ID], node.ID)
		if err != nil {
			return nil, err
		}
		if taken {
			addConflict(model.ConflictItem{
				ID:         node.ID,
				Name:       finalDisplay[node.ID],
				EntityType: RootTypeSystemNode,
			})
		}
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
		if err := system.RenameDeletedNodeInGroup(ctx, nodeID, rootID, display, renameKeys[nodeID], now); err != nil {
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

// restoreCharacter 恢复人物根；作品仍在回收站时阻断，同名时支持 new_name。
func (s *Trash) restoreCharacter(ctx context.Context, tx *gorm.DB, rootID string, in RestoreInput, now time.Time) (*RestoreResult, error) {
	repo := repository.NewCharacters(tx)
	item, err := repo.GetDeleted(ctx, rootID)
	if err != nil {
		return nil, alreadyDeleted(err)
	}
	if _, err := repository.NewWorks(tx).GetForUpdate(ctx, item.WorkID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.ErrParentDeleted
		}
		return nil, err
	}

	name, nameKey := item.Name, item.NameKey
	if in.NewName != nil {
		name, nameKey, err = model.NormalizeCharacterName(*in.NewName)
		if err != nil {
			return nil, err
		}
	}

	taken, err := repo.NameTaken(ctx, item.WorkID, nameKey, item.ID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, &model.RestoreConflictError{Items: []model.ConflictItem{{
			ID:         item.ID,
			Name:       name,
			EntityType: RootTypeCharacter,
		}}}
	}

	if name != item.Name || nameKey != item.NameKey {
		if err := repo.RenameDeleted(ctx, item.ID, name, nameKey, now); err != nil {
			return nil, alreadyDeleted(err)
		}
	}
	if err := repo.Restore(ctx, item.ID, now); err != nil {
		return nil, alreadyDeleted(err)
	}
	return &RestoreResult{
		RootType:      RootTypeCharacter,
		RootID:        item.ID,
		Name:          name,
		RestoredCount: 1,
	}, nil
}

// Purge 永久删除单个根；先落 trash_purge 快照，失败即阻止；
// 引用检查与物理删除共用同一事务，避免 TOCTOU。
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
		now := s.now().UTC()
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// 作品内人物引用指向本作品的类别/节点，属于随组删除的子记录，不构成入向阻断。
			if _, err := repository.NewWorks(tx).PurgeGroup(ctx, rootID); err != nil {
				return err
			}
			return repository.NewSettings(tx).ClearCurrentWorkIf(ctx, rootID, now)
		})
	case RootTypeSystemCategory:
		if _, err := system.GetDeletedCategoryRoot(ctx, rootID); err != nil {
			return alreadyDeleted(err)
		}
		if _, err := s.snapshots.Create(ctx, model.SnapshotReasonTrashPurge); err != nil {
			return err
		}
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			repo := repository.NewSystem(tx)
			rows, err := repo.ListReferencesByCategory(ctx, rootID)
			if err != nil {
				return err
			}
			if err := referenceConflict(rows, "类别", "永久删除"); err != nil {
				return err
			}
			_, err = repo.PurgeCategoryGroup(ctx, rootID)
			return err
		})
	case RootTypeSystemNode:
		if _, err := system.GetDeletedNodeRoot(ctx, rootID); err != nil {
			return alreadyDeleted(err)
		}
		children, err := system.ListNodeChildren(ctx, rootID)
		if err != nil {
			return err
		}
		if len(children) > 0 {
			return &model.HasChildrenError{Items: childConflictItems(children)}
		}
		if _, err := s.snapshots.Create(ctx, model.SnapshotReasonTrashPurge); err != nil {
			return err
		}
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			repo := repository.NewSystem(tx)
			rows, err := repo.ListReferencesByNode(ctx, rootID)
			if err != nil {
				return err
			}
			if err := referenceConflict(rows, "节点", "永久删除"); err != nil {
				return err
			}
			_, err = repo.PurgeNode(ctx, rootID)
			return err
		})
	case RootTypeCharacter:
		if _, err := repository.NewCharacters(s.db).GetDeleted(ctx, rootID); err != nil {
			return alreadyDeleted(err)
		}
		if _, err := s.snapshots.Create(ctx, model.SnapshotReasonTrashPurge); err != nil {
			return err
		}
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			rows, err := repository.NewCharacters(tx).Purge(ctx, rootID)
			if err != nil {
				return err
			}
			if rows == 0 {
				return model.ErrAlreadyDeleted
			}
			return nil
		})
	default:
		return model.ErrInvalidRootType
	}
}

// PurgeAll 清空回收站；先做全量引用检查，再落快照，最后在事务内复核并物理删除。
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
	characters, err := repository.NewCharacters(s.db).ListDeletedRoots(ctx)
	if err != nil {
		return nil, err
	}
	if len(works)+len(categories)+len(nodes)+len(characters) == 0 {
		return &PurgeAllResult{Purged: 0}, nil
	}
	nodes = orderNodeRootsForPurge(nodes)
	total := int64(len(works) + len(categories) + len(nodes) + len(characters))

	// 先检查后快照：引用命中即中止，不落快照。
	references, err := s.collectReferences(ctx, system, categories, nodes)
	if err != nil {
		return nil, err
	}
	if err := referenceConflict(references, "回收站", "清空"); err != nil {
		return nil, err
	}

	snapshot, err := s.snapshots.Create(ctx, model.SnapshotReasonPurgeAll)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txSystem := repository.NewSystem(tx)
		// 事务内复核引用，确保引用检查与物理删除共用同一快照点。
		references, err := s.collectReferences(ctx, txSystem, categories, nodes)
		if err != nil {
			return err
		}
		if err := referenceConflict(references, "回收站", "清空"); err != nil {
			return err
		}
		txWorks := repository.NewWorks(tx)
		for _, work := range works {
			if _, err := txWorks.PurgeGroup(ctx, work.ID); err != nil {
				return err
			}
		}
		for _, category := range categories {
			if _, err := txSystem.PurgeCategoryGroup(ctx, category.ID); err != nil {
				return err
			}
		}
		for _, node := range nodes {
			if _, err := txSystem.PurgeNode(ctx, node.ID); err != nil {
				return err
			}
		}
		// 作品组删除已清理随作品删除的人物；此处只处理独立人物根。
		txCharacters := repository.NewCharacters(tx)
		for _, character := range characters {
			if _, err := txCharacters.Purge(ctx, character.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &PurgeAllResult{
		Purged:     total,
		SnapshotID: snapshot.ID,
	}, nil
}

// collectReferences 汇总类别/节点的引用明细；作品内引用随作品整组删除，不计入阻断。
func (s *Trash) collectReferences(ctx context.Context, system *repository.System, categories []model.SystemCategory, nodes []model.SystemNode) ([]repository.ReferenceRow, error) {
	var references []repository.ReferenceRow
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
	return references, nil
}

// orderNodeRootsForPurge 保证子节点根先于父节点根删除，避免自引用 FK RESTRICT。
func orderNodeRootsForPurge(nodes []model.SystemNode) []model.SystemNode {
	if len(nodes) < 2 {
		return nodes
	}
	rootIDs := make(map[string]struct{}, len(nodes))
	byID := make(map[string]model.SystemNode, len(nodes))
	children := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		rootIDs[node.ID] = struct{}{}
		byID[node.ID] = node
	}
	for _, node := range nodes {
		if node.ParentID == nil {
			continue
		}
		if _, ok := rootIDs[*node.ParentID]; ok {
			children[*node.ParentID] = append(children[*node.ParentID], node.ID)
		}
	}
	visited := make(map[string]bool, len(nodes))
	ordered := make([]model.SystemNode, 0, len(nodes))
	var visit func(id string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, childID := range children[id] {
			visit(childID)
		}
		ordered = append(ordered, byID[id])
	}
	for _, node := range nodes {
		visit(node.ID)
	}
	return ordered
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
			if _, err := repository.NewWorks(tx).ActiveName(ctx, category.WorkID); err != nil {
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
		case RootTypeCharacter:
			repo := repository.NewCharacters(tx)
			character, err := repo.GetDeleted(ctx, record.RootID)
			if err != nil {
				return alreadyDeleted(err)
			}
			if _, err := repository.NewWorks(tx).Get(ctx, character.WorkID); err != nil {
				if errors.Is(err, model.ErrNotFound) {
					return model.ErrParentDeleted
				}
				return err
			}
			taken, err := repo.NameTaken(ctx, character.WorkID, character.NameKey, character.ID)
			if err != nil {
				return err
			}
			if taken {
				return &model.RestoreConflictError{Items: []model.ConflictItem{{
					ID:         character.ID,
					Name:       character.Name,
					EntityType: RootTypeCharacter,
				}}}
			}
			if err := repo.Restore(ctx, record.RootID, now); err != nil {
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

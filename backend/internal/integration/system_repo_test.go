//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"

	"gorm.io/gorm"
)

func systemRepo(h *harness) *repository.System { return repository.NewSystem(h.tx) }

func seedCategory(t *testing.T, h *harness, repo *repository.System, workID, name, categoryType string) model.SystemCategory {
	t.Helper()
	ctx := context.Background()
	display, key, err := model.NormalizeKey(name, "name")
	if err != nil {
		t.Fatalf("生成类别名称失败：%v", err)
	}
	sortOrder, err := repo.NextCategorySortOrder(ctx, workID)
	if err != nil {
		t.Fatalf("读取类别排序失败：%v", err)
	}
	item := model.SystemCategory{
		ID:           model.NewID(),
		WorkID:       workID,
		Name:         display,
		NameKey:      key,
		CategoryType: categoryType,
		IsBuiltin:    false,
		IsEnabled:    true,
		SortOrder:    sortOrder,
		Version:      1,
	}
	if err := repo.InsertCategory(ctx, &item); err != nil {
		t.Fatalf("插入类别失败：%v", err)
	}
	return item
}

func seedNode(t *testing.T, h *harness, repo *repository.System, workID, categoryID string, parentID *string, name, review string) model.SystemNode {
	t.Helper()
	ctx := context.Background()
	display, key, err := model.NormalizeKey(name, "name")
	if err != nil {
		t.Fatalf("生成节点名称失败：%v", err)
	}
	sortOrder, err := repo.NextNodeSortOrder(ctx, categoryID, parentID)
	if err != nil {
		t.Fatalf("读取节点排序失败：%v", err)
	}
	item := model.SystemNode{
		ID:           model.NewID(),
		WorkID:       workID,
		CategoryID:   categoryID,
		ParentID:     parentID,
		Name:         display,
		NameKey:      key,
		SortOrder:    sortOrder,
		ReviewStatus: review,
		Version:      1,
	}
	if err := repo.InsertNode(ctx, &item); err != nil {
		t.Fatalf("插入节点失败：%v", err)
	}
	return item
}

func seedReference(t *testing.T, h *harness, workID, characterID, categoryID, nodeID string) {
	t.Helper()
	err := h.tx.Table("character_system_refs").Create(map[string]any{
		"id":           model.NewID(),
		"work_id":      workID,
		"character_id": characterID,
		"category_id":  categoryID,
		"node_id":      nodeID,
	}).Error
	if err != nil {
		t.Fatalf("插入人物引用失败：%v", err)
	}
}

func TestSystemRepoCategories(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	work := h.createWork("仓储类别")
	repo := systemRepo(h)

	items, err := repo.ListCategories(ctx, work.ID, true)
	if err != nil {
		t.Fatalf("列出类别失败：%v", err)
	}
	if len(items) != 1 || !items[0].IsBuiltin || items[0].Name != model.BuiltinRealmCategory {
		t.Fatalf("应只有内置境界，实际 %+v", items)
	}
	if items[0].NodeCount != 0 || items[0].ReferenceCount != 0 {
		t.Fatalf("内置类别计数应为 0，实际 %+v", items[0])
	}

	era := seedCategory(t, h, repo, work.ID, "纪元", model.CategoryTypeEra)
	custom := seedCategory(t, h, repo, work.ID, "自定义", model.CategoryTypeNormal)

	if count, err := repo.CountActiveCategories(ctx, work.ID); err != nil || count != 3 {
		t.Fatalf("类别数量应为 3，实际 %d（err=%v）", count, err)
	}
	if next, err := repo.NextCategorySortOrder(ctx, work.ID); err != nil || next != 3 {
		t.Fatalf("下一个排序应为 3，实际 %d（err=%v）", next, err)
	}

	if taken, err := repo.CategoryNameTaken(ctx, work.ID, "纪元", ""); err != nil || !taken {
		t.Fatalf("同名未删除类别应被判定占用，taken=%v err=%v", taken, err)
	}
	if taken, err := repo.CategoryNameTaken(ctx, work.ID, "纪元", era.ID); err != nil || taken {
		t.Fatalf("排除自身后不应占用，taken=%v err=%v", taken, err)
	}

	// 作品隔离：另一个作品的类别不进入列表。
	other := h.createWork("仓储类别第二部")
	if items, err := repo.ListCategories(ctx, other.ID, true); err != nil || len(items) != 1 {
		t.Fatalf("作品隔离失败：数量 %d err=%v", len(items), err)
	}

	rows, err := repo.UpdateCategory(ctx, custom.ID, custom.Name, custom.NameKey, false, custom.Version, time.Now())
	if err != nil || rows != 1 {
		t.Fatalf("停用类别应更新 1 行，实际 %d err=%v", rows, err)
	}
	rows, err = repo.UpdateCategory(ctx, custom.ID, custom.Name, custom.NameKey, false, custom.Version, time.Now())
	if err != nil || rows != 0 {
		t.Fatalf("过期版本应更新 0 行，实际 %d err=%v", rows, err)
	}

	items, err = repo.ListCategories(ctx, work.ID, false)
	if err != nil {
		t.Fatalf("列出启用类别失败：%v", err)
	}
	if len(items) != 2 {
		t.Fatalf("include_disabled=false 应返回 2 条，实际 %d", len(items))
	}
	for _, item := range items {
		if item.ID == custom.ID {
			t.Fatal("停用类别不应出现在启用列表")
		}
	}

	display, key, err := model.NormalizeKey("  新自定义  ", "name")
	if err != nil {
		t.Fatalf("规范化失败：%v", err)
	}
	rows, err = repo.UpdateCategory(ctx, custom.ID, display, key, false, 2, time.Now())
	if err != nil || rows != 1 {
		t.Fatalf("改名应更新 1 行，实际 %d err=%v", rows, err)
	}
	got, err := repo.GetCategory(ctx, custom.ID)
	if err != nil {
		t.Fatalf("读取类别失败：%v", err)
	}
	if got.Name != display || got.Version != 3 || got.IsEnabled {
		t.Fatalf("改名结果不符：%+v", got)
	}
}

func TestSystemRepoCategoryDeletionGroup(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	work := h.createWork("仓储类别删除")
	repo := systemRepo(h)
	category := seedCategory(t, h, repo, work.ID, "整组删除", model.CategoryTypeNormal)

	root := seedNode(t, h, repo, work.ID, category.ID, nil, "主层级", model.ReviewStatusReviewed)
	child := seedNode(t, h, repo, work.ID, category.ID, &root.ID, "子层级", model.ReviewStatusPending)

	if total, err := repo.CountNodeChildren(ctx, root.ID); err != nil || total != 1 {
		t.Fatalf("子节点数量应为 1，实际 %d err=%v", total, err)
	}

	now := time.Now()
	affected, err := repo.SoftDeleteCategoryGroup(ctx, category.ID, category.Version, "manual_delete", now)
	if err != nil || affected != 2 {
		t.Fatalf("整组软删除应影响 2 个节点，实际 %d err=%v", affected, err)
	}

	if _, err := repo.GetCategory(ctx, category.ID); err != model.ErrNotFound {
		t.Fatalf("删除后 GetCategory 应返回 ErrNotFound，实际 %v", err)
	}
	deleted, err := repo.GetDeletedCategoryRoot(ctx, category.ID)
	if err != nil || deleted.DeletedRootID == nil || *deleted.DeletedRootID != category.ID {
		t.Fatalf("回收站根不符：%+v err=%v", deleted, err)
	}

	counts, err := repo.CountDeletedNodesByCategoryRoot(ctx)
	if err != nil || counts[category.ID] != 2 {
		t.Fatalf("回收站节点统计应为 2，实际 %v err=%v", counts, err)
	}
	if nodes, err := repo.ListNodes(ctx, category.ID, ""); err != nil || len(nodes) != 0 {
		t.Fatalf("删除后不应再列出节点：%d err=%v", len(nodes), err)
	}
	if roots, err := repo.ListDeletedCategoryRoots(ctx); err != nil || len(roots) == 0 {
		t.Fatalf("回收站类别根缺失：%v err=%v", roots, err)
	}

	if err := repo.RestoreCategoryGroup(ctx, category.ID, time.Now()); err != nil {
		t.Fatalf("整组恢复失败：%v", err)
	}
	nodes, err := repo.ListNodes(ctx, category.ID, "")
	if err != nil || len(nodes) != 2 {
		t.Fatalf("恢复后应有 2 个节点，实际 %d err=%v", len(nodes), err)
	}
	if nodes[0].ID != root.ID || nodes[1].ID != child.ID {
		t.Fatalf("恢复后顺序不符：%+v", nodes)
	}

	current, err := repo.GetCategory(ctx, category.ID)
	if err != nil {
		t.Fatalf("恢复后读取类别失败：%v", err)
	}
	if _, err := repo.SoftDeleteCategoryGroup(ctx, category.ID, current.Version, "manual_delete", time.Now()); err != nil {
		t.Fatalf("二次软删除失败：%v", err)
	}
	if purged, err := repo.PurgeCategoryGroup(ctx, category.ID); err != nil || purged != 1 {
		t.Fatalf("永久删除应删 1 个类别，实际 %d err=%v", purged, err)
	}
	var remaining int64
	if err := h.tx.Table("system_nodes").Where("category_id = ?", category.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("统计残留节点失败：%v", err)
	}
	if remaining != 0 {
		t.Fatalf("永久删除后节点应清空，实际 %d", remaining)
	}
}

func TestSystemRepoNodeUniqueness(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("仓储节点唯一")
	repo := systemRepo(h)
	category := seedCategory(t, h, repo, work.ID, "同级唯一", model.CategoryTypeNormal)

	seedNode(t, h, repo, work.ID, category.ID, nil, "Ｓｗｏｒｄ", model.ReviewStatusReviewed)

	display, key, err := model.NormalizeKey("  sword  ", "name")
	if err != nil {
		t.Fatalf("规范化失败：%v", err)
	}
	dup := model.SystemNode{
		ID:           model.NewID(),
		WorkID:       work.ID,
		CategoryID:   category.ID,
		Name:         display,
		NameKey:      key,
		ReviewStatus: model.ReviewStatusReviewed,
		Version:      1,
	}
	err = h.tx.Transaction(func(tx *gorm.DB) error {
		return repository.NewSystem(tx).InsertNode(context.Background(), &dup)
	})
	if !repository.IsUniqueViolation(err, "system_nodes_sibling_name_uq") {
		t.Fatalf("同类同级同名应触发唯一索引，实际 %v", err)
	}

	root, err := repo.ListNodes(context.Background(), category.ID, "")
	if err != nil || len(root) != 1 {
		t.Fatalf("唯一索引拦截后应只剩 1 个节点：%d err=%v", len(root), err)
	}

	// 不同父级允许同名。
	seedNode(t, h, repo, work.ID, category.ID, &root[0].ID, "sword", model.ReviewStatusReviewed)
}

func TestSystemRepoNodeListReviewAndDisabled(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	work := h.createWork("仓储节点列表")
	repo := systemRepo(h)
	category := seedCategory(t, h, repo, work.ID, "列表筛选", model.CategoryTypeNormal)

	root := seedNode(t, h, repo, work.ID, category.ID, nil, "已校对", model.ReviewStatusReviewed)
	seedNode(t, h, repo, work.ID, category.ID, &root.ID, "待校对", model.ReviewStatusPending)
	seedNode(t, h, repo, work.ID, category.ID, nil, "根待校对", model.ReviewStatusPending)

	if rows, err := repo.UpdateCategory(ctx, category.ID, category.Name, category.NameKey, false, category.Version, time.Now()); err != nil || rows != 1 {
		t.Fatalf("停用类别失败：%d err=%v", rows, err)
	}

	nodes, err := repo.ListNodes(ctx, category.ID, "")
	if err != nil || len(nodes) != 3 {
		t.Fatalf("停用类别仍应列出全部节点，实际 %d err=%v", len(nodes), err)
	}
	pending, err := repo.ListNodes(ctx, category.ID, model.ReviewStatusPending)
	if err != nil || len(pending) != 2 {
		t.Fatalf("待校对筛选应返回 2 条，实际 %d err=%v", len(pending), err)
	}
	for _, node := range pending {
		if node.ReviewStatus != model.ReviewStatusPending {
			t.Fatalf("筛选结果包含已校对节点：%+v", node)
		}
	}
	if nodes[0].ID != root.ID {
		t.Fatalf("根节点应按 sort_order 排在最前：%+v", nodes)
	}
}

func TestSystemRepoReferenceCounts(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	work := h.createWork("仓储引用")
	repo := systemRepo(h)
	category := seedCategory(t, h, repo, work.ID, "引用计数", model.CategoryTypeNormal)
	other := seedCategory(t, h, repo, work.ID, "另一类别", model.CategoryTypeNormal)

	nodeA := seedNode(t, h, repo, work.ID, category.ID, nil, "剑", model.ReviewStatusReviewed)
	nodeB := seedNode(t, h, repo, work.ID, category.ID, nil, "刀", model.ReviewStatusReviewed)
	nodeC := seedNode(t, h, repo, work.ID, other.ID, nil, "境界一", model.ReviewStatusReviewed)

	seedReference(t, h, work.ID, "char-a", category.ID, nodeA.ID)
	seedReference(t, h, work.ID, "char-b", category.ID, nodeA.ID)
	seedReference(t, h, work.ID, "char-c", other.ID, nodeC.ID)

	items, err := repo.ListCategories(ctx, work.ID, true)
	if err != nil {
		t.Fatalf("列出类别失败：%v", err)
	}
	counts := make(map[string]model.SystemCategory, len(items))
	for _, item := range items {
		counts[item.ID] = item
	}
	if counts[category.ID].ReferenceCount != 2 || counts[category.ID].NodeCount != 2 {
		t.Fatalf("类别计数不符：%+v", counts[category.ID])
	}
	if counts[other.ID].ReferenceCount != 1 {
		t.Fatalf("另一类别引用计数应为 1：%+v", counts[other.ID])
	}

	nodes, err := repo.ListNodes(ctx, category.ID, "")
	if err != nil {
		t.Fatalf("列出节点失败：%v", err)
	}
	refByNode := make(map[string]int64, len(nodes))
	for _, node := range nodes {
		refByNode[node.ID] = node.ReferenceCount
	}
	if refByNode[nodeA.ID] != 2 || refByNode[nodeB.ID] != 0 {
		t.Fatalf("节点引用计数应按人物去重：%+v", refByNode)
	}

	if has, err := repo.AnyReferenceForCategory(ctx, category.ID); err != nil || !has {
		t.Fatalf("类别引用应为 true：%v err=%v", has, err)
	}
	if has, err := repo.AnyReferenceForNode(ctx, nodeB.ID); err != nil || has {
		t.Fatalf("无引用节点应为 false：%v err=%v", has, err)
	}

	rows, err := repo.ListReferencesByNode(ctx, nodeA.ID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("节点引用明细应为 2 条，实际 %d err=%v", len(rows), err)
	}
	if rows[0].NodeName != "剑" {
		t.Fatalf("引用明细节点名不符：%+v", rows[0])
	}
}

func TestSystemRepoReorderAndNodeLifecycle(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	work := h.createWork("仓储排序")
	repo := systemRepo(h)
	category := seedCategory(t, h, repo, work.ID, "排序", model.CategoryTypeNormal)

	a := seedNode(t, h, repo, work.ID, category.ID, nil, "甲", model.ReviewStatusReviewed)
	b := seedNode(t, h, repo, work.ID, category.ID, nil, "乙", model.ReviewStatusReviewed)
	c := seedNode(t, h, repo, work.ID, category.ID, nil, "丙", model.ReviewStatusReviewed)

	siblings, err := repo.SiblingIDs(ctx, category.ID, nil)
	if err != nil || len(siblings) != 3 {
		t.Fatalf("同级集合应为 3 条，实际 %v err=%v", siblings, err)
	}
	if siblings[0] != a.ID || siblings[2] != c.ID {
		t.Fatalf("初始排序不符：%v", siblings)
	}
	if next, err := repo.NextNodeSortOrder(ctx, category.ID, nil); err != nil || next != 3 {
		t.Fatalf("下一个节点排序应为 3，实际 %d err=%v", next, err)
	}

	reversed := []string{c.ID, b.ID, a.ID}
	if err := repo.ReorderNodes(ctx, reversed, time.Now()); err != nil {
		t.Fatalf("重排失败：%v", err)
	}
	siblings, err = repo.SiblingIDs(ctx, category.ID, nil)
	if err != nil || siblings[0] != c.ID || siblings[2] != a.ID {
		t.Fatalf("重排后顺序不符：%v err=%v", siblings, err)
	}
	if err := model.ValidateSiblingOrder(siblings, []string{c.ID, b.ID}); err == nil {
		t.Fatal("不完整排序集合应被拒绝")
	}

	now := time.Now()
	if rows, err := repo.SoftDeleteNode(ctx, a.ID, a.Version, "manual_delete", now); err != nil || rows != 1 {
		t.Fatalf("软删除节点失败：%d err=%v", rows, err)
	}
	deleted, err := repo.GetDeletedNodeRoot(ctx, a.ID)
	if err != nil || deleted.DeletedRootID == nil || *deleted.DeletedRootID != a.ID {
		t.Fatalf("节点回收站根不符：%+v err=%v", deleted, err)
	}
	if err := repo.RenameDeletedNode(ctx, a.ID, "甲改", "甲改", now); err != nil {
		t.Fatalf("重命名回收站节点失败：%v", err)
	}
	if err := repo.RestoreNode(ctx, a.ID, time.Now()); err != nil {
		t.Fatalf("恢复节点失败：%v", err)
	}
	restored, err := repo.GetNode(ctx, a.ID)
	if err != nil || restored.Name != "甲改" || restored.DeletedAt != nil {
		t.Fatalf("恢复结果不符：%+v err=%v", restored, err)
	}

	if rows, err := repo.SoftDeleteNode(ctx, a.ID, restored.Version, "manual_delete", time.Now()); err != nil || rows != 1 {
		t.Fatalf("二次软删除失败：%d err=%v", rows, err)
	}
	if purged, err := repo.PurgeNode(ctx, a.ID); err != nil || purged != 1 {
		t.Fatalf("永久删除节点应删 1 行，实际 %d err=%v", purged, err)
	}
	if _, err := repo.GetNode(ctx, a.ID); err != model.ErrNotFound {
		t.Fatalf("永久删除后应 ErrNotFound，实际 %v", err)
	}
}

func TestSystemRepoCategoryNameAndEnabledHelpers(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	work := h.createWork("仓储辅助")
	repo := systemRepo(h)
	category := seedCategory(t, h, repo, work.ID, "辅助", model.CategoryTypeNormal)

	name, err := repo.CategoryName(ctx, category.ID)
	if err != nil || name != "辅助" {
		t.Fatalf("类别名称不符：%q err=%v", name, err)
	}
	enabled, err := repo.CategoryEnabled(ctx, category.ID)
	if err != nil || !enabled {
		t.Fatalf("类别应启用：%v err=%v", enabled, err)
	}
	if _, err := repo.CategoryEnabled(ctx, model.NewID()); err != model.ErrNotFound {
		t.Fatalf("不存在类别应 ErrNotFound，实际 %v", err)
	}

	node := seedNode(t, h, repo, work.ID, category.ID, nil, "辅助节点", model.ReviewStatusReviewed)
	if name, err := repo.NodeName(ctx, node.ID); err != nil || name != "辅助节点" {
		t.Fatalf("节点名称不符：%q err=%v", name, err)
	}
}

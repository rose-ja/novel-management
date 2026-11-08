//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
)

type trashItemPayload struct {
	RootType      string  `json:"root_type"`
	RootID        string  `json:"root_id"`
	Name          string  `json:"name"`
	ParentName    *string `json:"parent_name"`
	DeletedAt     *string `json:"deleted_at"`
	AffectedCount int64   `json:"affected_count"`
}

type trashPagePayload struct {
	Items    []trashItemPayload `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}

func (h *harness) listTrash(rootType string) trashPagePayload {
	h.t.Helper()
	path := "/api/v1/trash"
	if rootType != "" {
		path += "?root_type=" + rootType
	}
	rec := h.do(http.MethodGet, path, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取回收站失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[trashPagePayload](h.t, rec)
}

func TestSystemTrashListsThreeTypes(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("回收站三类型")
	category := h.createCategory(work.ID, "类别甲", model.CategoryTypeNormal)
	nodeA := h.createNode(category.ID, "节点甲", nil, nil, nil)
	h.createNode(category.ID, "节点乙", nil, nil, nil)

	rec := h.do(http.MethodDelete, "/api/v1/system-nodes/"+nodeA.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	nodes := h.listTrash("system_node")
	if nodes.Total != 1 || nodes.Items[0].RootID != nodeA.ID {
		t.Fatalf("节点回收站应只有节点甲：%+v", nodes)
	}
	if nodes.Items[0].ParentName == nil || *nodes.Items[0].ParentName != "类别甲" {
		t.Fatalf("节点 parent_name 应为类别名：%+v", nodes.Items[0])
	}

	categories := h.listTrash("system_category")
	if categories.Total != 1 || categories.Items[0].RootID != category.ID {
		t.Fatalf("类别回收站应只有类别甲：%+v", categories)
	}
	if categories.Items[0].ParentName == nil || *categories.Items[0].ParentName != "回收站三类型" {
		t.Fatalf("类别 parent_name 应为作品名：%+v", categories.Items[0])
	}
	if categories.Items[0].AffectedCount != 1 {
		t.Fatalf("类别受影响节点应为 1，实际 %d", categories.Items[0].AffectedCount)
	}

	all := h.listTrash("")
	if all.Total != 2 || all.Items[0].RootType != "system_category" {
		t.Fatalf("默认应聚合两类且按删除时间倒序：%+v", all)
	}

	if works := h.listTrash("work"); works.Total != 0 {
		t.Fatalf("作品仍在用，不应出现在回收站：%+v", works)
	}

	rec = h.do(http.MethodGet, "/api/v1/trash?root_type=unknown", nil, nil)
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	if field := errorField(t, envelope); field != "root_type" {
		t.Fatalf("非法 root_type 应指向 root_type，实际 %q", field)
	}
}

func TestSystemTrashRestoreCategory(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("类别恢复")
	category := h.createCategory(work.ID, "纪元", model.CategoryTypeEra)
	root := h.createNode(category.ID, "第一层", nil, nil, nil)
	rootTwo := h.createNode(category.ID, "第一层二", nil, nil, nil)
	child := h.createNode(category.ID, "第二层", &root.ID, nil, nil)

	rec := h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	h.createCategory(work.ID, "纪元", model.CategoryTypeEra)
	rec = h.do(http.MethodPost, "/api/v1/trash/system_category/"+category.ID+"/restore", map[string]any{}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "RESTORE_CONFLICT")

	// 同父级两个节点改成同名 → 恢复冲突，且只列出节点冲突项。
	rec = h.do(http.MethodPost, "/api/v1/trash/system_category/"+category.ID+"/restore", map[string]any{
		"new_name": "纪元二",
		"renames": []map[string]any{
			{"entity_type": "system_node", "entity_id": root.ID, "new_name": "重复"},
			{"entity_type": "system_node", "entity_id": rootTwo.ID, "new_name": "重复"},
		},
	}, nil)
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "RESTORE_CONFLICT")
	items := decodeConflictItems(t, envelope)
	if len(items) != 1 || items[0].EntityType != "system_node" {
		t.Fatalf("组内同名应产生 1 条节点冲突：%+v", items)
	}

	rec = h.do(http.MethodPost, "/api/v1/trash/system_category/"+category.ID+"/restore", map[string]any{
		"new_name": "纪元二",
		"renames": []map[string]any{
			{"entity_type": "system_node", "entity_id": root.ID, "new_name": "第一层改"},
			{"entity_type": "system_node", "entity_id": child.ID, "new_name": "第二层改"},
		},
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("恢复类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	payload := decodeBody[restorePayload](t, rec)
	if payload.Name != "纪元二" || payload.RestoredCount != 4 {
		t.Fatalf("恢复结果不符：%+v", payload)
	}

	tree := h.getNodes(category.ID, "")
	if len(tree.Items) != 2 ||
		tree.Items[0].Name != "第一层改" || len(tree.Items[0].Children) != 1 || tree.Items[0].Children[0].Name != "第二层改" ||
		tree.Items[1].Name != "第一层二" {
		t.Fatalf("恢复后节点名称不符：%+v", tree.Items)
	}
}

func TestSystemTrashRestoreNode(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("节点恢复")
	category := h.createCategory(work.ID, "受保护父级", model.CategoryTypeNormal)
	root := h.createNode(category.ID, "根节点", nil, nil, nil)
	child := h.createNode(category.ID, "子节点", &root.ID, nil, nil)

	rec := h.do(http.MethodDelete, "/api/v1/system-nodes/"+child.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除子节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec = h.do(http.MethodPost, "/api/v1/trash/system_node/"+child.ID+"/restore", map[string]any{}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "PARENT_DELETED")

	rec = h.do(http.MethodPost, "/api/v1/trash/system_category/"+category.ID+"/restore", map[string]any{
		"new_name": "受保护父级二",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("恢复类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if payload := decodeBody[restorePayload](t, rec); payload.RestoredCount != 2 {
		t.Fatalf("类别与随组节点应恢复 2 条：%+v", payload)
	}
	if remaining := h.listTrash("system_node"); remaining.Total != 1 || remaining.Items[0].RootID != child.ID {
		t.Fatalf("单独删除的子节点应仍在回收站：%+v", remaining)
	}

	rec = h.do(http.MethodPost, "/api/v1/trash/system_node/"+child.ID+"/restore", map[string]any{}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("恢复节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	// 活跃同名节点阻止恢复，需改名。
	h.createNode(category.ID, "同名节点", nil, nil, nil)
	extra := h.createNode(category.ID, "同名节点二", nil, nil, nil)
	rec = h.do(http.MethodDelete, "/api/v1/system-nodes/"+extra.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodPost, "/api/v1/trash/system_node/"+extra.ID+"/restore", map[string]any{
		"new_name": "同名节点",
	}, nil)
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "RESTORE_CONFLICT")
	if items := decodeConflictItems(t, envelope); len(items) != 1 || items[0].EntityType != "system_node" {
		t.Fatalf("节点恢复冲突不符：%+v", items)
	}
	rec = h.do(http.MethodPost, "/api/v1/trash/system_node/"+extra.ID+"/restore", map[string]any{
		"new_name": "同名节点二改",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("改名恢复节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
}

func TestSystemTrashUndoCategoryAndNode(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("体系撤销")
	category := h.createCategory(work.ID, "撤销类别", model.CategoryTypeNormal)
	h.createNode(category.ID, "根", nil, nil, nil)
	h.createNode(category.ID, "子", nil, nil, nil)

	rec := h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	deleted := decodeBody[categoryDeletePayload](t, rec)
	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销类别失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if payload := decodeBody[undoPayload](t, undo); !payload.Restored || payload.RootType != "system_category" {
		t.Fatalf("撤销结果不符：%+v", payload)
	}
	if nodes := h.getNodes(category.ID, ""); len(nodes.Items) != 2 {
		t.Fatalf("撤销后节点应恢复 2 个，实际 %d", len(nodes.Items))
	}

	again := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	assertErrorCode(t, again, http.StatusConflict, "ALREADY_DELETED")

	node := h.createNode(category.ID, "单独撤销", nil, nil, nil)
	rec = h.do(http.MethodDelete, "/api/v1/system-nodes/"+node.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	nodeDeleted := decodeBody[categoryDeletePayload](t, rec)
	undo = h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": nodeDeleted.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销节点失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if payload := decodeBody[undoPayload](t, undo); payload.RootType != "system_node" || payload.RootID != node.ID {
		t.Fatalf("撤销节点结果不符：%+v", payload)
	}
}

func TestSystemTrashPurgeCategoryAndNode(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("体系永久删除")
	category := h.createCategory(work.ID, "待永久删除", model.CategoryTypeNormal)
	node := h.createNode(category.ID, "被引用节点", nil, nil, nil)

	rec := h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	// 删除进回收站后补一条引用，验证永久删除阶段的引用阻断。
	seedReference(t, h, work.ID, "char-1", category.ID, node.ID)
	rec = h.do(http.MethodDelete, "/api/v1/trash/system_category/"+category.ID, map[string]any{"confirm": true}, nil)
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	if items := decodeReferenceItems(t, envelope); len(items) != 1 || items[0].Value != "被引用节点" {
		t.Fatalf("类别永久删除引用明细不符：%+v", items)
	}

	if err := h.tx.Exec("DELETE FROM character_system_refs WHERE work_id = ?", work.ID).Error; err != nil {
		t.Fatalf("清理引用失败：%v", err)
	}
	rec = h.do(http.MethodDelete, "/api/v1/trash/system_category/"+category.ID, map[string]any{"confirm": true}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("永久删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	var remaining int64
	if err := h.tx.Table("system_nodes").Where("category_id = ?", category.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("统计节点失败：%v", err)
	}
	if remaining != 0 {
		t.Fatalf("永久删除类别后节点应清空，实际 %d", remaining)
	}

	other := h.createCategory(work.ID, "节点永久删除", model.CategoryTypeNormal)
	leaf := h.createNode(other.ID, "叶子", nil, nil, nil)
	rec = h.do(http.MethodDelete, "/api/v1/system-nodes/"+leaf.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	seedReference(t, h, work.ID, "char-2", other.ID, leaf.ID)
	rec = h.do(http.MethodDelete, "/api/v1/trash/system_node/"+leaf.ID, map[string]any{"confirm": true}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")

	if err := h.tx.Exec("DELETE FROM character_system_refs WHERE work_id = ?", work.ID).Error; err != nil {
		t.Fatalf("清理引用失败：%v", err)
	}
	rec = h.do(http.MethodDelete, "/api/v1/trash/system_node/"+leaf.ID, map[string]any{"confirm": true}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("永久删除节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	var nodeRows int64
	if err := h.tx.Table("system_nodes").Where("id = ?", leaf.ID).Count(&nodeRows).Error; err != nil {
		t.Fatalf("统计节点失败：%v", err)
	}
	if nodeRows != 0 {
		t.Fatalf("永久删除后节点应不存在，实际 %d", nodeRows)
	}
}

func TestSystemTrashWorkCascade(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("作品级联")
	category := h.createCategory(work.ID, "级联类别", model.CategoryTypeNormal)
	h.createNode(category.ID, "根", nil, nil, nil)
	h.createNode(category.ID, "子", nil, nil, nil)

	rec := h.deleteWork(work.ID, work.Name, "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	page := h.listTrash("work")
	if page.Total != 1 || page.Items[0].AffectedCount != 4 {
		t.Fatalf("作品受影响条数应为 4（2 类别+2 节点）：%+v", page.Items)
	}

	rec = h.do(http.MethodPost, "/api/v1/trash/work/"+work.ID+"/restore", map[string]any{}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("恢复作品失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if payload := decodeBody[restorePayload](t, rec); payload.RestoredCount != 5 {
		t.Fatalf("恢复条数应为 5：%+v", payload)
	}
	if nodes := h.getNodes(category.ID, ""); len(nodes.Items) != 2 {
		t.Fatalf("作品恢复后节点应恢复 2 个，实际 %d", len(nodes.Items))
	}
}

func TestSystemTrashPurgeAllTypes(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("清空三类")
	category := h.createCategory(work.ID, "清空类别", model.CategoryTypeNormal)
	node := h.createNode(category.ID, "清空节点", nil, nil, nil)

	rec := h.do(http.MethodDelete, "/api/v1/system-nodes/"+node.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/system-categories/"+category.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.deleteWork(work.ID, work.Name, "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	if total := h.trashTotal(); total != 3 {
		t.Fatalf("回收站应有 3 个根，实际 %d", total)
	}
	rec = h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("清空失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	payload := decodeBody[purgeAllPayload](t, rec)
	if payload.Purged != 3 || payload.SnapshotID == "" {
		t.Fatalf("清空结果不符：%+v", payload)
	}
	if total := h.trashTotal(); total != 0 {
		t.Fatalf("清空后回收站应为空，实际 %d", total)
	}
}

func decodeConflictItems(t *testing.T, envelope errorEnvelope) []model.ConflictItem {
	t.Helper()
	var payload struct {
		Items []model.ConflictItem `json:"items"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &payload); err != nil {
		t.Fatalf("解析冲突明细失败：%v，body=%s", err, envelope.Error.Details)
	}
	return payload.Items
}

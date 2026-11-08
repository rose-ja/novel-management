//go:build integration

package integration

import (
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
)

type nodePayload struct {
	ID             string        `json:"id"`
	WorkID         string        `json:"work_id"`
	CategoryID     string        `json:"category_id"`
	ParentID       *string       `json:"parent_id"`
	Name           string        `json:"name"`
	SortOrder      int           `json:"sort_order"`
	EraStartText   *string       `json:"era_start_text"`
	EraEndText     *string       `json:"era_end_text"`
	ReviewStatus   string        `json:"review_status"`
	ReviewedAt     *string       `json:"reviewed_at"`
	ReviewedBy     *string       `json:"reviewed_by"`
	Version        int           `json:"version"`
	ReferenceCount int64         `json:"reference_count"`
	Children       []nodePayload `json:"children"`
}

type nodeListPayload struct {
	Items []nodePayload `json:"items"`
}

func (h *harness) getNodes(categoryID, query string) nodeListPayload {
	h.t.Helper()
	path := "/api/v1/system-categories/" + categoryID + "/nodes"
	if query != "" {
		path += "?" + query
	}
	rec := h.do(http.MethodGet, path, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[nodeListPayload](h.t, rec)
}

func (h *harness) createNode(categoryID, name string, parentID *string, eraStart, eraEnd *string) nodePayload {
	h.t.Helper()
	body := map[string]any{"category_id": categoryID, "name": name}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	if eraStart != nil {
		body["era_start_text"] = *eraStart
	}
	if eraEnd != nil {
		body["era_end_text"] = *eraEnd
	}
	rec := h.do(http.MethodPost, "/api/v1/system-nodes", body, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[nodePayload](h.t, rec)
}

func strPtr(value string) *string { return &value }

func TestSystemNodesTreeAndEra(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("节点树")
	era := h.createCategory(work.ID, "纪元体系", model.CategoryTypeEra)

	root := h.createNode(era.ID, "第一纪元", nil, strPtr("元年"), strPtr("一百年"))
	if root.ReviewStatus != "confirmed" || root.Version != 1 || root.SortOrder != 0 {
		t.Fatalf("新节点字段不符：%+v", root)
	}
	if root.EraStartText == nil || *root.EraStartText != "元年" || root.EraEndText == nil || *root.EraEndText != "一百年" {
		t.Fatalf("纪元文本不符：%+v", root)
	}
	child := h.createNode(era.ID, "第一纪元早期", &root.ID, strPtr(" 元年 "), nil)
	if child.ParentID == nil || *child.ParentID != root.ID || child.SortOrder != 0 {
		t.Fatalf("子节点字段不符：%+v", child)
	}
	if child.EraStartText == nil || *child.EraStartText != "元年" {
		t.Fatalf("纪元文本应规范化修剪：%+v", child)
	}
	h.createNode(era.ID, "第二纪元", nil, nil, nil)

	tree := h.getNodes(era.ID, "")
	if len(tree.Items) != 2 {
		t.Fatalf("默认树形应只返回 2 个根，实际 %d", len(tree.Items))
	}
	if len(tree.Items[0].Children) != 1 || tree.Items[0].Children[0].ID != child.ID {
		t.Fatalf("树形 children 不符：%+v", tree.Items[0])
	}

	flat := h.getNodes(era.ID, "flat=true")
	if len(flat.Items) != 3 {
		t.Fatalf("平铺应返回 3 条，实际 %d", len(flat.Items))
	}

	pending := h.getNodes(era.ID, "review_status=pending")
	if len(pending.Items) != 0 {
		t.Fatalf("本变更新节点均为 confirmed，pending 应为空：%+v", pending.Items)
	}

	rec := h.do(http.MethodGet, "/api/v1/system-categories/"+era.ID+"/nodes?review_status=unknown", nil, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
}

func TestSystemNodesHierarchyAndUnique(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("节点层级")
	category := h.createCategory(work.ID, "层级", model.CategoryTypeNormal)
	other := h.createCategory(work.ID, "另一层级", model.CategoryTypeNormal)

	root := h.createNode(category.ID, "Ｓｗｏｒｄ", nil, nil, nil)

	rec := h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"category_id": category.ID,
		"name":        "  sword  ",
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "NAME_CONFLICT")

	child := h.createNode(category.ID, "child", &root.ID, nil, nil)
	rec = h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"category_id": category.ID,
		"parent_id":   child.ID,
		"name":        "第三级",
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "NODE_LEVEL_LIMIT")

	rec = h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"category_id": other.ID,
		"parent_id":   root.ID,
		"name":        "跨类别",
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "NODE_PARENT_MISMATCH")

	rec = h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"category_id": category.ID,
		"parent_id":   model.NewID(),
		"name":        "父级不存在",
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "NODE_PARENT_MISMATCH")

	// 不同父级允许同名。
	h.createNode(category.ID, "sword", &root.ID, nil, nil)

	rec = h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"category_id":    category.ID,
		"name":           "带纪元",
		"era_start_text": "元年",
	}, nil)
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	if field := errorField(t, envelope); field != "era_start_text" {
		t.Fatalf("普通类别纪元文本应指向 era_start_text，实际 %q", field)
	}

	long := make([]rune, model.MaxEraTextRunes+1)
	for i := range long {
		long[i] = '年'
	}
	rec = h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"category_id":    category.ID,
		"name":           "超长纪元",
		"era_start_text": string(long),
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
}

func TestSystemNodesUpdate(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("节点更新")
	era := h.createCategory(work.ID, "更新纪元", model.CategoryTypeEra)
	node := h.createNode(era.ID, "旧名", nil, strPtr("元年"), strPtr("末年"))

	rec := h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"name": "新名",
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")

	rec = h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"name":       "新名",
		"sort_order": 5,
	}, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("更新节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	updated := decodeBody[nodePayload](t, rec)
	if updated.Name != "新名" || updated.SortOrder != 5 || updated.Version != 2 || updated.ReviewStatus != "confirmed" {
		t.Fatalf("更新结果不符：%+v", updated)
	}

	rec = h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"era_start_text": nil,
		"era_end_text":   "   ",
	}, map[string]string{"If-Match": `"2"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("清空纪元失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	cleared := decodeBody[nodePayload](t, rec)
	if cleared.EraStartText != nil || cleared.EraEndText != nil {
		t.Fatalf("纪元文本应被清空：%+v", cleared)
	}

	rec = h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"parent_id": model.NewID(),
	}, map[string]string{"If-Match": `"3"`})
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	if field := errorField(t, envelope); field != "parent_id" {
		t.Fatalf("parent_id 不可改应指向 parent_id，实际 %q", field)
	}

	rec = h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"category_type": "normal",
	}, map[string]string{"If-Match": `"3"`})
	envelope = assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	if field := errorField(t, envelope); field != "category_type" {
		t.Fatalf("category_type 不可改应指向 category_type，实际 %q", field)
	}

	rec = h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"name": "  ",
	}, map[string]string{"If-Match": `"3"`})
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
}

func TestSystemNodesDelete(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("节点删除")
	category := h.createCategory(work.ID, "删除节点", model.CategoryTypeNormal)
	root := h.createNode(category.ID, "根", nil, nil, nil)
	child := h.createNode(category.ID, "子", &root.ID, nil, nil)
	leaf := h.createNode(category.ID, "叶", nil, nil, nil)

	rec := h.do(http.MethodDelete, "/api/v1/system-nodes/"+root.ID, nil, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusConflict, "HAS_CHILDREN")

	seedReference(t, h, work.ID, "char-1", category.ID, child.ID)
	rec = h.do(http.MethodDelete, "/api/v1/system-nodes/"+child.ID, nil, map[string]string{"If-Match": `"1"`})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	if details := decodeReferenceItems(t, envelope); len(details) != 1 || details[0].Value != "子" {
		t.Fatalf("节点引用明细不符：%+v", details)
	}

	rec = h.do(http.MethodDelete, "/api/v1/system-nodes/"+leaf.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除叶子失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	result := decodeBody[categoryDeletePayload](t, rec)
	if !result.Deleted || result.UndoToken == "" {
		t.Fatalf("删除结果不符：%+v", result)
	}
	var deletedRows int64
	if err := h.tx.Table("system_nodes").
		Where("id = ? AND deleted_at IS NOT NULL AND deleted_root_id = id", leaf.ID).
		Count(&deletedRows).Error; err != nil {
		t.Fatalf("检查节点软删除失败：%v", err)
	}
	if deletedRows != 1 {
		t.Fatalf("节点应带删除三件套，实际 %d", deletedRows)
	}

	rec = h.do(http.MethodDelete, "/api/v1/system-nodes/"+root.ID, nil, map[string]string{"If-Match": `"2"`})
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestSystemNodesReorder(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("节点排序")
	category := h.createCategory(work.ID, "重排", model.CategoryTypeNormal)
	a := h.createNode(category.ID, "甲", nil, nil, nil)
	b := h.createNode(category.ID, "乙", nil, nil, nil)
	c := h.createNode(category.ID, "丙", nil, nil, nil)

	rec := h.do(http.MethodPost, "/api/v1/system-nodes/reorder", map[string]any{
		"category_id": category.ID,
		"ordered_ids": []string{c.ID, b.ID, a.ID},
	}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("重排失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	flat := h.getNodes(category.ID, "flat=true")
	var roots []string
	for _, node := range flat.Items {
		if node.ParentID == nil {
			roots = append(roots, node.ID)
		}
	}
	if len(roots) != 3 || roots[0] != c.ID || roots[2] != a.ID {
		t.Fatalf("重排后顺序不符：%v", roots)
	}

	rec = h.do(http.MethodPost, "/api/v1/system-nodes/reorder", map[string]any{
		"category_id": category.ID,
		"ordered_ids": []string{a.ID, b.ID},
	}, nil)
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	if field := errorField(t, envelope); field != "ordered_ids" {
		t.Fatalf("排序集合不完整应指向 ordered_ids，实际 %q", field)
	}

	rec = h.do(http.MethodPost, "/api/v1/system-nodes/reorder", map[string]any{
		"category_id": category.ID,
		"ordered_ids": []string{a.ID, b.ID, model.NewID()},
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
}

func TestSystemNodesDisabledCategory(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("停用类别")
	category := h.createCategory(work.ID, "待停用", model.CategoryTypeNormal)
	node := h.createNode(category.ID, "已有节点", nil, nil, nil)

	rec := h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{
		"is_enabled": false,
	}, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("停用类别失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec = h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"category_id": category.ID,
		"name":        "新节点",
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_DISABLED")

	rec = h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"name": "改名",
	}, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_DISABLED")

	rec = h.do(http.MethodDelete, "/api/v1/system-nodes/"+node.ID, nil, map[string]string{"If-Match": `"1"`})
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_DISABLED")

	rec = h.do(http.MethodPost, "/api/v1/system-nodes/reorder", map[string]any{
		"category_id": category.ID,
		"ordered_ids": []string{node.ID},
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "CATEGORY_DISABLED")

	// 停用类别仍可读取节点。
	if nodes := h.getNodes(category.ID, ""); len(nodes.Items) != 1 {
		t.Fatalf("停用类别应仍能读取节点：%d", len(nodes.Items))
	}
}

//go:build integration

package integration

import (
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
)

func TestSystemRefsWorkPurge(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("永久删除引用")
	category := h.createCategory(work.ID, "引用类别", model.CategoryTypeNormal)
	node := h.createNode(category.ID, "金丹", nil, nil, nil)
	seedReference(t, h, work.ID, "char-1", category.ID, node.ID)

	rec := h.deleteWork(work.ID, work.Name, "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec = h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	items := decodeReferenceItems(t, envelope)
	if len(items) != 1 || items[0].EntityID != "char-1" || items[0].Value != "金丹" || items[0].Name != nil || items[0].InTrash {
		t.Fatalf("作品引用明细不符：%+v", items)
	}

	if err := h.tx.Exec("DELETE FROM character_system_refs WHERE work_id = ?", work.ID).Error; err != nil {
		t.Fatalf("清理引用失败：%v", err)
	}
	rec = h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("清理引用后永久删除应成功（status=%d）：%s", rec.Code, rec.Body.String())
	}

	for _, table := range []string{"works", "system_categories", "system_nodes", "character_system_refs"} {
		var total int64
		column := "work_id"
		if table == "works" {
			column = "id"
		}
		if err := h.tx.Table(table).Where(column+" = ?", work.ID).Count(&total).Error; err != nil {
			t.Fatalf("统计 %s 残留失败：%v", table, err)
		}
		if total != 0 {
			t.Fatalf("永久删除后 %s 仍有 %d 行", table, total)
		}
	}
}

func TestSystemRefsPurgeAllDedup(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("清空引用")
	first := h.createCategory(work.ID, "第一类", model.CategoryTypeNormal)
	second := h.createCategory(work.ID, "第二类", model.CategoryTypeNormal)
	nodeA := h.createNode(first.ID, "节点甲", nil, nil, nil)
	nodeB := h.createNode(second.ID, "节点乙", nil, nil, nil)
	seedReference(t, h, work.ID, "char-1", first.ID, nodeA.ID)
	seedReference(t, h, work.ID, "char-1", second.ID, nodeB.ID)

	rec := h.deleteWork(work.ID, work.Name, "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec = h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	if items := decodeReferenceItems(t, envelope); len(items) != 1 || items[0].EntityID != "char-1" {
		t.Fatalf("同一人物跨类别应去重为 1 条：%+v", items)
	}

	if err := h.tx.Exec("DELETE FROM character_system_refs WHERE work_id = ?", work.ID).Error; err != nil {
		t.Fatalf("清理引用失败：%v", err)
	}
	rec = h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("清空回收站失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	result := decodeBody[purgeAllPayload](t, rec)
	if result.Purged != 1 || result.SnapshotID == "" {
		t.Fatalf("清空结果不符：%+v", result)
	}
	var remaining int64
	if err := h.tx.Table("system_nodes").Where("work_id = ?", work.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("统计节点残留失败：%v", err)
	}
	if remaining != 0 {
		t.Fatalf("清空后节点应无残留，实际 %d", remaining)
	}
}

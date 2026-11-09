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

	// 作品内人物引用指向本作品的类别/节点，永久删除作品时随组清理，不构成阻断。
	rec = h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("作品内引用不应阻断永久删除（status=%d）：%s", rec.Code, rec.Body.String())
	}

	for _, table := range []string{"works", "system_categories", "system_nodes", "character_system_refs", "characters", "character_names"} {
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

func TestSystemRefsPurgeAllWithWorkChildren(t *testing.T) {
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
	if rec.Code != http.StatusOK {
		t.Fatalf("清空回收站失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	result := decodeBody[purgeAllPayload](t, rec)
	if result.Purged != 1 || result.SnapshotID == "" {
		t.Fatalf("清空结果不符：%+v", result)
	}
	for _, table := range []string{"system_nodes", "characters", "character_names", "character_system_refs"} {
		var remaining int64
		if err := h.tx.Table(table).Where("work_id = ?", work.ID).Count(&remaining).Error; err != nil {
			t.Fatalf("统计 %s 残留失败：%v", table, err)
		}
		if remaining != 0 {
			t.Fatalf("清空后 %s 应无残留，实际 %d", table, remaining)
		}
	}
}

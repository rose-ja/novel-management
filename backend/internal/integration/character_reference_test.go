//go:build integration

package integration

import (
	"net/http"
	"testing"
)

func TestCharacterReferencesIncludeNameAndTrashState(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("引用收口")
	category := h.createCategory(work.ID, "血统", "normal")
	node := h.createCharacterNode(work.ID, category.ID, "", "王族")
	character := h.createCharacter(work.ID, "陆沉", nil, []map[string]string{
		{"category_id": category.ID, "node_id": node},
	})

	list := h.listCategories(work.ID, nil)
	for _, item := range list.Items {
		if item.ID == category.ID && item.ReferenceCount != 1 {
			t.Fatalf("类别引用计数应含人物，实际 %d", item.ReferenceCount)
		}
	}

	rec := h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{"is_enabled": false}, map[string]string{
		"If-Match": `"1"`,
	})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	items := decodeReferenceItems(t, envelope)
	if len(items) != 1 {
		t.Fatalf("引用明细条数不符：%+v", items)
	}
	if items[0].EntityID != character.ID || items[0].Name == nil || *items[0].Name != "陆沉" ||
		items[0].Value != "王族" || items[0].InTrash {
		t.Fatalf("活跃人物引用明细不符：%+v", items[0])
	}

	h.deleteCharacter(character.ID, "1")
	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{"is_enabled": false}, map[string]string{
		"If-Match": `"1"`,
	})
	envelope = assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	items = decodeReferenceItems(t, envelope)
	if len(items) != 1 || !items[0].InTrash || items[0].Name == nil || *items[0].Name != "陆沉" {
		t.Fatalf("回收站人物引用明细不符：%+v", items)
	}

	purged := h.do(http.MethodDelete, "/api/v1/trash/character/"+character.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除人物失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	rec = h.do(http.MethodPatch, "/api/v1/system-categories/"+category.ID, map[string]any{"is_enabled": false}, map[string]string{
		"If-Match": `"1"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("人物永久删除后类别应可停用（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if disabled := decodeBody[categoryPayload](t, rec); disabled.IsEnabled {
		t.Fatalf("类别应已停用：%+v", disabled)
	}
}

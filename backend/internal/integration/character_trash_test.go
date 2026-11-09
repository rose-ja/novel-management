//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type characterDeletePayload struct {
	ID        string `json:"id"`
	DeletedAt string `json:"deleted_at"`
	UndoToken string `json:"undo_token"`
	ExpiresAt string `json:"expires_at"`
}

func (h *harness) deleteCharacter(id, version string) characterDeletePayload {
	h.t.Helper()
	rec := h.do(http.MethodDelete, "/api/v1/characters/"+id, nil, map[string]string{
		"If-Match": `"` + version + `"`,
	})
	if rec.Code != http.StatusOK {
		h.t.Fatalf("删除人物失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[characterDeletePayload](h.t, rec)
}

func (h *harness) characterRowCount(id string) (characters, names, refs int64) {
	h.t.Helper()
	h.tx.Raw("SELECT count(*) FROM characters WHERE id = ?", id).Scan(&characters)
	h.tx.Raw("SELECT count(*) FROM character_names WHERE character_id = ?", id).Scan(&names)
	h.tx.Raw("SELECT count(*) FROM character_system_refs WHERE character_id = ?", id).Scan(&refs)
	return characters, names, refs
}

func TestCharacterDeleteUndoAndTrash(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("人物回收站")
	category := h.listCategories(work.ID, nil).Items[0]
	node := h.createCharacterNode(work.ID, category.ID, "", "帝境")
	character := h.createCharacter(work.ID, "陆沉", []characterNamePayload{{Kind: "alias", Value: "小圣"}}, []map[string]string{
		{"category_id": category.ID, "node_id": node},
	})

	missing := h.do(http.MethodDelete, "/api/v1/characters/"+character.ID, nil, nil)
	assertErrorCode(t, missing, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")

	deleted := h.deleteCharacter(character.ID, "1")
	if deleted.ID != character.ID || deleted.UndoToken == "" || deleted.DeletedAt == "" || deleted.ExpiresAt == "" {
		t.Fatalf("删除响应不完整：%+v", deleted)
	}

	gone := h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil)
	assertErrorCode(t, gone, http.StatusNotFound, "NOT_FOUND")

	trash := h.do(http.MethodGet, "/api/v1/trash?root_type=character", nil, nil)
	page := decodeBody[trashPage](t, trash)
	if page.Total != 1 || page.Items[0].RootType != "character" || page.Items[0].RootID != character.ID {
		t.Fatalf("回收站人物根不符：%+v", page)
	}
	if page.Items[0].ParentName == nil || *page.Items[0].ParentName != "人物回收站" || page.Items[0].AffectedCount != 1 {
		t.Fatalf("回收站父级或计数不符：%+v", page.Items[0])
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if payload := decodeBody[undoPayload](t, undo); !payload.Restored || payload.RootType != "character" {
		t.Fatalf("撤销结果不符：%+v", payload)
	}
	detail := h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("撤销后人物应可见（status=%d）：%s", detail.Code, detail.Body.String())
	}
	restored := decodeBody[characterPayload](t, detail)
	if len(restored.SystemRefs) != 1 || len(restored.Names) != 1 {
		t.Fatalf("撤销应保留体系引用与别名：%+v", restored)
	}
	if restored.Version != 3 {
		t.Fatalf("删除与恢复应各递增一次版本，实际 %d", restored.Version)
	}

	again := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	assertErrorCode(t, again, http.StatusConflict, "ALREADY_DELETED")

	second := h.deleteCharacter(character.ID, "3")
	repeat := h.do(http.MethodDelete, "/api/v1/characters/"+character.ID, nil, map[string]string{"If-Match": `"3"`})
	assertErrorCode(t, repeat, http.StatusConflict, "ALREADY_DELETED")
	if second.UndoToken == "" {
		t.Fatal("再次删除应发放新令牌")
	}

	purged := h.do(http.MethodDelete, "/api/v1/trash/character/"+character.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	characters, names, refs := h.characterRowCount(character.ID)
	if characters != 0 || names != 0 || refs != 0 {
		t.Fatalf("永久删除后应无残留：characters=%d names=%d refs=%d", characters, names, refs)
	}
	repeatPurge := h.do(http.MethodDelete, "/api/v1/trash/character/"+character.ID, map[string]any{"confirm": true}, nil)
	assertErrorCode(t, repeatPurge, http.StatusConflict, "ALREADY_DELETED")
}

func TestCharacterRestoreConflictAndParentDeleted(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("恢复冲突")
	first := h.createCharacter(work.ID, "Ace", nil, nil)
	h.deleteCharacter(first.ID, "1")

	active := h.createCharacter(work.ID, "Ace", nil, nil)
	conflict := h.do(http.MethodPost, "/api/v1/trash/character/"+first.ID+"/restore", map[string]any{}, nil)
	envelope := assertErrorCode(t, conflict, http.StatusUnprocessableEntity, "RESTORE_CONFLICT")
	var details struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil || len(details.Items) != 1 || details.Items[0].ID != first.ID {
		t.Fatalf("RESTORE_CONFLICT details 不符：%s", string(envelope.Error.Details))
	}

	if row := h.do(http.MethodGet, "/api/v1/characters/"+first.ID, nil, nil); row.Code != http.StatusNotFound {
		t.Fatal("冲突时不应恢复人物")
	}

	renamed := h.do(http.MethodPost, "/api/v1/trash/character/"+first.ID+"/restore", map[string]any{"new_name": "Ace II"}, nil)
	if renamed.Code != http.StatusOK {
		t.Fatalf("new_name 恢复失败（status=%d）：%s", renamed.Code, renamed.Body.String())
	}
	payload := decodeBody[restorePayload](t, renamed)
	if payload.Name != "Ace II" || payload.RestoredCount != 1 {
		t.Fatalf("恢复结果不符：%+v", payload)
	}
	if active.ID == first.ID {
		t.Fatal("夹具错误：应为两个不同人物")
	}

	second := h.createCharacter(work.ID, "Orphan", nil, nil)
	h.deleteCharacter(second.ID, "1")
	workDeleted := h.deleteWork(work.ID, "恢复冲突", "1")
	if workDeleted.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", workDeleted.Code, workDeleted.Body.String())
	}
	blocked := h.do(http.MethodPost, "/api/v1/trash/character/"+second.ID+"/restore", map[string]any{}, nil)
	assertErrorCode(t, blocked, http.StatusConflict, "PARENT_DELETED")
}

func TestCharacterWorkGroupLifecycle(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("作品联动")
	category := h.listCategories(work.ID, nil).Items[0]
	node := h.createCharacterNode(work.ID, category.ID, "", "帝境")
	character := h.createCharacter(work.ID, "陆沉", []characterNamePayload{{Kind: "alias", Value: "小圣"}}, []map[string]string{
		{"category_id": category.ID, "node_id": node},
	})

	deleted := h.deleteWork(work.ID, "作品联动", "1")
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	if gone := h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil); gone.Code != http.StatusNotFound {
		t.Fatalf("作品删除后人物应不可见，实际 %d", gone.Code)
	}
	if list := h.listCharacters(work.ID, ""); list.Total != 0 {
		t.Fatalf("作品删除后列表应为空，实际 %d", list.Total)
	}
	trash := decodeBody[trashPage](t, h.do(http.MethodGet, "/api/v1/trash?root_type=character", nil, nil))
	if trash.Total != 0 {
		t.Fatalf("随作品删除的人物不应作为独立根出现，实际 %d", trash.Total)
	}
	if h.trashTotal() != 1 {
		t.Fatalf("回收站应只有作品根，实际 %d", h.trashTotal())
	}

	restored := h.do(http.MethodPost, "/api/v1/trash/work/"+work.ID+"/restore", map[string]any{}, nil)
	if restored.Code != http.StatusOK {
		t.Fatalf("恢复作品失败（status=%d）：%s", restored.Code, restored.Body.String())
	}
	detail := h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("恢复作品后人物应回来（status=%d）：%s", detail.Code, detail.Body.String())
	}
	payload := decodeBody[characterPayload](t, detail)
	if len(payload.SystemRefs) != 1 || len(payload.Names) != 1 {
		t.Fatalf("恢复后体系引用与别名应原状返回：%+v", payload)
	}

	workDetail := h.do(http.MethodGet, "/api/v1/works/"+work.ID, nil, nil)
	workVersion := decodeBody[workPayload](t, workDetail).Version
	if again := h.deleteWork(work.ID, "作品联动", fmt.Sprint(workVersion)); again.Code != http.StatusOK {
		t.Fatalf("再次删除作品失败（status=%d）：%s", again.Code, again.Body.String())
	}
	purged := h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除作品失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	characters, names, refs := h.characterRowCount(character.ID)
	if characters != 0 || names != 0 || refs != 0 {
		t.Fatalf("永久删除作品后人物应物理清除：characters=%d names=%d refs=%d", characters, names, refs)
	}
}

func TestCharacterPurgeAll(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("清空人物")
	character := h.createCharacter(work.ID, "孤舟", nil, nil)
	h.deleteCharacter(character.ID, "1")

	other := h.createWork("清空作品")
	otherDeleted := h.deleteWork(other.ID, "清空作品", "1")
	if otherDeleted.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", otherDeleted.Code, otherDeleted.Body.String())
	}
	if h.trashTotal() != 2 {
		t.Fatalf("清空前回收站应有 2 条，实际 %d", h.trashTotal())
	}

	purged := h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	if purged.Code != http.StatusOK {
		t.Fatalf("清空失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	result := decodeBody[purgeAllPayload](t, purged)
	if result.Purged != 2 {
		t.Fatalf("应清空 2 条根，实际 %d", result.Purged)
	}
	if h.trashTotal() != 0 {
		t.Fatal("清空后回收站应为空")
	}
	characters, names, refs := h.characterRowCount(character.ID)
	if characters != 0 || names != 0 || refs != 0 {
		t.Fatalf("清空后人物应物理删除：characters=%d names=%d refs=%d", characters, names, refs)
	}
}

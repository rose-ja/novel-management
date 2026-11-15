//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/search"
)

// TestSearchDocumentsFollowTrashLifecycle 验证删除、恢复、撤销与永久删除后的检索可见性。
func TestSearchDocumentsFollowTrashLifecycle(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("检索回收站作品")
	other := h.createWork("检索隔离作品")
	character := h.createCharacter(work.ID, "司空摘天", nil, nil)
	otherCharacter := h.createCharacter(other.ID, "司空摘天", nil, nil)

	// 删除后不可见，恢复后重新可见。
	h.deleteCharacter(character.ID, "1")
	docs := mustLoadDocs(t, h, work.ID)
	if !docs[model.SearchEntityCharacter+":"+character.ID].IsDeleted {
		t.Fatal("删除人物后文档应标记 is_deleted=true")
	}
	restore := h.do(http.MethodPost, "/api/v1/trash/character/"+character.ID+"/restore", map[string]any{}, nil)
	if restore.Code != http.StatusOK {
		t.Fatalf("恢复人物失败（status=%d）：%s", restore.Code, restore.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	if docs[model.SearchEntityCharacter+":"+character.ID].IsDeleted {
		t.Fatal("恢复人物后文档应标记 is_deleted=false")
	}

	// 撤销删除。
	undoTarget := h.createCharacter(work.ID, "小圣", nil, nil)
	undoDelete := h.deleteCharacter(undoTarget.ID, "1")
	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": undoDelete.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销删除失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	if docs[model.SearchEntityCharacter+":"+undoTarget.ID].IsDeleted {
		t.Fatal("撤销删除后文档应标记 is_deleted=false")
	}

	// 永久删除后文档清理；其他作品同名实体不受影响。
	h.deleteCharacter(character.ID, "3")
	purged := h.do(http.MethodDelete, "/api/v1/trash/character/"+character.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除人物失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	if docs[model.SearchEntityCharacter+":"+character.ID] != nil {
		t.Fatal("永久删除后人物文档应被清理")
	}
	otherDocs := mustLoadDocs(t, h, other.ID)
	if otherDocs[model.SearchEntityCharacter+":"+otherCharacter.ID].IsDeleted {
		t.Fatal("其他作品同名人物不应受影响")
	}

	refresher := search.NewRefresher(h.tx)
	for _, workID := range []string{work.ID, other.ID} {
		consistency, err := refresher.Consistency(context.Background(), workID)
		if err != nil {
			t.Fatalf("一致性检查失败：%v", err)
		}
		if consistency.Drift != 0 {
			t.Fatalf("作品 %s 一致性漂移应为 0，实际 %d（%+v）", workID, consistency.Drift, consistency.ByType)
		}
	}
}

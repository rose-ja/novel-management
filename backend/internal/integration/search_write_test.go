//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/search"
)

// TestSearchDocumentsFollowCharacterAndFactionWrites 验证人物/势力写入后同事务刷新与引用方同步。
func TestSearchDocumentsFollowCharacterAndFactionWrites(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("检索写入作品")
	character := h.createCharacter(work.ID, "司空摘天", []characterNamePayload{
		{Kind: "alias", Value: "小圣"},
		{Kind: "title", Value: "神逆"},
	}, nil)
	faction := h.createFaction(work.ID, "山海妖族", nil, &character.ID)
	memberRec := h.patchCharacter(character.ID, "1", map[string]any{"faction_id": faction.ID})
	if memberRec.Code != http.StatusOK {
		t.Fatalf("设置人物势力失败（status=%d）：%s", memberRec.Code, memberRec.Body.String())
	}

	docs := mustLoadDocs(t, h, work.ID)
	assertContains(t, "人物关联", docs[model.SearchEntityCharacter+":"+character.ID].RelationText, "山海妖族")
	assertContains(t, "势力关联", docs[model.SearchEntityFaction+":"+faction.ID].RelationText, "司空摘天")

	// 人物改名后：名称更新，旧名不再出现在 name_text，别名保留。
	rec := h.patchCharacter(character.ID, "2", map[string]any{"name": "司空摘星"})
	if rec.Code != http.StatusOK {
		t.Fatalf("人物改名失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	renamed := docs[model.SearchEntityCharacter+":"+character.ID]
	if renamed.NameText != "司空摘星" {
		t.Fatalf("人物文档名称未更新：%q", renamed.NameText)
	}
	assertContains(t, "人物别名", renamed.AliasText, "小圣")
	assertContains(t, "人物称号", renamed.TitleText, "神逆")

	// 势力改名后，人物的关联实体文本同步。
	factionRec := h.do(http.MethodPatch, "/api/v1/factions/"+faction.ID, map[string]any{
		"name": "山海神族",
	}, map[string]string{"If-Match": `"1"`})
	if factionRec.Code != http.StatusOK {
		t.Fatalf("势力改名失败（status=%d）：%s", factionRec.Code, factionRec.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	assertContains(t, "人物关联", docs[model.SearchEntityCharacter+":"+character.ID].RelationText, "山海神族")
	assertContains(t, "势力关联", docs[model.SearchEntityFaction+":"+faction.ID].RelationText, "司空摘星")

	// 清空首领后，势力关联文本不再包含人物。
	clearRec := h.do(http.MethodPatch, "/api/v1/factions/"+faction.ID, map[string]any{
		"leader_character_id": nil,
	}, map[string]string{"If-Match": `"2"`})
	if clearRec.Code != http.StatusOK {
		t.Fatalf("清空首领失败（status=%d）：%s", clearRec.Code, clearRec.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	if docs[model.SearchEntityFaction+":"+faction.ID].RelationText != nil {
		t.Fatalf("清空首领后势力关联文本应为空，实际 %q", *docs[model.SearchEntityFaction+":"+faction.ID].RelationText)
	}

	// 删除人物后文档标记删除。
	h.deleteCharacter(character.ID, "3")
	docs = mustLoadDocs(t, h, work.ID)
	if !docs[model.SearchEntityCharacter+":"+character.ID].IsDeleted {
		t.Fatal("删除人物后文档应标记 is_deleted=true")
	}

	consistency, err := search.NewRefresher(h.tx).Consistency(context.Background(), work.ID)
	if err != nil {
		t.Fatalf("一致性检查失败：%v", err)
	}
	if consistency.Drift != 0 {
		t.Fatalf("一致性漂移应为 0，实际 %d（%+v）", consistency.Drift, consistency.ByType)
	}
}

func mustLoadDocs(t *testing.T, h *harness, workID string) map[string]*model.SearchDocument {
	t.Helper()
	docs, err := loadSearchDocs(h, workID)
	if err != nil {
		t.Fatalf("读取检索文档失败：%v", err)
	}
	return docs
}

//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/search"
)

// TestSearchDocumentsFollowLocationRankingAndSystemWrites 验证地点、榜单、体系写入后文档同步。
func TestSearchDocumentsFollowLocationRankingAndSystemWrites(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("检索关系作品")
	character := h.createCharacter(work.ID, "司空摘天", nil, nil)
	faction := h.createFaction(work.ID, "山海妖族", nil, nil)
	location := h.createLocation(work.ID, "青丘", nil, nil)

	// 人物关联地点：地点文档应包含人物名，人物文档应包含地点名。
	linkRec := h.patchCharacter(character.ID, "1", map[string]any{
		"location_links": []map[string]any{{"location_id": location.ID}},
	})
	if linkRec.Code != http.StatusOK {
		t.Fatalf("关联人物地点失败（status=%d）：%s", linkRec.Code, linkRec.Body.String())
	}
	addFaction := h.do(http.MethodPost, "/api/v1/locations/"+location.ID+"/factions", map[string]any{
		"faction_id": faction.ID,
	}, nil)
	if addFaction.Code != http.StatusCreated {
		t.Fatalf("关联地点势力失败（status=%d）：%s", addFaction.Code, addFaction.Body.String())
	}
	docs := mustLoadDocs(t, h, work.ID)
	assertContains(t, "地点关联", docs[model.SearchEntityLocation+":"+location.ID].RelationText, "司空摘天", "山海妖族")
	assertContains(t, "人物关联", docs[model.SearchEntityCharacter+":"+character.ID].RelationText, "青丘")

	// 体系节点与人物引用。
	category := h.createCategory(work.ID, "检索境界", "normal")
	nodeResp := h.do(http.MethodPost, "/api/v1/system-nodes", map[string]any{
		"work_id":     work.ID,
		"category_id": category.ID,
		"name":        "圣者境",
	}, nil)
	if nodeResp.Code != http.StatusCreated {
		t.Fatalf("创建体系节点失败（status=%d）：%s", nodeResp.Code, nodeResp.Body.String())
	}
	node := decodeBody[struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}](t, nodeResp)
	refRec := h.patchCharacter(character.ID, "2", map[string]any{
		"system_refs": []map[string]any{{"category_id": category.ID, "node_id": node.ID}},
	})
	if refRec.Code != http.StatusOK {
		t.Fatalf("设置人物体系引用失败（status=%d）：%s", refRec.Code, refRec.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	assertContains(t, "节点关联", docs[model.SearchEntitySystemNode+":"+node.ID].RelationText, "检索境界")
	assertContains(t, "人物关联", docs[model.SearchEntityCharacter+":"+character.ID].RelationText, "圣者境", "检索境界")

	// 节点改名后，节点文档与人物关联同步。
	nodePatch := h.do(http.MethodPatch, "/api/v1/system-nodes/"+node.ID, map[string]any{
		"name": "大圣境",
	}, map[string]string{"If-Match": `"1"`})
	if nodePatch.Code != http.StatusOK {
		t.Fatalf("节点改名失败（status=%d）：%s", nodePatch.Code, nodePatch.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	if docs[model.SearchEntitySystemNode+":"+node.ID].NameText != "大圣境" {
		t.Fatalf("节点文档名称未更新：%q", docs[model.SearchEntitySystemNode+":"+node.ID].NameText)
	}
	assertContains(t, "人物关联", docs[model.SearchEntityCharacter+":"+character.ID].RelationText, "大圣境")

	// 榜单席位与改名。
	ranking := h.createRanking(work.ID, "天榜", 10)
	seatRec := h.do(http.MethodPut, "/api/v1/rankings/"+ranking.ID+"/seats/1", map[string]any{
		"character_id": character.ID,
	}, map[string]string{"If-Match": `"1"`})
	if seatRec.Code != http.StatusOK {
		t.Fatalf("指派席位失败（status=%d）：%s", seatRec.Code, seatRec.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	assertContains(t, "榜单关联", docs[model.SearchEntityRanking+":"+ranking.ID].RelationText, "司空摘天")
	assertContains(t, "人物关联", docs[model.SearchEntityCharacter+":"+character.ID].RelationText, "天榜")

	rankingPatch := h.do(http.MethodPatch, "/api/v1/rankings/"+ranking.ID, map[string]any{
		"name": "地榜",
	}, map[string]string{"If-Match": `"1"`})
	if rankingPatch.Code != http.StatusOK {
		t.Fatalf("榜单改名失败（status=%d）：%s", rankingPatch.Code, rankingPatch.Body.String())
	}
	docs = mustLoadDocs(t, h, work.ID)
	if docs[model.SearchEntityRanking+":"+ranking.ID].NameText != "地榜" {
		t.Fatalf("榜单文档名称未更新：%q", docs[model.SearchEntityRanking+":"+ranking.ID].NameText)
	}
	assertContains(t, "人物关联", docs[model.SearchEntityCharacter+":"+character.ID].RelationText, "地榜")

	consistency, err := search.NewRefresher(h.tx).Consistency(context.Background(), work.ID)
	if err != nil {
		t.Fatalf("一致性检查失败：%v", err)
	}
	if consistency.Drift != 0 {
		t.Fatalf("一致性漂移应为 0，实际 %d（%+v）", consistency.Drift, consistency.ByType)
	}
}

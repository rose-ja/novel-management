//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestLocationCharacterRelation(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点人物作品")
	other := h.createWork("地点人物他作")
	first := h.createLocation(work.ID, "东荒", nil, nil)
	second := h.createLocation(work.ID, "青丘山", &first.ID, nil)
	foreign := h.createLocation(other.ID, "西洲", nil, nil)

	rec := h.do(http.MethodPost, "/api/v1/characters", map[string]any{
		"work_id": work.ID,
		"name":    "涂山氏",
		"location_links": []map[string]any{
			{"location_id": first.ID, "note": "出生地"},
			{"location_id": second.ID, "note": "常驻"},
			{"location_id": first.ID, "note": "祖地"},
		},
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建带地点关系的人物失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	character := decodeBody[characterPayload](t, rec)
	if len(character.LocationLinks) != 2 {
		t.Fatalf("重复地点应去重为 2 条，实际 %d：%+v", len(character.LocationLinks), character.LocationLinks)
	}

	detail := h.getLocation(first.ID)
	if len(detail.Relations.Characters) != 1 {
		t.Fatalf("地点详情应显示 1 个人物，实际 %d", len(detail.Relations.Characters))
	}
	if detail.Relations.Characters[0].Note == nil || *detail.Relations.Characters[0].Note != "祖地" {
		t.Fatalf("重复地点备注应保留最后一条：%+v", detail.Relations.Characters[0].Note)
	}
	if detail.CharactersCount != 1 {
		t.Fatalf("地点关联人物数应为 1，实际 %d", detail.CharactersCount)
	}

	// 人物详情两侧一致。
	rec = h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil)
	got := decodeBody[characterPayload](t, rec)
	if len(got.LocationLinks) != 2 || got.LocationLinks[0].Name == "" {
		t.Fatalf("人物详情地点关系不完整：%+v", got.LocationLinks)
	}

	// 修改备注后地点详情同步。
	rec = h.do(http.MethodPatch, "/api/v1/characters/"+character.ID, map[string]any{
		"location_links": []map[string]any{
			{"location_id": first.ID, "note": "祖地（新）"},
			{"location_id": second.ID, "note": "常驻"},
		},
	}, map[string]string{"If-Match": `"` + itoa(character.Version) + `"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("更新人物地点关系失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	detail = h.getLocation(first.ID)
	if detail.Relations.Characters[0].Note == nil || *detail.Relations.Characters[0].Note != "祖地（新）" {
		t.Fatalf("备注修改未同步：%+v", detail.Relations.Characters[0].Note)
	}

	// 列表按地点筛选。
	list := h.listCharacters(work.ID, "&location_id="+first.ID)
	if list.Total != 1 {
		t.Fatalf("按地点筛选应返回 1 人，实际 %d", list.Total)
	}

	// 跨作品地点拒绝。
	rec = h.do(http.MethodPatch, "/api/v1/characters/"+character.ID, map[string]any{
		"location_links": []map[string]any{{"location_id": foreign.ID}},
	}, map[string]string{"If-Match": `"2"`})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("跨作品地点应返回 422，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 人物删除不被地点关系阻止；地点详情保留关系并标记回收站。
	rec = h.do(http.MethodDelete, "/api/v1/characters/"+character.ID, nil,
		map[string]string{"If-Match": `"2"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("人物删除不应被地点关系阻止（status=%d）：%s", rec.Code, rec.Body.String())
	}
	detail = h.getLocation(first.ID)
	if len(detail.Relations.Characters) != 1 || !detail.Relations.Characters[0].InTrash {
		t.Fatalf("人物删除后地点详情应保留关系并标记回收站：%+v", detail.Relations.Characters)
	}
	if detail.CharactersCount != 0 {
		t.Fatalf("回收站人物不应计入活跃关联数，实际 %d", detail.CharactersCount)
	}

	// 关系行仍在：删除地点被阻止，并标记引用方在回收站。
	rec = h.do(http.MethodDelete, "/api/v1/locations/"+second.ID, nil,
		map[string]string{"If-Match": `"1"`})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	var items []struct {
		Type     string `json:"type"`
		EntityID string `json:"entity_id"`
		InTrash  bool   `json:"in_trash"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &items); err != nil {
		t.Fatalf("解析引用明细失败：%v（%s）", err, string(envelope.Error.Details))
	}
	if len(items) != 1 || items[0].Type != "character_location" || items[0].EntityID != character.ID || !items[0].InTrash {
		t.Fatalf("地点删除阻断明细不正确：%+v", items)
	}
}

func TestLocationFactionRelation(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点势力作品")
	other := h.createWork("地点势力他作")
	location := h.createLocation(work.ID, "青云山", nil, nil)
	faction := h.createFaction(work.ID, "青云宗", nil, nil)
	foreign := h.createFaction(other.ID, "外宗", nil, nil)

	rec := h.do(http.MethodPost, "/api/v1/locations/"+location.ID+"/factions", map[string]any{
		"faction_id": faction.ID,
		"note":       "宗门驻地",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("新增地点-势力关系失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodPost, "/api/v1/locations/"+location.ID+"/factions", map[string]any{
		"faction_id": faction.ID,
	}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("重复关系应返回 409，实际 %d：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodPost, "/api/v1/locations/"+location.ID+"/factions", map[string]any{
		"faction_id": foreign.ID,
	}, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("跨作品势力应返回 422，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 备注更新。
	rec = h.do(http.MethodPatch, "/api/v1/locations/"+location.ID+"/factions/"+faction.ID, map[string]any{
		"note": "宗门驻地（新）",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("更新关系备注失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	// 地点与势力详情双向一致。
	locDetail := h.getLocation(location.ID)
	if len(locDetail.Relations.Factions) != 1 || *locDetail.Relations.Factions[0].Note != "宗门驻地（新）" {
		t.Fatalf("地点详情势力关系不正确：%+v", locDetail.Relations.Factions)
	}
	factionDetail := h.getFaction(faction.ID)
	if len(factionDetail.Relations.Locations) != 1 ||
		factionDetail.Relations.Locations[0].ID != location.ID ||
		*factionDetail.Relations.Locations[0].Note != "宗门驻地（新）" {
		t.Fatalf("势力详情地点关系不正确：%+v", factionDetail.Relations.Locations)
	}

	// 地点删除被势力关系阻止。
	rec = h.do(http.MethodDelete, "/api/v1/locations/"+location.ID, nil,
		map[string]string{"If-Match": `"1"`})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	var items []struct {
		Type     string `json:"type"`
		EntityID string `json:"entity_id"`
		InTrash  bool   `json:"in_trash"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &items); err != nil {
		t.Fatalf("解析引用明细失败：%v（%s）", err, string(envelope.Error.Details))
	}
	if len(items) != 1 || items[0].Type != "location_faction" || items[0].EntityID != faction.ID || items[0].InTrash {
		t.Fatalf("地点删除阻断明细不正确：%+v", items)
	}

	// 删除关系后地点可删除。
	rec = h.do(http.MethodDelete, "/api/v1/locations/"+location.ID+"/factions/"+faction.ID, nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("删除关系失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/locations/"+location.ID, nil,
		map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("移除关系后地点应可删除（status=%d）：%s", rec.Code, rec.Body.String())
	}
}

//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCharacterFactionIntegration(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("人物势力")
	other := h.createWork("人物势力-其他")
	faction := h.createFaction(work.ID, "妖族", nil, nil)
	foreign := h.createFaction(other.ID, "外族", nil, nil)

	rec := h.postCharacter(map[string]any{"work_id": work.ID, "name": "青丘氏", "faction_id": faction.ID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建带势力人物失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	created := decodeBody[characterPayload](t, rec)
	if created.Faction == nil || created.Faction.ID != faction.ID || created.Faction.Name != "妖族" {
		t.Fatalf("创建响应应返回真实势力：%+v", created.Faction)
	}

	detail := decodeBody[characterPayload](t, h.do(http.MethodGet, "/api/v1/characters/"+created.ID, nil, nil))
	if detail.Faction == nil || detail.Faction.ID != faction.ID {
		t.Fatalf("详情应返回真实势力：%+v", detail.Faction)
	}

	list := h.listCharacters(work.ID, "faction_id="+faction.ID)
	if list.Total != 1 || list.Items[0].Faction == nil || list.Items[0].Faction.Name != "妖族" {
		t.Fatalf("按势力筛选不符：%+v", list)
	}

	// 跨作品势力 → CROSS_WORK_REFERENCE。
	rec = h.postCharacter(map[string]any{"work_id": work.ID, "name": "跨界者", "faction_id": foreign.ID})
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "CROSS_WORK_REFERENCE")
	if field := decodeFieldDetails(t, envelope); field != "faction_id" {
		t.Fatalf("错误字段应为 faction_id，实际 %s", field)
	}

	// 已删势力 → PARENT_DELETED。
	trashable := h.createFaction(work.ID, "待删势力", nil, nil)
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+trashable.ID, nil, map[string]string{
		"If-Match": `"1"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.postCharacter(map[string]any{"work_id": work.ID, "name": "迟到者", "faction_id": trashable.ID})
	assertErrorCode(t, rec, http.StatusConflict, "PARENT_DELETED")

	// 清空势力。
	patched := h.patchCharacter(created.ID, itoa(created.Version), map[string]any{"faction_id": nil})
	if patched.Code != http.StatusOK {
		t.Fatalf("清空势力失败（status=%d）：%s", patched.Code, patched.Body.String())
	}
	if payload := decodeBody[characterPayload](t, patched); payload.Faction != nil {
		t.Fatalf("清空后 faction 应为 null：%+v", payload.Faction)
	}
}

func TestCharacterDeleteBlockedByFactionLeader(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("首领阻断")
	leader := h.createCharacter(work.ID, "首领", nil, nil)
	faction := h.createFaction(work.ID, "山海妖族", nil, &leader.ID)

	rec := h.do(http.MethodDelete, "/api/v1/characters/"+leader.ID, nil, map[string]string{
		"If-Match": `"` + itoa(leader.Version) + `"`,
	})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	var items []struct {
		Type     string `json:"type"`
		EntityID string `json:"entity_id"`
		InTrash  bool   `json:"in_trash"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &items); err != nil {
		t.Fatalf("解析引用明细失败：%v（%s）", err, string(envelope.Error.Details))
	}
	if len(items) != 1 || items[0].Type != "faction_leader" || items[0].EntityID != faction.ID || items[0].InTrash {
		t.Fatalf("首领阻断明细不正确：%+v", items)
	}

	// 清空首领后可删除。
	patched := h.patchFaction(faction.ID, `"1"`, map[string]any{"leader_character_id": nil})
	if patched.Leader != nil {
		t.Fatalf("清空首领失败：%+v", patched.Leader)
	}
	rec = h.do(http.MethodDelete, "/api/v1/characters/"+leader.ID, nil, map[string]string{
		"If-Match": `"` + itoa(leader.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("清空首领后删除人物失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	// 势力已在回收站时仍阻断删除首领人物。
	second := h.createCharacter(work.ID, "二首领", nil, nil)
	trashFaction := h.createFaction(work.ID, "已删势力", nil, &second.ID)
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+trashFaction.ID, nil, map[string]string{
		"If-Match": `"1"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/characters/"+second.ID, nil, map[string]string{
		"If-Match": `"` + itoa(second.Version) + `"`,
	})
	envelope = assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	items = nil
	if err := json.Unmarshal(envelope.Error.Details, &items); err != nil {
		t.Fatalf("解析引用明细失败：%v", err)
	}
	if len(items) != 1 || !items[0].InTrash {
		t.Fatalf("回收站中的势力仍应阻断首领人物删除：%+v", items)
	}
}

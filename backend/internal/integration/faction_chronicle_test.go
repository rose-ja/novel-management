//go:build integration

package integration

import (
	"net/http"
	"strings"
	"testing"
)

func TestFactionChronicleCRUDAndUndo(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("势力大事记")
	faction := h.createFaction(work.ID, "妖族", nil, nil)

	created := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "faction", "owner_id": faction.ID, "text": "妖族立族", "time_label": "元年前",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("创建势力大事记失败（status=%d）：%s", created.Code, created.Body.String())
	}
	item := decodeBody[chroniclePayload](t, created)
	if item.OwnerType != "faction" || item.OwnerID != faction.ID {
		t.Fatalf("归属不正确：%+v", item)
	}

	var list struct {
		Items []chroniclePayload `json:"items"`
	}
	listRec := h.do(http.MethodGet, "/api/v1/chronicles?owner_type=faction&owner_id="+faction.ID, nil, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("列表失败（status=%d）：%s", listRec.Code, listRec.Body.String())
	}
	if err := decodeJSON(t, listRec, &list); err != nil {
		t.Fatalf("解析列表失败：%v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != item.ID {
		t.Fatalf("势力大事记列表不正确：%+v", list.Items)
	}
	if detail := h.getFaction(faction.ID); detail.Relations.ChronicleCount != 1 {
		t.Fatalf("详情大事记计数应为 1，实际 %d", detail.Relations.ChronicleCount)
	}

	conflict := h.do(http.MethodPatch, "/api/v1/chronicles/"+item.ID, map[string]any{"text": "改写"}, map[string]string{
		"If-Match": `"999"`,
	})
	assertErrorCode(t, conflict, http.StatusConflict, "VERSION_CONFLICT")

	updated := h.do(http.MethodPatch, "/api/v1/chronicles/"+item.ID, map[string]any{"text": "妖族立族（修订）"}, map[string]string{
		"If-Match": `"` + itoa(item.Version) + `"`,
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("更新势力大事记失败（status=%d）：%s", updated.Code, updated.Body.String())
	}
	if payload := decodeBody[chroniclePayload](t, updated); payload.Text != "妖族立族（修订）" || payload.Version != item.Version+1 {
		t.Fatalf("更新结果不正确：%+v", payload)
	}

	tooLong := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "faction", "owner_id": faction.ID, "text": strings.Repeat("字", 2001),
	}, nil)
	envelope := assertErrorCode(t, tooLong, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	if field := decodeFieldDetails(t, envelope); field != "text" {
		t.Fatalf("错误字段应为 text，实际 %q", field)
	}

	deleted := h.do(http.MethodDelete, "/api/v1/chronicles/"+item.ID, nil, nil)
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	var deletedPayload struct {
		UndoToken string `json:"undo_token"`
	}
	if err := decodeJSON(t, deleted, &deletedPayload); err != nil {
		t.Fatalf("解析删除结果失败：%v", err)
	}
	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deletedPayload.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	listRec = h.do(http.MethodGet, "/api/v1/chronicles?owner_type=faction&owner_id="+faction.ID, nil, nil)
	if err := decodeJSON(t, listRec, &list); err != nil {
		t.Fatalf("撤销后解析列表失败：%v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("撤销后应恢复 1 条，实际 %d", len(list.Items))
	}

	// 归属不存在 → 404；未知类型 → 422。
	missing := h.do(http.MethodGet, "/api/v1/chronicles?owner_type=faction&owner_id=missing", nil, nil)
	assertErrorCode(t, missing, http.StatusNotFound, "NOT_FOUND")
	unsupported := h.do(http.MethodGet, "/api/v1/chronicles?owner_type=location&owner_id="+faction.ID, nil, nil)
	assertErrorCode(t, unsupported, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	// 势力在回收站时不允许新增大事记。
	trashable := h.createFaction(work.ID, "待删势力", nil, nil)
	rec := h.do(http.MethodDelete, "/api/v1/factions/"+trashable.ID, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	blocked := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "faction", "owner_id": trashable.ID, "text": "不应写入",
	}, nil)
	assertErrorCode(t, blocked, http.StatusConflict, "PARENT_DELETED")
}

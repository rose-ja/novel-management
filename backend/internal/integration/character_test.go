//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func (h *harness) postCharacter(body map[string]any) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.do(http.MethodPost, "/api/v1/characters", body, nil)
}

func (h *harness) patchCharacter(id, version string, body map[string]any) *httptest.ResponseRecorder {
	h.t.Helper()
	headers := map[string]string{}
	if version != "" {
		headers["If-Match"] = `"` + version + `"`
	}
	return h.do(http.MethodPatch, "/api/v1/characters/"+id, body, headers)
}

func decodeFieldDetails(t *testing.T, envelope errorEnvelope) string {
	t.Helper()
	var details struct {
		Field string `json:"field"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil {
		t.Fatalf("解析字段明细失败：%v（%s）", err, string(envelope.Error.Details))
	}
	return details.Field
}

func TestCharacterCreateValidationMatrix(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("人物校验")
	category := h.listCategories(work.ID, nil).Items[0]
	node := h.createCharacterNode(work.ID, category.ID, "", "帝境")

	created := h.postCharacter(map[string]any{"work_id": work.ID, "name": "陆沉"})
	if created.Code != http.StatusCreated {
		t.Fatalf("创建失败（status=%d）：%s", created.Code, created.Body.String())
	}
	if location := created.Header().Get("Location"); !strings.HasSuffix(location, "/api/v1/characters/"+decodeBody[characterPayload](t, created).ID) {
		t.Fatalf("Location 头不正确：%q", location)
	}
	if payload := decodeBody[characterPayload](t, created); payload.ReviewStatus != "confirmed" {
		t.Fatalf("初始校对态应为 confirmed，实际 %q", payload.ReviewStatus)
	}

	cases := []struct {
		name   string
		body   map[string]any
		status int
		code   string
		field  string
	}{
		{"缺作品", map[string]any{"name": "无主"}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "work_id"},
		{"作品已删", map[string]any{"work_id": "NOPE", "name": "无主"}, http.StatusUnprocessableEntity, "PARENT_DELETED", ""},
		{"空名称", map[string]any{"work_id": work.ID, "name": "  "}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name"},
		{"名称超长", map[string]any{"work_id": work.ID, "name": strings.Repeat("龙", 201)}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name"},
		{"简介超长", map[string]any{"work_id": work.ID, "name": "长简介", "description": strings.Repeat("龙", 20001)}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "description"},
		{"势力未开放", map[string]any{"work_id": work.ID, "name": "势力", "faction_id": "faction-1"}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "faction_id"},
		{"势力空串", map[string]any{"work_id": work.ID, "name": "势力空串", "faction_id": ""}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "faction_id"},
		{"地点未开放", map[string]any{"work_id": work.ID, "name": "地点", "location_links": []map[string]any{{"location_id": "loc-1"}}}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "location_links"},
		{"别名非法", map[string]any{"work_id": work.ID, "name": "别名", "names": []map[string]any{{"kind": "nickname", "value": "小圣"}}}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "names"},
		{"类别不存在", map[string]any{"work_id": work.ID, "name": "类别缺失", "system_refs": []map[string]any{{"category_id": "NOPE", "node_id": node}}}, http.StatusNotFound, "NOT_FOUND", ""},
		{"重复类别", map[string]any{"work_id": work.ID, "name": "重复类别", "system_refs": []map[string]any{
			{"category_id": category.ID, "node_id": node},
			{"category_id": category.ID, "node_id": node},
		}}, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "system_refs"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := h.postCharacter(tc.body)
			envelope := assertErrorCode(t, rec, tc.status, tc.code)
			if tc.field != "" && decodeFieldDetails(t, envelope) != tc.field {
				t.Fatalf("期望字段 %s，实际 %s", tc.field, decodeFieldDetails(t, envelope))
			}
		})
	}

	tooMany := make([]map[string]any, 101)
	for i := range tooMany {
		tooMany[i] = map[string]any{"kind": "alias", "value": fmt.Sprintf("别名%03d", i)}
	}
	rec := h.postCharacter(map[string]any{"work_id": work.ID, "name": "超量别名", "names": tooMany})
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	duplicate := h.postCharacter(map[string]any{"work_id": work.ID, "name": "陆沉"})
	envelope := assertErrorCode(t, duplicate, http.StatusConflict, "NAME_CONFLICT")
	var conflict struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &conflict); err != nil || len(conflict.Items) != 1 || conflict.Items[0].Name != "陆沉" {
		t.Fatalf("名称冲突明细不符：%s", string(envelope.Error.Details))
	}
	if rec := h.postCharacter(map[string]any{"work_id": work.ID, "name": "陆 沉"}); rec.Code != http.StatusCreated {
		t.Fatalf("名称仅差空白应允许：%s", rec.Body.String())
	}

	dedup := h.postCharacter(map[string]any{"work_id": work.ID, "name": "去重别名", "names": []map[string]any{
		{"kind": "alias", "value": "小圣"},
		{"kind": "alias", "value": " 小圣 "},
		{"kind": "title", "value": "小圣"},
	}})
	if dedup.Code != http.StatusCreated {
		t.Fatalf("别名去重创建失败（status=%d）：%s", dedup.Code, dedup.Body.String())
	}
	if payload := decodeBody[characterPayload](t, dedup); len(payload.Names) != 2 {
		t.Fatalf("同 kind 重复别名应去重且保留 title，实际 %+v", payload.Names)
	}
}

func TestCharacterCrossWorkAndDisabledCategory(t *testing.T) {
	h := newHarness(t)
	first := h.createWork("作品甲")
	second := h.createWork("作品乙")
	secondCategory := h.listCategories(second.ID, nil).Items[0]
	secondNode := h.createCharacterNode(second.ID, secondCategory.ID, "", "凡境")

	cross := h.postCharacter(map[string]any{"work_id": first.ID, "name": "越界", "system_refs": []map[string]any{
		{"category_id": secondCategory.ID, "node_id": secondNode},
	}})
	assertErrorCode(t, cross, http.StatusUnprocessableEntity, "CROSS_WORK_REFERENCE")

	firstCategory := h.listCategories(first.ID, nil).Items[0]
	otherCategory := h.createCategory(first.ID, "血统", "normal")
	otherNode := h.createCharacterNode(first.ID, otherCategory.ID, "", "王族")
	mismatch := h.postCharacter(map[string]any{"work_id": first.ID, "name": "错位节点", "system_refs": []map[string]any{
		{"category_id": firstCategory.ID, "node_id": otherNode},
	}})
	assertErrorCode(t, mismatch, http.StatusUnprocessableEntity, "NODE_PARENT_MISMATCH")

	disabledCategory := h.createCategory(first.ID, "禁术", "normal")
	disabledNode := h.createCharacterNode(first.ID, disabledCategory.ID, "", "禁术一重")
	disable := h.do(http.MethodPatch, "/api/v1/system-categories/"+disabledCategory.ID, map[string]any{"is_enabled": false}, map[string]string{
		"If-Match": fmt.Sprintf(`"%d"`, disabledCategory.Version),
	})
	if disable.Code != http.StatusOK {
		t.Fatalf("停用类别失败（status=%d）：%s", disable.Code, disable.Body.String())
	}
	blocked := h.postCharacter(map[string]any{"work_id": first.ID, "name": "引用停用类别", "system_refs": []map[string]any{
		{"category_id": disabledCategory.ID, "node_id": disabledNode},
	}})
	envelope := assertErrorCode(t, blocked, http.StatusUnprocessableEntity, "CATEGORY_DISABLED")
	var details struct {
		CategoryName string `json:"category_name"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil || details.CategoryName != "禁术" {
		t.Fatalf("CATEGORY_DISABLED 明细不符：%s", string(envelope.Error.Details))
	}

	if sameName := h.postCharacter(map[string]any{"work_id": first.ID, "name": "同名甲"}); sameName.Code != http.StatusCreated {
		t.Fatalf("作品甲同名创建失败：%s", sameName.Body.String())
	}
	if sameName := h.postCharacter(map[string]any{"work_id": second.ID, "name": "同名甲"}); sameName.Code != http.StatusCreated {
		t.Fatalf("跨作品同名应允许：%s", sameName.Body.String())
	}
	if list := h.listCharacters(second.ID, ""); list.Total != 1 {
		t.Fatalf("作品乙列表不应看到作品甲人物，实际 %d", list.Total)
	}
}

func TestCharacterUpdateOptimisticLockAndFields(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("人物更新")
	category := h.listCategories(work.ID, nil).Items[0]
	node := h.createCharacterNode(work.ID, category.ID, "", "帝境")
	character := h.createCharacter(work.ID, "陆沉", []characterNamePayload{{Kind: "alias", Value: "小圣"}}, []map[string]string{
		{"category_id": category.ID, "node_id": node},
	})

	missing := h.patchCharacter(character.ID, "", map[string]any{"name": "陆沉之"})
	assertErrorCode(t, missing, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")

	updated := h.patchCharacter(character.ID, "1", map[string]any{
		"name":        "陆沉之",
		"names":       []map[string]any{{"kind": "title", "value": "帝君"}},
		"description": nil,
		"system_refs": []map[string]any{},
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("更新失败（status=%d）：%s", updated.Code, updated.Body.String())
	}
	payload := decodeBody[characterPayload](t, updated)
	if payload.Name != "陆沉之" || payload.Version != 2 || len(payload.Names) != 1 || payload.Names[0].Kind != "title" || len(payload.SystemRefs) != 0 {
		t.Fatalf("更新结果不符：%+v", payload)
	}

	stale := h.patchCharacter(character.ID, "1", map[string]any{"description": "旧版本写入"})
	envelope := assertErrorCode(t, stale, http.StatusConflict, "VERSION_CONFLICT")
	var conflict struct {
		CurrentVersion int `json:"current_version"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &conflict); err != nil || conflict.CurrentVersion != 2 {
		t.Fatalf("版本冲突明细不符：%s", string(envelope.Error.Details))
	}
	current := h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil)
	if decodeBody[characterPayload](t, current).Version != 2 {
		t.Fatal("版本冲突不应写入")
	}

	unknown := h.patchCharacter(character.ID, "2", map[string]any{"work_id": "other"})
	assertErrorCode(t, unknown, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	emptyFaction := h.patchCharacter(character.ID, "2", map[string]any{"faction_id": ""})
	assertErrorCode(t, emptyFaction, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	other := h.createCharacter(work.ID, "顾清", nil, nil)
	rename := h.patchCharacter(other.ID, "1", map[string]any{"name": "陆沉之"})
	assertErrorCode(t, rename, http.StatusConflict, "NAME_CONFLICT")

	disabledCategory := h.createCategory(work.ID, "禁术", "normal")
	disabledNode := h.createCharacterNode(work.ID, disabledCategory.ID, "", "禁术一重")
	disable := h.do(http.MethodPatch, "/api/v1/system-categories/"+disabledCategory.ID, map[string]any{"is_enabled": false}, map[string]string{
		"If-Match": fmt.Sprintf(`"%d"`, disabledCategory.Version),
	})
	if disable.Code != http.StatusOK {
		t.Fatalf("停用类别失败（status=%d）：%s", disable.Code, disable.Body.String())
	}
	blocked := h.patchCharacter(character.ID, "2", map[string]any{"system_refs": []map[string]any{
		{"category_id": disabledCategory.ID, "node_id": disabledNode},
	}})
	assertErrorCode(t, blocked, http.StatusUnprocessableEntity, "CATEGORY_DISABLED")
}

func TestCharacterDatabaseConstraints(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("数据库约束")
	character := h.createCharacter(work.ID, "Sword", []characterNamePayload{{Kind: "alias", Value: "Blade"}}, nil)

	if err := h.tx.SavePoint("dup_name").Error; err != nil {
		t.Fatalf("创建保存点失败：%v", err)
	}
	err := h.tx.Exec(`
		INSERT INTO characters (id, work_id, name, name_key)
		VALUES ('DUPNAME', ?, 'sword', 'sword')`, work.ID).Error
	if err == nil {
		t.Fatal("活跃名称唯一索引应阻止大小写等价重名")
	}
	if err := h.tx.RollbackTo("dup_name").Error; err != nil {
		t.Fatalf("回滚保存点失败：%v", err)
	}

	if err := h.tx.Exec("DELETE FROM characters WHERE id = ?", character.ID).Error; err != nil {
		t.Fatalf("物理删除人物失败：%v", err)
	}
	characters, names, refs := h.characterRowCount(character.ID)
	if characters != 0 || names != 0 || refs != 0 {
		t.Fatalf("人物删除应级联清理别名，实际 characters=%d names=%d refs=%d", characters, names, refs)
	}
}

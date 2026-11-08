//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"novel-management/backend/internal/model"
)

func TestCreateWorkCreatesBuiltinCategoryAndActivates(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")

	if work.Version != 1 {
		t.Fatalf("新作品 version 应为 1，实际 %d", work.Version)
	}
	if work.DeletedAt != nil {
		t.Fatal("新作品不应带 deleted_at")
	}

	var category model.SystemCategory
	if err := h.tx.Where("work_id = ?", work.ID).First(&category).Error; err != nil {
		t.Fatalf("内置类别缺失：%v", err)
	}
	if category.Name != model.BuiltinRealmCategory || !category.IsBuiltin || category.CategoryType != "normal" {
		t.Fatalf("内置类别不符：%+v", category)
	}

	current := h.currentWorkID()
	if current == nil || *current != work.ID {
		t.Fatalf("新建作品应成为当前作品，实际 %v", current)
	}
}

func TestCreateWorkDuplicateName(t *testing.T) {
	h := newHarness(t)
	h.createWork("山海")

	rec := h.do(http.MethodPost, "/api/v1/works", map[string]any{"name": " 山海 "}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "NAME_CONFLICT")

	var total int64
	if err := h.tx.Table("works").Where("name_key = ?", "山海").Count(&total).Error; err != nil {
		t.Fatalf("统计失败：%v", err)
	}
	if total != 1 {
		t.Fatalf("同名作品应只有 1 条，实际 %d", total)
	}
}

func TestListWorksSearchAndPagination(t *testing.T) {
	h := newHarness(t)
	h.createWork("山海")
	h.createWork("星河")

	rec := h.do(http.MethodGet, "/api/v1/works?q=山&page=1&page_size=20", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("列表失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	page := decodeBody[workPage](t, rec)
	if len(page.Items) != 1 || page.Items[0].Name != "山海" {
		t.Fatalf("搜索结果不符：%+v", page.Items)
	}
	if page.Page != 1 || page.PageSize != 20 || page.Total != 1 {
		t.Fatalf("分页字段不符：%+v", page)
	}
}

func TestUpdateWorkIfMatch(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")

	rec := h.do(http.MethodPatch, "/api/v1/works/"+work.ID, map[string]any{"name": "山海传"}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")

	rec = h.do(http.MethodPatch, "/api/v1/works/"+work.ID, map[string]any{"name": "山海传"}, map[string]string{
		"If-Match": `"2"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")

	rec = h.do(http.MethodPatch, "/api/v1/works/"+work.ID, map[string]any{"name": "山海传"}, map[string]string{
		"If-Match": `"1"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("正确版本应更新成功（status=%d）：%s", rec.Code, rec.Body.String())
	}
	updated := decodeBody[workPayload](t, rec)
	if updated.Name != "山海传" || updated.Version != 2 {
		t.Fatalf("更新结果不符：%+v", updated)
	}
}

func TestUpdateWorkNameConflict(t *testing.T) {
	h := newHarness(t)
	h.createWork("山海")
	other := h.createWork("星河")

	rec := h.do(http.MethodPatch, "/api/v1/works/"+other.ID, map[string]any{"name": "山海"}, map[string]string{
		"If-Match": `"1"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "NAME_CONFLICT")
}

func TestDeleteWorkConfirmNameAndVersion(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")

	rec := h.deleteWork(work.ID, "错误名称", "1")
	envelope := assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	var details struct {
		Field string `json:"field"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil {
		t.Fatalf("解析 details 失败：%v", err)
	}
	if details.Field != "confirm_name" {
		t.Fatalf("details.field 应为 confirm_name，实际 %q", details.Field)
	}

	if check := h.do(http.MethodGet, "/api/v1/works/"+work.ID, nil, nil); check.Code != http.StatusOK {
		t.Fatalf("确认名不符时作品不应被删除（status=%d）", check.Code)
	}

	rec = h.do(http.MethodDelete, "/api/v1/works/"+work.ID, map[string]any{"confirm_name": "山海"}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestSnapshotFailureBlocksDelete(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")

	h.dumper.fail = true
	rec := h.deleteWork(work.ID, "山海", "1")
	assertErrorCode(t, rec, http.StatusServiceUnavailable, "SNAPSHOT_FAILED")

	if check := h.do(http.MethodGet, "/api/v1/works/"+work.ID, nil, nil); check.Code != http.StatusOK {
		t.Fatalf("快照失败时作品应保留（status=%d）", check.Code)
	}

	h.dumper.fail = false
	rec = h.deleteWork(work.ID, "山海", "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("快照恢复后删除应成功（status=%d）：%s", rec.Code, rec.Body.String())
	}
	result := decodeBody[deletePayload](t, rec)
	if !result.Deleted || result.UndoToken == "" || result.SnapshotID == "" {
		t.Fatalf("删除结果不完整：%+v", result)
	}
}

func TestDeleteThenUndo(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")

	rec := h.deleteWork(work.ID, "山海", "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	deleted := decodeBody[deletePayload](t, rec)

	if h.trashTotal() != 1 {
		t.Fatal("删除后回收站应有 1 条")
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if payload := decodeBody[undoPayload](t, undo); !payload.Restored || payload.RootID != work.ID {
		t.Fatalf("撤销结果不符：%+v", payload)
	}

	if check := h.do(http.MethodGet, "/api/v1/works/"+work.ID, nil, nil); check.Code != http.StatusOK {
		t.Fatalf("撤销后作品应可见（status=%d）", check.Code)
	}
	if h.currentWorkID() == nil {
		t.Fatal("撤销后当前作品应被补回")
	}
	if h.trashTotal() != 0 {
		t.Fatal("撤销后回收站应为空")
	}

	again := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	assertErrorCode(t, again, http.StatusConflict, "ALREADY_DELETED")
}

func TestUndoExpired(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")

	rec := h.deleteWork(work.ID, "山海", "1")
	deleted := decodeBody[deletePayload](t, rec)

	if err := h.tx.Exec("UPDATE undo_tokens SET expires_at = now() - interval '1 minute' WHERE token = ?", deleted.UndoToken).Error; err != nil {
		t.Fatalf("调整令牌过期时间失败：%v", err)
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	assertErrorCode(t, undo, http.StatusConflict, "UNDO_EXPIRED")
}

func TestRestoreConflictThenRename(t *testing.T) {
	h := newHarness(t)
	first := h.createWork("山海")

	rec := h.deleteWork(first.ID, "山海", "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除失败：%s", rec.Body.String())
	}

	// 同名作品重新创建后，恢复旧作品会产生冲突。
	h.createWork("山海")

	conflict := h.do(http.MethodPost, "/api/v1/trash/work/"+first.ID+"/restore", map[string]any{}, nil)
	envelope := assertErrorCode(t, conflict, http.StatusUnprocessableEntity, "RESTORE_CONFLICT")
	var details struct {
		Items []model.ConflictItem `json:"items"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil {
		t.Fatalf("解析冲突详情失败：%v", err)
	}
	if len(details.Items) != 1 || details.Items[0].ID != first.ID {
		t.Fatalf("冲突详情不符：%+v", details.Items)
	}
	if h.trashTotal() != 1 {
		t.Fatal("冲突时不应部分恢复")
	}

	restored := h.do(http.MethodPost, "/api/v1/trash/work/"+first.ID+"/restore", map[string]any{"new_name": "山海传"}, nil)
	if restored.Code != http.StatusOK {
		t.Fatalf("改名恢复失败（status=%d）：%s", restored.Code, restored.Body.String())
	}
	payload := decodeBody[restorePayload](t, restored)
	if payload.Name != "山海传" || payload.RestoredCount != 2 {
		t.Fatalf("恢复结果不符：%+v", payload)
	}

	check := h.do(http.MethodGet, "/api/v1/works/"+first.ID, nil, nil)
	if check.Code != http.StatusOK {
		t.Fatalf("恢复后作品应可见（status=%d）", check.Code)
	}
	if h.trashTotal() != 0 {
		t.Fatal("恢复后回收站应为空")
	}
}

func TestPurgeWorkRequiresConfirm(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")

	rec := h.deleteWork(work.ID, "山海", "1")
	if rec.Code != http.StatusOK {
		t.Fatalf("删除失败：%s", rec.Body.String())
	}

	bad := h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": false}, nil)
	assertErrorCode(t, bad, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	ok := h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	if ok.Code != http.StatusNoContent {
		t.Fatalf("永久删除失败（status=%d）：%s", ok.Code, ok.Body.String())
	}

	if check := h.do(http.MethodGet, "/api/v1/works/"+work.ID, nil, nil); check.Code != http.StatusNotFound {
		t.Fatalf("永久删除后作品应返回 404，实际 %d", check.Code)
	}
	if h.trashTotal() != 0 {
		t.Fatal("永久删除后回收站应为空")
	}

	var snapshots int64
	if err := h.tx.Table("snapshots").Where("reason = ?", model.SnapshotReasonTrashPurge).Count(&snapshots).Error; err != nil {
		t.Fatalf("统计快照失败：%v", err)
	}
	if snapshots < 1 {
		t.Fatal("永久删除应留下 trash_purge 快照")
	}
}

func TestPurgeAllTrash(t *testing.T) {
	h := newHarness(t)
	first := h.createWork("山海")
	second := h.createWork("星河")
	_ = h.deleteWork(first.ID, "山海", "1")
	_ = h.deleteWork(second.ID, "星河", "1")

	if h.trashTotal() != 2 {
		t.Fatalf("清空前应有 2 条，实际 %d", h.trashTotal())
	}

	bad := h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "NOT_ALL"}, nil)
	assertErrorCode(t, bad, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	rec := h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("清空失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	payload := decodeBody[purgeAllPayload](t, rec)
	if payload.Purged != 2 || payload.SnapshotID == "" {
		t.Fatalf("清空结果不符：%+v", payload)
	}
	if h.trashTotal() != 0 {
		t.Fatal("清空后回收站应为空")
	}
}

func TestSettingsCurrentWorkPatch(t *testing.T) {
	h := newHarness(t)

	clear := h.do(http.MethodPatch, "/api/v1/settings", map[string]any{"current_work_id": nil}, nil)
	if clear.Code != http.StatusOK {
		t.Fatalf("清空当前作品失败（status=%d）：%s", clear.Code, clear.Body.String())
	}
	if got := h.currentWorkID(); got != nil {
		t.Fatalf("清空后应为 null，实际 %v", got)
	}

	work := h.createWork("山海")
	set := h.do(http.MethodPatch, "/api/v1/settings", map[string]any{"current_work_id": work.ID}, nil)
	if set.Code != http.StatusOK {
		t.Fatalf("切换当前作品失败（status=%d）：%s", set.Code, set.Body.String())
	}
	if got := h.currentWorkID(); got == nil || *got != work.ID {
		t.Fatalf("切换后当前作品不符：%v", got)
	}

	missing := h.do(http.MethodPatch, "/api/v1/settings", map[string]any{"current_work_id": "01NOSUCHWORK0000000000000"}, nil)
	assertErrorCode(t, missing, http.StatusNotFound, "NOT_FOUND")
}

func TestSnapshotsEndpointReturnsRecentItems(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("山海")
	if rec := h.deleteWork(work.ID, "山海", "1"); rec.Code != http.StatusOK {
		t.Fatalf("删除失败：%s", rec.Body.String())
	}

	rec := h.do(http.MethodGet, "/api/v1/snapshots", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("读取快照失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	var items []model.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("解析快照响应失败：%v", err)
	}
	if len(items) == 0 {
		t.Fatal("删除作品后应至少有 1 条快照")
	}
	if items[0].Status != model.SnapshotStatusOK || items[0].Path == "" {
		t.Fatalf("快照字段不符：%+v", items[0])
	}
}

func TestCrossOriginWriteIsForbidden(t *testing.T) {
	h := newHarness(t)
	rec := h.do(http.MethodPost, "/api/v1/works", map[string]any{"name": "山海"}, map[string]string{
		"Origin": "https://evil.example",
	})
	assertErrorCode(t, rec, http.StatusForbidden, "FORBIDDEN_ORIGIN")
}

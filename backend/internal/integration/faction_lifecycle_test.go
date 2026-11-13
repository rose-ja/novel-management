//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestFactionDeleteMatrix(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("势力删除矩阵")
	parent := h.createFaction(work.ID, "万族", nil, nil)
	child := h.createFaction(work.ID, "妖族", &parent.ID, nil)

	rec := h.do(http.MethodDelete, "/api/v1/factions/"+parent.ID, nil, map[string]string{
		"If-Match": `"` + itoa(parent.Version) + `"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "HAS_CHILDREN")

	// 软删除下级后仍阻止删除父级。
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+child.ID, nil, map[string]string{
		"If-Match": `"` + itoa(child.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除下级失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+parent.ID, nil, map[string]string{
		"If-Match": `"` + itoa(parent.Version) + `"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "HAS_CHILDREN")

	// 活跃成员阻止删除。
	memberFaction := h.createFaction(work.ID, "有成员", nil, nil)
	h.seedFactionCharacter(work.ID, "成员甲", memberFaction.ID, false)
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+memberFaction.ID, nil, map[string]string{
		"If-Match": `"` + itoa(memberFaction.Version) + `"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
}

func TestFactionTrashUndoRestorePurge(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("势力回收站")
	faction := h.createFaction(work.ID, "妖族", nil, nil)
	created := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "faction", "owner_id": faction.ID, "text": "妖族立族",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("创建势力大事记失败（status=%d）：%s", created.Code, created.Body.String())
	}

	deleted := h.do(http.MethodDelete, "/api/v1/factions/"+faction.ID, nil, map[string]string{
		"If-Match": `"` + itoa(faction.Version) + `"`,
	})
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除势力失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	token := decodeBody[deletePayload](t, deleted).UndoToken
	if token == "" {
		t.Fatal("删除应返回 undo_token")
	}

	page := decodeBody[trashPage](t, h.do(http.MethodGet, "/api/v1/trash?root_type=faction", nil, nil))
	if page.Total != 1 || page.Items[0].RootType != "faction" || page.Items[0].RootID != faction.ID {
		t.Fatalf("回收站势力根不正确：%+v", page)
	}
	if page.Items[0].ParentName == nil || *page.Items[0].ParentName != "势力回收站" {
		t.Fatalf("顶层势力的父级应显示作品名：%+v", page.Items[0].ParentName)
	}
	if page.Items[0].AffectedCount != 2 {
		t.Fatalf("影响条数应为势力+大事记=2，实际 %d", page.Items[0].AffectedCount)
	}
	if rec := h.do(http.MethodGet, "/api/v1/factions/"+faction.ID, nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("删除后详情应为 404，实际 %d", rec.Code)
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": token}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if detail := h.getFaction(faction.ID); detail.Relations.ChronicleCount != 1 {
		t.Fatalf("撤销后大事记应恢复，实际 %d", detail.Relations.ChronicleCount)
	}

	// 回收站恢复。
	deleted = h.do(http.MethodDelete, "/api/v1/factions/"+faction.ID, nil, map[string]string{
		"If-Match": `"` + itoa(detailVersion(t, h, faction.ID)) + `"`,
	})
	if deleted.Code != http.StatusOK {
		t.Fatalf("再次删除失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	restored := h.do(http.MethodPost, "/api/v1/trash/faction/"+faction.ID+"/restore", map[string]any{}, nil)
	if restored.Code != http.StatusOK {
		t.Fatalf("恢复失败（status=%d）：%s", restored.Code, restored.Body.String())
	}
	if payload := decodeBody[restorePayload](t, restored); payload.RestoredCount != 2 {
		t.Fatalf("恢复条数应为 2，实际 %d", payload.RestoredCount)
	}

	// 永久删除：无势力行与势力大事记残留。
	deleted = h.do(http.MethodDelete, "/api/v1/factions/"+faction.ID, nil, map[string]string{
		"If-Match": `"` + itoa(detailVersion(t, h, faction.ID)) + `"`,
	})
	if deleted.Code != http.StatusOK {
		t.Fatalf("第三次删除失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	purged := h.do(http.MethodDelete, "/api/v1/trash/faction/"+faction.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	var factionsLeft, chroniclesLeft int64
	if err := h.tx.Raw("SELECT count(*) FROM factions WHERE id = ?", faction.ID).Scan(&factionsLeft).Error; err != nil {
		t.Fatalf("统计势力失败：%v", err)
	}
	if err := h.tx.Raw("SELECT count(*) FROM chronicles WHERE owner_type = 'faction' AND owner_id = ?", faction.ID).Scan(&chroniclesLeft).Error; err != nil {
		t.Fatalf("统计大事记失败：%v", err)
	}
	if factionsLeft != 0 || chroniclesLeft != 0 {
		t.Fatalf("永久删除后不应残留：factions=%d chronicles=%d", factionsLeft, chroniclesLeft)
	}

	// 已软删人物引用势力时，永久删除不触发外键违约。
	memberFaction := h.createFaction(work.ID, "有历史成员", nil, nil)
	member := h.seedFactionCharacter(work.ID, "旧成员", memberFaction.ID, false)
	rec := h.do(http.MethodDelete, "/api/v1/characters/"+member, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除成员失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+memberFaction.ID, nil, map[string]string{
		"If-Match": `"` + itoa(memberFaction.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/trash/faction/"+memberFaction.ID, map[string]any{"confirm": true}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("带历史成员的势力永久删除失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
}

func TestFactionRestoreBlockedByDeletedParent(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("父级阻断")
	parent := h.createFaction(work.ID, "上级", nil, nil)
	child := h.createFaction(work.ID, "下级", &parent.ID, nil)

	rec := h.do(http.MethodDelete, "/api/v1/factions/"+child.ID, nil, map[string]string{
		"If-Match": `"` + itoa(child.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除下级失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	// 直接软删除父级，构造「恢复时父级不在线」。
	now := time.Now().UTC()
	if err := h.tx.Exec(
		"UPDATE factions SET deleted_at = ?, deleted_root_id = id, deleted_by_action = 'faction_delete' WHERE id = ?",
		now, parent.ID,
	).Error; err != nil {
		t.Fatalf("软删除父级失败：%v", err)
	}
	restore := h.do(http.MethodPost, "/api/v1/trash/faction/"+child.ID+"/restore", map[string]any{}, nil)
	assertErrorCode(t, restore, http.StatusConflict, "PARENT_DELETED")
}

func TestWorkLifecycleCoversFactions(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("势力作品组")
	faction := h.createFaction(work.ID, "妖族", nil, nil)
	created := h.do(http.MethodPost, "/api/v1/chronicles", map[string]any{
		"owner_type": "faction", "owner_id": faction.ID, "text": "作品组势力大事记",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("创建大事记失败（status=%d）：%s", created.Code, created.Body.String())
	}

	deleted := h.deleteWork(work.ID, work.Name, itoa(work.Version))
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	if rec := h.do(http.MethodGet, "/api/v1/factions/"+faction.ID, nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("作品删除后势力应不可见，实际 %d", rec.Code)
	}

	restored := h.do(http.MethodPost, "/api/v1/trash/work/"+work.ID+"/restore", map[string]any{}, nil)
	if restored.Code != http.StatusOK {
		t.Fatalf("恢复作品失败（status=%d）：%s", restored.Code, restored.Body.String())
	}
	if detail := h.getFaction(faction.ID); detail.Relations.ChronicleCount != 1 {
		t.Fatalf("作品恢复后势力与大事记应恢复，实际 %d", detail.Relations.ChronicleCount)
	}

	work = h.getWork(work.ID)
	deleted = h.deleteWork(work.ID, work.Name, itoa(work.Version))
	if deleted.Code != http.StatusOK {
		t.Fatalf("再次删除作品失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	purged := h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除作品失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	var factionsLeft, chroniclesLeft int64
	if err := h.tx.Raw("SELECT count(*) FROM factions WHERE work_id = ?", work.ID).Scan(&factionsLeft).Error; err != nil {
		t.Fatalf("统计势力失败：%v", err)
	}
	if err := h.tx.Raw("SELECT count(*) FROM chronicles WHERE work_id = ?", work.ID).Scan(&chroniclesLeft).Error; err != nil {
		t.Fatalf("统计大事记失败：%v", err)
	}
	if factionsLeft != 0 || chroniclesLeft != 0 {
		t.Fatalf("作品永久删除后不应残留：factions=%d chronicles=%d", factionsLeft, chroniclesLeft)
	}
}

func detailVersion(t *testing.T, h *harness, id string) int {
	t.Helper()
	return h.getFaction(id).Version
}

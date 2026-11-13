//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// RV-01：人物恢复后重新引用回收站势力，永久删除必须先阻断而非 500。
func TestFactionPurgeBlockedByRestoredMember(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("恢复成员阻断")
	faction := h.createFaction(work.ID, "有历史成员", nil, nil)
	member := h.seedFactionCharacter(work.ID, "回归成员", faction.ID, false)

	rec := h.do(http.MethodDelete, "/api/v1/characters/"+member, nil, map[string]string{"If-Match": `"1"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除成员失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+faction.ID, nil, map[string]string{
		"If-Match": `"` + itoa(faction.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec = h.do(http.MethodPost, "/api/v1/trash/character/"+member+"/restore", map[string]any{}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("恢复成员失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec = h.do(http.MethodDelete, "/api/v1/trash/faction/"+faction.ID, map[string]any{"confirm": true}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	var left int64
	if err := h.tx.Raw("SELECT count(*) FROM factions WHERE id = ? AND deleted_at IS NOT NULL", faction.ID).Scan(&left).Error; err != nil {
		t.Fatalf("统计势力失败：%v", err)
	}
	if left != 1 {
		t.Fatalf("被阻断后势力应仍在回收站，实际 %d 行", left)
	}

	// 解除成员引用后可成功永久删除。
	if err := h.tx.Exec("UPDATE characters SET faction_id = NULL WHERE id = ?", member).Error; err != nil {
		t.Fatalf("解除成员引用失败：%v", err)
	}
	rec = h.do(http.MethodDelete, "/api/v1/trash/faction/"+faction.ID, map[string]any{"confirm": true}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("解除引用后永久删除失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
}

// RV-02：势力根已被作品组删除时，清空回收站应按已处理跳过而不是回滚。
func TestPurgeAllWithIndividuallyDeletedFactionRoot(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("回收站重复根")
	faction := h.createFaction(work.ID, "单独删除势力", nil, nil)
	if rec := h.do(http.MethodDelete, "/api/v1/factions/"+faction.ID, nil, map[string]string{
		"If-Match": `"` + itoa(faction.Version) + `"`,
	}); rec.Code != http.StatusOK {
		t.Fatalf("删除势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	work = h.getWork(work.ID)
	if rec := h.deleteWork(work.ID, work.Name, itoa(work.Version)); rec.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec := h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("清空回收站失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	var left int64
	if err := h.tx.Raw("SELECT count(*) FROM factions WHERE id = ?", faction.ID).Scan(&left).Error; err != nil {
		t.Fatalf("统计势力失败：%v", err)
	}
	if left != 0 {
		t.Fatalf("清空回收站后势力应删除，实际 %d 行", left)
	}
}

// RV-04：首领在回收站时详情需标记不可用；仅改其他字段不提交首领，显式设置仍被拒绝。
func TestFactionLeaderInTrashMarked(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("回收站首领")
	leader := h.seedCharacter(work.ID, "已删首领")
	faction := h.createFaction(work.ID, "首领势力", nil, &leader)

	// 正常删除会被首领引用阻断；这里直接软删，模拟导入/历史数据中的「首领在回收站」。
	if err := h.tx.Exec(
		"UPDATE characters SET deleted_at = ?, deleted_root_id = id WHERE id = ?",
		time.Now().UTC(), leader,
	).Error; err != nil {
		t.Fatalf("软删除首领失败：%v", err)
	}

	detail := h.getFaction(faction.ID)
	if detail.Leader == nil || !detail.Leader.InTrash {
		t.Fatalf("首领应标记为回收站：%+v", detail.Leader)
	}

	updated := h.patchFaction(faction.ID, `"`+itoa(detail.Version)+`"`, map[string]any{"name": "改名势力"})
	if updated.Leader == nil || !updated.Leader.InTrash {
		t.Fatalf("未提交首领字段时应保留回收站首领：%+v", updated.Leader)
	}

	rec := h.do(http.MethodPatch, "/api/v1/factions/"+faction.ID, map[string]any{
		"leader_character_id": leader,
	}, map[string]string{"If-Match": `"` + itoa(updated.Version) + `"`})
	assertErrorCode(t, rec, http.StatusConflict, "PARENT_DELETED")
}

// RV-05：被软删下级阻断父级删除时，冲突项需标记回收站。
func TestFactionDeleteConflictMarksTrashedChild(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("回收站下级阻断")
	parent := h.createFaction(work.ID, "上级", nil, nil)
	child := h.createFaction(work.ID, "下级", &parent.ID, nil)
	if rec := h.do(http.MethodDelete, "/api/v1/factions/"+child.ID, nil, map[string]string{
		"If-Match": `"` + itoa(child.Version) + `"`,
	}); rec.Code != http.StatusOK {
		t.Fatalf("删除下级失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec := h.do(http.MethodDelete, "/api/v1/factions/"+parent.ID, nil, map[string]string{
		"If-Match": `"` + itoa(parent.Version) + `"`,
	})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "HAS_CHILDREN")
	var details struct {
		Items []struct {
			ID      string `json:"id"`
			InTrash bool   `json:"in_trash"`
		} `json:"items"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil {
		t.Fatalf("解析 details 失败：%v", err)
	}
	if len(details.Items) != 1 || details.Items[0].ID != child.ID || !details.Items[0].InTrash {
		t.Fatalf("冲突项应标记回收站：%+v", details.Items)
	}
}

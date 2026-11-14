//go:build integration

package integration

import (
	"net/http"
	"strconv"
	"testing"
)

func (h *harness) trashRankings() []struct {
	RootType string `json:"root_type"`
	RootID   string `json:"root_id"`
	Name     string `json:"name"`
} {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/trash?root_type=ranking", nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取榜单回收站失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	page := decodeBody[struct {
		Items []struct {
			RootType string `json:"root_type"`
			RootID   string `json:"root_id"`
			Name     string `json:"name"`
		} `json:"items"`
	}](h.t, rec)
	return page.Items
}

func TestRankingDeleteUndoAndRestore(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("榜单回收站")
	ranking := h.createRanking(work.ID, "天榜", 10)
	alpha := h.createCharacter(work.ID, "甲", nil, nil)
	seat1 := h.rankingSeat(ranking.ID, 1)
	if _, apiErr := h.putSeat(ranking.ID, 1, seat1.Version, map[string]any{"character_id": alpha.ID}); apiErr != nil {
		t.Fatalf("指派失败：%+v", *apiErr)
	}

	rec := h.do(http.MethodDelete, "/api/v1/rankings/"+ranking.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(ranking.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除榜单失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	deleted := decodeBody[rankingPayload](t, rec)
	if deleted.UndoToken == "" {
		t.Fatal("删除榜单应返回 undo_token")
	}
	if items := h.trashRankings(); len(items) != 1 || items[0].RootID != ranking.ID {
		t.Fatalf("回收站应包含榜单：%+v", items)
	}

	// 撤销。
	undoRec := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": deleted.UndoToken}, nil)
	if undoRec.Code != http.StatusOK {
		t.Fatalf("撤销失败（status=%d）：%s", undoRec.Code, undoRec.Body.String())
	}
	restored := h.getRanking(ranking.ID)
	if restored.Seats.Items[0].Character == nil || restored.Seats.Items[0].Character.ID != alpha.ID {
		t.Fatalf("撤销后席位应保留：%+v", restored.Seats.Items[0])
	}

	// 再次删除并从回收站恢复。
	deletedAgain := h.do(http.MethodDelete, "/api/v1/rankings/"+ranking.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(restored.Version) + `"`,
	})
	if deletedAgain.Code != http.StatusOK {
		t.Fatalf("再次删除失败：%s", deletedAgain.Body.String())
	}
	restoreRec := h.do(http.MethodPost, "/api/v1/trash/ranking/"+ranking.ID+"/restore", map[string]any{}, nil)
	if restoreRec.Code != http.StatusOK {
		t.Fatalf("回收站恢复失败（status=%d）：%s", restoreRec.Code, restoreRec.Body.String())
	}
	h.getRanking(ranking.ID)
}

func TestRankingRestoreConflictAndMissingCharacter(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("榜单恢复校验")
	ranking := h.createRanking(work.ID, "天榜", 10)
	alpha := h.createCharacter(work.ID, "甲", nil, nil)
	seat1 := h.rankingSeat(ranking.ID, 1)
	if _, apiErr := h.putSeat(ranking.ID, 1, seat1.Version, map[string]any{"character_id": alpha.ID}); apiErr != nil {
		t.Fatalf("指派失败：%+v", *apiErr)
	}

	rec := h.do(http.MethodDelete, "/api/v1/rankings/"+ranking.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(ranking.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除榜单失败：%s", rec.Body.String())
	}

	// 同名活跃榜单导致恢复冲突。
	h.createRanking(work.ID, "天榜", 5)
	rec = h.do(http.MethodPost, "/api/v1/trash/ranking/"+ranking.ID+"/restore", map[string]any{}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "RESTORE_CONFLICT")

	// 模拟席位人物已被删除：恢复应返回 REFERENCE_MISSING。
	if err := h.tx.Exec("UPDATE characters SET deleted_at = now(), deleted_root_id = id WHERE id = ?", alpha.ID).Error; err != nil {
		t.Fatalf("模拟人物删除失败：%v", err)
	}
	if err := h.tx.Exec("UPDATE rankings SET name = '天榜旧', name_key = '天榜旧' WHERE id = ?", ranking.ID).Error; err != nil {
		t.Fatalf("改名失败：%v", err)
	}
	rec = h.do(http.MethodPost, "/api/v1/trash/ranking/"+ranking.ID+"/restore", map[string]any{}, nil)
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_MISSING")
	if len(envelope.Error.Details) == 0 {
		t.Fatal("REFERENCE_MISSING 应包含 details")
	}
}

func TestRankingPurgeAndWorkLifecycle(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("榜单作品生命周期")
	ranking := h.createRanking(work.ID, "天榜", 10)
	alpha := h.createCharacter(work.ID, "甲", nil, nil)
	seat1 := h.rankingSeat(ranking.ID, 1)
	if _, apiErr := h.putSeat(ranking.ID, 1, seat1.Version, map[string]any{"character_id": alpha.ID}); apiErr != nil {
		t.Fatalf("指派失败：%+v", *apiErr)
	}

	// 作品整组删除后榜单随作品隐藏，恢复后重新可见。
	rec := h.deleteWork(work.ID, work.Name, strconv.Itoa(work.Version))
	if rec.Code != http.StatusOK {
		t.Fatalf("删除作品失败：%s", rec.Body.String())
	}
	restoreWork := h.do(http.MethodPost, "/api/v1/trash/work/"+work.ID+"/restore", map[string]any{}, nil)
	if restoreWork.Code != http.StatusOK {
		t.Fatalf("恢复作品失败：%s", restoreWork.Body.String())
	}
	restored := h.getRanking(ranking.ID)
	if restored.Seats.Items[0].Character == nil {
		t.Fatalf("作品恢复后席位应保留：%+v", restored.Seats.Items[0])
	}

	// 删除榜单后永久删除，席位行一并清理。
	del := h.do(http.MethodDelete, "/api/v1/rankings/"+ranking.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(restored.Version) + `"`,
	})
	if del.Code != http.StatusOK {
		t.Fatalf("删除榜单失败：%s", del.Body.String())
	}
	purge := h.do(http.MethodDelete, "/api/v1/trash/ranking/"+ranking.ID, map[string]any{"confirm": true}, nil)
	if purge.Code != http.StatusNoContent {
		t.Fatalf("永久删除榜单失败（status=%d）：%s", purge.Code, purge.Body.String())
	}
	var seats int64
	if err := h.tx.Raw("SELECT count(*) FROM ranking_seats WHERE ranking_id = ?", ranking.ID).Scan(&seats).Error; err != nil {
		t.Fatalf("统计席位失败：%v", err)
	}
	if seats != 0 {
		t.Fatalf("永久删除后席位应清空，实际 %d", seats)
	}

	// 再次整组删除作品并永久删除，验证席位 FK 不阻断。
	currentWork := h.getWork(work.ID)
	rec = h.deleteWork(work.ID, work.Name, strconv.Itoa(currentWork.Version))
	if rec.Code != http.StatusOK {
		t.Fatalf("再次删除作品失败：%s", rec.Body.String())
	}
	purgeWork := h.do(http.MethodDelete, "/api/v1/trash/work/"+work.ID, map[string]any{"confirm": true}, nil)
	if purgeWork.Code != http.StatusNoContent {
		t.Fatalf("永久删除作品失败（status=%d）：%s", purgeWork.Code, purgeWork.Body.String())
	}

	// 单独删除榜单后再删除作品，清空回收站不应因榜单根已被作品组删除而失败。
	secondWork := h.createWork("榜单清空")
	secondRanking := h.createRanking(secondWork.ID, "清空榜", 5)
	if del := h.do(http.MethodDelete, "/api/v1/rankings/"+secondRanking.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(secondRanking.Version) + `"`,
	}); del.Code != http.StatusOK {
		t.Fatalf("删除第二榜单失败：%s", del.Body.String())
	}
	if delWork := h.deleteWork(secondWork.ID, secondWork.Name, strconv.Itoa(secondWork.Version)); delWork.Code != http.StatusOK {
		t.Fatalf("删除第二作品失败：%s", delWork.Body.String())
	}
	purgeAll := h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	if purgeAll.Code != http.StatusOK {
		t.Fatalf("清空回收站失败（status=%d）：%s", purgeAll.Code, purgeAll.Body.String())
	}
}

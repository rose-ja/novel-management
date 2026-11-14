//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestLocationTrashUndoRestorePurge(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点回收站")
	location := h.createLocation(work.ID, "东荒", nil, strPtr("星域"))

	deleted := h.do(http.MethodDelete, "/api/v1/locations/"+location.ID, nil, map[string]string{
		"If-Match": `"` + itoa(location.Version) + `"`,
	})
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除地点失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	token := decodeBody[deletePayload](t, deleted).UndoToken
	if token == "" {
		t.Fatal("删除应返回 undo_token")
	}

	page := decodeBody[trashPage](t, h.do(http.MethodGet, "/api/v1/trash?root_type=location", nil, nil))
	if page.Total != 1 || page.Items[0].RootType != "location" || page.Items[0].RootID != location.ID {
		t.Fatalf("回收站地点根不正确：%+v", page)
	}
	if page.Items[0].ParentName == nil || *page.Items[0].ParentName != "地点回收站" {
		t.Fatalf("顶层地点父级应显示作品名：%+v", page.Items[0].ParentName)
	}

	undo := h.do(http.MethodPost, "/api/v1/trash/undo", map[string]any{"undo_token": token}, nil)
	if undo.Code != http.StatusOK {
		t.Fatalf("撤销失败（status=%d）：%s", undo.Code, undo.Body.String())
	}
	if detail := h.getLocation(location.ID); detail.Name != "东荒" {
		t.Fatalf("撤销后地点应恢复：%+v", detail)
	}

	// 回收站恢复。
	deleted = h.do(http.MethodDelete, "/api/v1/locations/"+location.ID, nil, map[string]string{
		"If-Match": `"` + itoa(h.getLocation(location.ID).Version) + `"`,
	})
	if deleted.Code != http.StatusOK {
		t.Fatalf("再次删除失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	restored := h.do(http.MethodPost, "/api/v1/trash/location/"+location.ID+"/restore", map[string]any{}, nil)
	if restored.Code != http.StatusOK {
		t.Fatalf("恢复失败（status=%d）：%s", restored.Code, restored.Body.String())
	}

	// 永久删除。
	deleted = h.do(http.MethodDelete, "/api/v1/locations/"+location.ID, nil, map[string]string{
		"If-Match": `"` + itoa(h.getLocation(location.ID).Version) + `"`,
	})
	if deleted.Code != http.StatusOK {
		t.Fatalf("第三次删除失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	purged := h.do(http.MethodDelete, "/api/v1/trash/location/"+location.ID, map[string]any{"confirm": true}, nil)
	if purged.Code != http.StatusNoContent {
		t.Fatalf("永久删除失败（status=%d）：%s", purged.Code, purged.Body.String())
	}
	var left int64
	if err := h.tx.Raw("SELECT count(*) FROM locations WHERE id = ?", location.ID).Scan(&left).Error; err != nil {
		t.Fatalf("统计地点失败：%v", err)
	}
	if left != 0 {
		t.Fatalf("永久删除后地点仍有残留：%d", left)
	}
}

func TestLocationRestoreBlockedByDeletedParent(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点父级阻断")
	parent := h.createLocation(work.ID, "东荒", nil, nil)
	child := h.createLocation(work.ID, "青丘山", &parent.ID, nil)

	rec := h.do(http.MethodDelete, "/api/v1/locations/"+child.ID, nil, map[string]string{
		"If-Match": `"` + itoa(child.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除下级失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	now := time.Now().UTC()
	if err := h.tx.Exec(
		"UPDATE locations SET deleted_at = ?, deleted_root_id = id, deleted_by_action = 'location_delete' WHERE id = ?",
		now, parent.ID,
	).Error; err != nil {
		t.Fatalf("软删除父级失败：%v", err)
	}
	restore := h.do(http.MethodPost, "/api/v1/trash/location/"+child.ID+"/restore", map[string]any{}, nil)
	assertErrorCode(t, restore, http.StatusConflict, "PARENT_DELETED")
}

func TestWorkLifecycleCoversLocations(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点作品组")
	location := h.createLocation(work.ID, "东荒", nil, nil)
	child := h.createLocation(work.ID, "青丘山", &location.ID, nil)
	faction := h.createFaction(work.ID, "青丘狐族", nil, nil)

	rec := h.do(http.MethodPost, "/api/v1/locations/"+location.ID+"/factions", map[string]any{
		"faction_id": faction.ID, "note": "狐族聚居",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建地点-势力关系失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	rec = h.do(http.MethodPost, "/api/v1/characters", map[string]any{
		"work_id":        work.ID,
		"name":           "涂山氏",
		"location_links": []map[string]any{{"location_id": child.ID, "note": "出生地"}},
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建关联人物失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	character := decodeBody[characterPayload](t, rec)

	deleted := h.deleteWork(work.ID, work.Name, itoa(work.Version))
	if deleted.Code != http.StatusOK {
		t.Fatalf("删除作品失败（status=%d）：%s", deleted.Code, deleted.Body.String())
	}
	if rec := h.do(http.MethodGet, "/api/v1/locations/"+location.ID, nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("作品删除后地点应不可见，实际 %d", rec.Code)
	}
	if rec := h.do(http.MethodGet, "/api/v1/characters/"+character.ID, nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("作品删除后人物应不可见，实际 %d", rec.Code)
	}

	restored := h.do(http.MethodPost, "/api/v1/trash/work/"+work.ID+"/restore", map[string]any{}, nil)
	if restored.Code != http.StatusOK {
		t.Fatalf("恢复作品失败（status=%d）：%s", restored.Code, restored.Body.String())
	}
	detail := h.getLocation(location.ID)
	if len(detail.Relations.Factions) != 1 {
		t.Fatalf("作品恢复后地点势力关系应完整：%+v", detail.Relations)
	}
	if childDetail := h.getLocation(child.ID); len(childDetail.Relations.Characters) != 1 {
		t.Fatalf("作品恢复后地点人物关系应完整：%+v", childDetail.Relations)
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
	var locationsLeft, factionsLeft, charactersLeft, linksLeft, charLinksLeft int64
	for query, target := range map[string]*int64{
		"SELECT count(*) FROM locations WHERE work_id = ?":           &locationsLeft,
		"SELECT count(*) FROM factions WHERE work_id = ?":            &factionsLeft,
		"SELECT count(*) FROM characters WHERE work_id = ?":          &charactersLeft,
		"SELECT count(*) FROM location_factions WHERE work_id = ?":   &linksLeft,
		"SELECT count(*) FROM character_locations WHERE work_id = ?": &charLinksLeft,
	} {
		if err := h.tx.Raw(query, work.ID).Scan(target).Error; err != nil {
			t.Fatalf("统计失败（%s）：%v", query, err)
		}
	}
	if locationsLeft+factionsLeft+charactersLeft+linksLeft+charLinksLeft != 0 {
		t.Fatalf("作品永久删除后仍有残留：locations=%d factions=%d characters=%d links=%d charLinks=%d",
			locationsLeft, factionsLeft, charactersLeft, linksLeft, charLinksLeft)
	}
}

func TestPurgeAllIncludesLocations(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("清空回收站地点")
	location := h.createLocation(work.ID, "待清空地点", nil, nil)
	if rec := h.do(http.MethodDelete, "/api/v1/locations/"+location.ID, nil, map[string]string{
		"If-Match": `"` + itoa(location.Version) + `"`,
	}); rec.Code != http.StatusOK {
		t.Fatalf("删除地点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}

	rec := h.do(http.MethodDelete, "/api/v1/trash", map[string]any{"confirm": "PURGE_ALL"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("清空回收站失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if payload := decodeBody[purgeAllPayload](t, rec); payload.Purged < 1 {
		t.Fatalf("清空回收站应包含地点根，实际 purged=%d", payload.Purged)
	}
	var left int64
	if err := h.tx.Raw("SELECT count(*) FROM locations WHERE id = ?", location.ID).Scan(&left).Error; err != nil {
		t.Fatalf("统计地点失败：%v", err)
	}
	if left != 0 {
		t.Fatalf("清空回收站后地点应删除，实际 %d 行", left)
	}
}

//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"

	"novel-management/backend/internal/model"
)

type memberPage struct {
	Items []struct {
		ID      string        `json:"id"`
		Name    string        `json:"name"`
		Faction *factionBrief `json:"faction"`
	} `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func (h *harness) seedFactionCharacter(workID, name, factionID string, deleted bool) string {
	h.t.Helper()
	id := model.NewID()
	now := time.Now().UTC()
	if deleted {
		if err := h.tx.Exec(
			"INSERT INTO characters (id, work_id, name, name_key, faction_id, deleted_at, deleted_root_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, workID, name, name, factionID, now, id,
		).Error; err != nil {
			h.t.Fatalf("播种已删人物失败：%v", err)
		}
		return id
	}
	if err := h.tx.Exec(
		"INSERT INTO characters (id, work_id, name, name_key, faction_id) VALUES (?, ?, ?, ?, ?)",
		id, workID, name, name, factionID,
	).Error; err != nil {
		h.t.Fatalf("播种人物失败：%v", err)
	}
	return id
}

func (h *harness) listMembers(factionID, scope, query string) memberPage {
	h.t.Helper()
	path := "/api/v1/factions/" + factionID + "/members"
	if scope != "" {
		path += "?scope=" + scope
	}
	if query != "" {
		if scope == "" {
			path += "?"
		} else {
			path += "&"
		}
		path += query
	}
	rec := h.do(http.MethodGet, path, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取成员失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[memberPage](h.t, rec)
}

func TestFactionMembersScopeAndCounts(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("成员作品")
	parent := h.createFaction(work.ID, "万族", nil, nil)
	child := h.createFaction(work.ID, "妖族", &parent.ID, nil)

	h.seedFactionCharacter(work.ID, "甲", parent.ID, false)
	h.seedFactionCharacter(work.ID, "乙", parent.ID, false)
	h.seedFactionCharacter(work.ID, "丙", child.ID, false)
	h.seedFactionCharacter(work.ID, "丁", parent.ID, true)

	direct := h.listMembers(parent.ID, "direct", "")
	if direct.Total != 2 || len(direct.Items) != 2 {
		t.Fatalf("直属成员应为 2，实际 total=%d len=%d", direct.Total, len(direct.Items))
	}
	for _, item := range direct.Items {
		if item.Faction == nil || item.Faction.ID != parent.ID || item.Faction.Name != "万族" {
			t.Fatalf("成员应带真实势力摘要：%+v", item.Faction)
		}
	}

	subtree := h.listMembers(parent.ID, "subtree", "")
	if subtree.Total != 3 || len(subtree.Items) != 3 {
		t.Fatalf("含下级成员应为 3，实际 total=%d len=%d", subtree.Total, len(subtree.Items))
	}

	paged := h.listMembers(parent.ID, "subtree", "page_size=2&page=2")
	if paged.Total != 3 || len(paged.Items) != 1 || paged.Page != 2 || paged.PageSize != 2 {
		t.Fatalf("分页不正确：%+v", paged)
	}

	detail := h.getFaction(parent.ID)
	if detail.MemberCount != 2 || detail.ChildrenCount != 1 {
		t.Fatalf("直属/下级计数不正确：%+v", detail)
	}
	if detail.Relations.SubtreeMemberCount != 3 {
		t.Fatalf("含下级成员数应为 3，实际 %d", detail.Relations.SubtreeMemberCount)
	}

	// 人物换势力后两侧成员即时变化。
	if err := h.tx.Exec("UPDATE characters SET faction_id = ? WHERE work_id = ? AND name = ?", parent.ID, work.ID, "丙").Error; err != nil {
		t.Fatalf("更新人物势力失败：%v", err)
	}
	if got := h.listMembers(parent.ID, "direct", "").Total; got != 3 {
		t.Fatalf("换势力后直属成员应为 3，实际 %d", got)
	}
	if got := h.listMembers(child.ID, "subtree", "").Total; got != 0 {
		t.Fatalf("换势力后原子势力成员应为 0，实际 %d", got)
	}

	rec := h.do(http.MethodGet, "/api/v1/factions/"+parent.ID+"/members?scope=bad", nil, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
}

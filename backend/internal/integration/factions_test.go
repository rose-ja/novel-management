//go:build integration

package integration

import (
	"net/http"
	"strconv"
	"testing"

	"novel-management/backend/internal/model"
)

func itoa(value int) string { return strconv.Itoa(value) }

type factionBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type factionPayload struct {
	ID            string        `json:"id"`
	WorkID        string        `json:"work_id"`
	Name          string        `json:"name"`
	Parent        *factionBrief `json:"parent"`
	Leader        *factionBrief `json:"leader"`
	Description   *string       `json:"description"`
	ReviewStatus  string        `json:"review_status"`
	ChildrenCount int64         `json:"children_count"`
	MemberCount   int64         `json:"member_count"`
	Version       int           `json:"version"`
	Relations     struct {
		Children  []factionBrief `json:"children"`
		Locations []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"locations"`
		ChronicleCount     int64 `json:"chronicle_count"`
		SubtreeMemberCount int64 `json:"subtree_member_count"`
	} `json:"relations"`
}

type factionList struct {
	Items []factionPayload `json:"items"`
	Total int              `json:"total"`
}

func (h *harness) createFaction(workID, name string, parentID, leaderID *string) factionPayload {
	h.t.Helper()
	body := map[string]any{"work_id": workID, "name": name}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	if leaderID != nil {
		body["leader_character_id"] = *leaderID
	}
	rec := h.do(http.MethodPost, "/api/v1/factions", body, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location != "/api/v1/factions/"+decodeBody[factionPayload](h.t, rec).ID {
		h.t.Fatalf("Location 头不正确：%q", location)
	}
	return decodeBody[factionPayload](h.t, rec)
}

func (h *harness) getFaction(id string) factionPayload {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/factions/"+id, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[factionPayload](h.t, rec)
}

func (h *harness) patchFaction(id, ifMatch string, body map[string]any) factionPayload {
	h.t.Helper()
	rec := h.do(http.MethodPatch, "/api/v1/factions/"+id, body, map[string]string{"If-Match": ifMatch})
	if rec.Code != http.StatusOK {
		h.t.Fatalf("更新势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[factionPayload](h.t, rec)
}

func (h *harness) seedCharacter(workID, name string) string {
	h.t.Helper()
	id := model.NewID()
	if err := h.tx.Exec(
		"INSERT INTO characters (id, work_id, name, name_key) VALUES (?, ?, ?, ?)",
		id, workID, name, name,
	).Error; err != nil {
		h.t.Fatalf("播种人物失败：%v", err)
	}
	return id
}

func TestFactionTreeCRUD(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("势力作品")

	wan := h.createFaction(work.ID, "万族", nil, nil)
	yao := h.createFaction(work.ID, "妖族", &wan.ID, nil)
	qingqiu := h.createFaction(work.ID, "青丘", &yao.ID, nil)
	h.createFaction(work.ID, "人族", nil, nil)

	if wan.Parent != nil {
		t.Fatalf("顶层势力的 parent 应为 null：%+v", wan.Parent)
	}
	if yao.Parent == nil || yao.Parent.ID != wan.ID {
		t.Fatalf("妖族的上级应为万族：%+v", yao.Parent)
	}
	if qingqiu.Parent == nil || qingqiu.Parent.ID != yao.ID {
		t.Fatalf("青丘的上级应为妖族：%+v", qingqiu.Parent)
	}
	if got := h.getFaction(wan.ID).ChildrenCount; got != 1 {
		t.Fatalf("万族的直接下级应为 1，实际 %d", got)
	}

	list := decodeBody[factionList](t, h.do(http.MethodGet, "/api/v1/factions?work_id="+work.ID, nil, nil))
	if list.Total != 4 {
		t.Fatalf("势力总数应为 4，实际 %d", list.Total)
	}
	var totalChildren int64
	for _, item := range list.Items {
		totalChildren += item.ChildrenCount
	}
	if totalChildren != 2 {
		t.Fatalf("计数合计应为 2，实际 %d", totalChildren)
	}

	// q 命中青丘时补齐祖先，但不返回无关的人族。
	search := decodeBody[factionList](t, h.do(http.MethodGet, "/api/v1/factions?work_id="+work.ID+"&q=青丘", nil, nil))
	names := make(map[string]bool, len(search.Items))
	for _, item := range search.Items {
		names[item.Name] = true
	}
	if !names["青丘"] || !names["妖族"] || !names["万族"] || names["人族"] {
		t.Fatalf("搜索应返回命中与祖先且不含无关节点：%+v", names)
	}

	// 父级筛选 root 只返回顶层。
	roots := decodeBody[factionList](t, h.do(http.MethodGet, "/api/v1/factions?work_id="+work.ID+"&parent_id=root", nil, nil))
	if roots.Total != 2 {
		t.Fatalf("顶层势力应为 2，实际 %d", roots.Total)
	}
}

func TestFactionNameAndReferenceValidation(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("校验作品")
	other := h.createWork("其他作品")

	h.createFaction(work.ID, "Sword", nil, nil)
	rec := h.do(http.MethodPost, "/api/v1/factions", map[string]any{"work_id": work.ID, "name": "sword"}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "NAME_CONFLICT")

	foreign := h.createFaction(other.ID, "外族", nil, nil)
	rec = h.do(http.MethodPost, "/api/v1/factions", map[string]any{
		"work_id": work.ID, "name": "跨界", "parent_id": foreign.ID,
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "CROSS_WORK_REFERENCE")

	leaderID := h.seedCharacter(work.ID, "首领甲")
	leader := h.createFaction(work.ID, "有首领", nil, &leaderID)
	if leader.Leader == nil || leader.Leader.ID != leaderID || leader.Leader.Name != "首领甲" {
		t.Fatalf("首领摘要不正确：%+v", leader.Leader)
	}
	otherLeader := h.seedCharacter(other.ID, "他族首领")
	rec = h.do(http.MethodPost, "/api/v1/factions", map[string]any{
		"work_id": work.ID, "name": "错首领", "leader_character_id": otherLeader,
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "CROSS_WORK_REFERENCE")

	// 名称规范化为 lower(trim)。
	added := h.createFaction(work.ID, "  灵族  ", nil, nil)
	if added.Name != "灵族" {
		t.Fatalf("名称应去除首尾空白，实际 %q", added.Name)
	}
}

func TestFactionMoveCycleAndVersion(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("移动作品")
	a := h.createFaction(work.ID, "甲", nil, nil)
	b := h.createFaction(work.ID, "乙", &a.ID, nil)
	c := h.createFaction(work.ID, "丙", &b.ID, nil)
	d := h.createFaction(work.ID, "丁", nil, nil)

	// 缺少 If-Match → 428。
	rec := h.do(http.MethodPatch, "/api/v1/factions/"+a.ID, map[string]any{"name": "甲改"}, nil)
	assertErrorCode(t, rec, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")

	for _, parentID := range []string{a.ID, b.ID, c.ID} {
		rec = h.do(http.MethodPatch, "/api/v1/factions/"+a.ID, map[string]any{"parent_id": parentID}, map[string]string{
			"If-Match": `"` + itoa(a.Version) + `"`,
		})
		assertErrorCode(t, rec, http.StatusUnprocessableEntity, "CYCLE_DETECTED")
	}

	moved := h.patchFaction(a.ID, `"`+itoa(a.Version)+`"`, map[string]any{"parent_id": d.ID})
	if moved.Parent == nil || moved.Parent.ID != d.ID || moved.Version != a.Version+1 {
		t.Fatalf("合法移动失败：%+v", moved)
	}

	rec = h.do(http.MethodPatch, "/api/v1/factions/"+a.ID, map[string]any{"name": "过期"}, map[string]string{
		"If-Match": `"` + itoa(a.Version) + `"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestFactionDeleteAndDetail(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("删除作品")
	parent := h.createFaction(work.ID, "上级", nil, nil)
	child := h.createFaction(work.ID, "下级", &parent.ID, nil)

	detail := h.getFaction(parent.ID)
	if len(detail.Relations.Children) != 1 || detail.Relations.Children[0].ID != child.ID {
		t.Fatalf("详情应包含下级：%+v", detail.Relations.Children)
	}
	if detail.Relations.Locations == nil || len(detail.Relations.Locations) != 0 {
		t.Fatalf("地点关系本轮应为空数组：%+v", detail.Relations.Locations)
	}

	// 有下级时删除被阻止。
	rec := h.do(http.MethodDelete, "/api/v1/factions/"+parent.ID, nil, map[string]string{
		"If-Match": `"` + itoa(parent.Version) + `"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "HAS_CHILDREN")

	leaf := h.createFaction(work.ID, "叶子", nil, nil)
	rec = h.do(http.MethodDelete, "/api/v1/factions/"+leaf.ID, nil, map[string]string{
		"If-Match": `"` + itoa(leaf.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删除无引用势力失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	deleted := decodeBody[deletePayload](t, rec)
	if deleted.UndoToken == "" {
		t.Fatal("删除应返回 undo_token")
	}
}

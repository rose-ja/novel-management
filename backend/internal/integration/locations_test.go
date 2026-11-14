//go:build integration

package integration

import (
	"net/http"
	"testing"
)

type locationBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type locationPayload struct {
	ID              string         `json:"id"`
	WorkID          string         `json:"work_id"`
	Name            string         `json:"name"`
	Parent          *locationBrief `json:"parent"`
	TypeText        *string        `json:"type_text"`
	Description     *string        `json:"description"`
	ReviewStatus    string         `json:"review_status"`
	ChildrenCount   int64          `json:"children_count"`
	CharactersCount int64          `json:"characters_count"`
	FactionsCount   int64          `json:"factions_count"`
	Version         int            `json:"version"`
	Relations       struct {
		Children []struct {
			ID              string  `json:"id"`
			Name            string  `json:"name"`
			TypeText        *string `json:"type_text"`
			CharactersCount int64   `json:"characters_count"`
			FactionsCount   int64   `json:"factions_count"`
		} `json:"children"`
		Characters []struct {
			CharacterID string  `json:"character_id"`
			Name        string  `json:"name"`
			Note        *string `json:"note"`
			InTrash     bool    `json:"in_trash"`
		} `json:"characters"`
		Factions []struct {
			FactionID string  `json:"faction_id"`
			Name      string  `json:"name"`
			Note      *string `json:"note"`
			InTrash   bool    `json:"in_trash"`
		} `json:"factions"`
	} `json:"relations"`
}

type locationList struct {
	Items []locationPayload `json:"items"`
	Total int               `json:"total"`
}

func (h *harness) createLocation(workID, name string, parentID *string, typeText *string) locationPayload {
	h.t.Helper()
	body := map[string]any{"work_id": workID, "name": name}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	if typeText != nil {
		body["type_text"] = *typeText
	}
	rec := h.do(http.MethodPost, "/api/v1/locations", body, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建地点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[locationPayload](h.t, rec)
}

func (h *harness) getLocation(id string) locationPayload {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/locations/"+id, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取地点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[locationPayload](h.t, rec)
}

func (h *harness) patchLocation(id, ifMatch string, body map[string]any) locationPayload {
	h.t.Helper()
	rec := h.do(http.MethodPatch, "/api/v1/locations/"+id, body, map[string]string{"If-Match": ifMatch})
	if rec.Code != http.StatusOK {
		h.t.Fatalf("更新地点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[locationPayload](h.t, rec)
}

func (h *harness) listLocations(query string) locationList {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/locations?"+query, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取地点列表失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[locationList](h.t, rec)
}

func TestLocationTreeCRUD(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点作品")

	xingyu := "星域"
	xingcheng := "星城"
	mijing := "秘境"
	donghuang := h.createLocation(work.ID, "东荒", nil, &xingyu)
	cangwu := h.createLocation(work.ID, "苍梧星城", &donghuang.ID, &xingcheng)
	guixu := h.createLocation(work.ID, "归墟秘境", &cangwu.ID, &mijing)

	if donghuang.Parent != nil {
		t.Fatalf("顶层地点 parent 应为 null：%+v", donghuang.Parent)
	}
	if cangwu.Parent == nil || cangwu.Parent.ID != donghuang.ID {
		t.Fatalf("苍梧星城上级应为东荒：%+v", cangwu.Parent)
	}
	if guixu.Parent == nil || guixu.Parent.ID != cangwu.ID {
		t.Fatalf("归墟秘境上级应为苍梧星城：%+v", guixu.Parent)
	}
	if got := h.getLocation(donghuang.ID).ChildrenCount; got != 1 {
		t.Fatalf("东荒直接下级应为 1，实际 %d", got)
	}

	list := h.listLocations("work_id=" + work.ID)
	if list.Total != 3 {
		t.Fatalf("地点总数应为 3，实际 %d", list.Total)
	}

	// 重名（大小写等价）拒绝。
	rec := h.do(http.MethodPost, "/api/v1/locations", map[string]any{
		"work_id": work.ID, "name": "  东荒 ",
	}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("重名应返回 409，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 搜索命中补齐祖先。
	hit := h.listLocations("work_id=" + work.ID + "&q=归墟")
	if hit.Total != 3 {
		t.Fatalf("搜索命中应补祖先共 3 个，实际 %d", hit.Total)
	}
	// 类型筛选保留祖先链。
	typed := h.listLocations("work_id=" + work.ID + "&type_text=" + "秘境")
	if typed.Total != 3 {
		t.Fatalf("类型筛选应保留祖先共 3 个，实际 %d", typed.Total)
	}
}

func TestLocationParentValidation(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点父级作品")
	other := h.createWork("另一作品")
	root := h.createLocation(work.ID, "东荒", nil, nil)
	child := h.createLocation(work.ID, "苍梧星城", &root.ID, nil)
	foreign := h.createLocation(other.ID, "西洲", nil, nil)

	// 跨作品父级。
	rec := h.do(http.MethodPost, "/api/v1/locations", map[string]any{
		"work_id": work.ID, "name": "越界", "parent_id": foreign.ID,
	}, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("跨作品父级应返回 422，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 移动到自身。
	rec = h.do(http.MethodPatch, "/api/v1/locations/"+root.ID, map[string]any{
		"parent_id": root.ID,
	}, map[string]string{"If-Match": itoa(root.Version)})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("移动到自身应返回 422，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 移动到后代。
	rec = h.do(http.MethodPatch, "/api/v1/locations/"+root.ID, map[string]any{
		"parent_id": child.ID,
	}, map[string]string{"If-Match": itoa(root.Version)})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("移动到后代应返回 422，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 缺少 If-Match。
	rec = h.do(http.MethodPatch, "/api/v1/locations/"+child.ID, map[string]any{"name": "改名"}, nil)
	if rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("缺少 If-Match 应返回 428，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 过期版本。
	rec = h.do(http.MethodPatch, "/api/v1/locations/"+child.ID, map[string]any{"name": "改名"},
		map[string]string{"If-Match": itoa(child.Version + 5)})
	if rec.Code != http.StatusConflict {
		t.Fatalf("旧版本应返回 409，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 合法改名与清空父级。
	renamed := h.patchLocation(child.ID, itoa(child.Version), map[string]any{"name": "苍梧"})
	if renamed.Name != "苍梧" || renamed.Version != child.Version+1 {
		t.Fatalf("改名或版本递增不正确：%+v", renamed)
	}
	promoted := h.patchLocation(child.ID, itoa(renamed.Version), map[string]any{"parent_id": nil})
	if promoted.Parent != nil {
		t.Fatalf("清空父级后 parent 应为 null：%+v", promoted.Parent)
	}
}

func TestLocationDeleteBlockedByChildren(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("地点删除作品")
	root := h.createLocation(work.ID, "东荒", nil, nil)
	child := h.createLocation(work.ID, "苍梧星城", &root.ID, nil)

	rec := h.do(http.MethodDelete, "/api/v1/locations/"+root.ID, nil, map[string]string{"If-Match": itoa(root.Version)})
	if rec.Code != http.StatusConflict {
		t.Fatalf("有下级地点删除应返回 409，实际 %d：%s", rec.Code, rec.Body.String())
	}

	deleted := decodeBody[struct {
		Deleted   bool   `json:"deleted"`
		UndoToken string `json:"undo_token"`
	}](t, h.do(http.MethodDelete, "/api/v1/locations/"+child.ID, nil, map[string]string{"If-Match": itoa(child.Version)}))
	if !deleted.Deleted || deleted.UndoToken == "" {
		t.Fatalf("无引用地点删除应成功并返回撤销令牌：%+v", deleted)
	}
	if rec := h.do(http.MethodGet, "/api/v1/locations/"+child.ID, nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("删除后详情应 404，实际 %d", rec.Code)
	}
}

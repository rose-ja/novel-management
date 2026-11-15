//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/search"
)

type searchFixture struct {
	work      workPayload
	character characterPayload
	sword     characterPayload
	faction   factionPayload
	location  locationPayload
	ranking   rankingPayload
	category  categoryPayload
	nodeID    string
}

func seedSearchFixture(t *testing.T, h *harness) searchFixture {
	t.Helper()
	work := h.createWork("检索接口作品")
	category := h.createCategory(work.ID, "检索境界", "normal")
	nodeID := h.createCharacterNode(work.ID, category.ID, "", "圣者境")
	character := h.createCharacter(work.ID, "司空摘天", []characterNamePayload{
		{Kind: "alias", Value: "小圣"},
		{Kind: "title", Value: "神逆"},
	}, []map[string]string{{"category_id": category.ID, "node_id": nodeID}})
	faction := h.createFaction(work.ID, "山海妖族", nil, nil)
	location := h.createLocation(work.ID, "青丘", nil, nil)
	ranking := h.createRanking(work.ID, "天榜", 1)

	member := h.patchCharacter(character.ID, "1", map[string]any{"faction_id": faction.ID})
	if member.Code != http.StatusOK {
		t.Fatalf("设置人物势力失败（status=%d）：%s", member.Code, member.Body.String())
	}
	links := h.patchCharacter(character.ID, "2", map[string]any{
		"location_links": []map[string]any{{"location_id": location.ID}},
	})
	if links.Code != http.StatusOK {
		t.Fatalf("设置人物地点失败（status=%d）：%s", links.Code, links.Body.String())
	}
	description := h.patchCharacter(character.ID, "3", map[string]any{"description": "摘星者"})
	if description.Code != http.StatusOK {
		t.Fatalf("设置人物简介失败（status=%d）：%s", description.Code, description.Body.String())
	}
	seat := h.do(http.MethodPut, "/api/v1/rankings/"+ranking.ID+"/seats/1", map[string]any{
		"character_id": character.ID,
	}, map[string]string{"If-Match": `"1"`})
	if seat.Code != http.StatusOK {
		t.Fatalf("指派席位失败（status=%d）：%s", seat.Code, seat.Body.String())
	}
	sword := h.createCharacter(work.ID, "sword", []characterNamePayload{
		{Kind: "alias", Value: "青丘狐"},
	}, nil)
	return searchFixture{
		work: work, character: character, sword: sword,
		faction: faction, location: location, ranking: ranking,
		category: category, nodeID: nodeID,
	}
}

func getSearch(t *testing.T, h *harness, workID string, params map[string]string) *model.SearchResult {
	t.Helper()
	values := url.Values{"work_id": {workID}}
	for key, value := range params {
		values.Set(key, value)
	}
	rec := h.do(http.MethodGet, "/api/v1/search?"+values.Encode(), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("检索失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Cache"); got != "BYPASS" {
		t.Fatalf("X-Cache 应为 BYPASS，实际 %q", got)
	}
	result := decodeBody[model.SearchResult](t, rec)
	return &result
}

func findGroup(result *model.SearchResult, entityType string) *model.SearchGroup {
	for i := range result.Groups {
		if result.Groups[i].EntityType == entityType {
			return &result.Groups[i]
		}
	}
	return nil
}

func TestSearchAPIMatchingGroupsAndURLs(t *testing.T) {
	h := newHarness(t)
	fixture := seedSearchFixture(t, h)

	// 别名命中。
	result := getSearch(t, h, fixture.work.ID, map[string]string{"q": "小圣"})
	group := findGroup(result, model.SearchEntityCharacter)
	if group == nil || group.Total != 1 || len(group.Items) != 1 {
		t.Fatalf("别名检索应命中 1 条人物，实际 %+v", result.Groups)
	}
	item := group.Items[0]
	if item.EntityID != fixture.character.ID || item.Title != "司空摘天" {
		t.Fatalf("别名命中人物不正确：%+v", item)
	}
	if item.URL != "/characters?id="+fixture.character.ID {
		t.Fatalf("人物链接不正确：%q", item.URL)
	}
	assertMatchedField(t, item, "alias", "小圣")
	if !strings.HasPrefix(item.Subtitle, "人物 · ") {
		t.Fatalf("人物副标题应以「人物 · 」开头，实际 %q", item.Subtitle)
	}

	// 英文大小写不敏感，命中主名称。
	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "SWORD"})
	group = findGroup(result, model.SearchEntityCharacter)
	if group == nil || group.Total != 1 || group.Items[0].EntityID != fixture.sword.ID {
		t.Fatalf("英文大小写检索未命中 sword：%+v", result.Groups)
	}
	assertMatchedField(t, group.Items[0], "name", "sword")

	// 空格分词 AND。
	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "青丘 狐"})
	group = findGroup(result, model.SearchEntityCharacter)
	if group == nil || group.Total != 1 || group.Items[0].EntityID != fixture.sword.ID {
		t.Fatalf("分词 AND 结果不正确：%+v", result.Groups)
	}

	// 简介与关联实体命中。
	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "摘星"})
	group = findGroup(result, model.SearchEntityCharacter)
	if group == nil || group.Total != 1 {
		t.Fatalf("简介检索未命中：%+v", result.Groups)
	}
	assertMatchedField(t, group.Items[0], "desc", "摘星")

	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "山海妖族"})
	group = findGroup(result, model.SearchEntityCharacter)
	if group == nil || group.Total != 1 {
		t.Fatalf("关联实体检索未命中：%+v", result.Groups)
	}
	assertMatchedField(t, group.Items[0], "relation", "山海妖族")

	// 同一关键词双命中时按权重排序：别名命中优先于关联命中。
	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "青丘"})
	group = findGroup(result, model.SearchEntityCharacter)
	if group == nil || group.Total != 2 {
		t.Fatalf("青丘应命中 2 条人物，实际 %+v", result.Groups)
	}
	if group.Items[0].EntityID != fixture.sword.ID || group.Items[0].Score != 3 {
		t.Fatalf("别名命中应排在关联命中之前：%+v", group.Items)
	}
	if group.Items[1].EntityID != fixture.character.ID || group.Items[1].Score != 1 {
		t.Fatalf("关联命中权重应为 1：%+v", group.Items)
	}

	// 五类分组与系统节点链接。
	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "圣"})
	if len(result.Groups) != 2 {
		t.Fatalf("圣 应命中人物与体系节点两组，实际 %+v", result.Groups)
	}
	if result.Groups[0].EntityType != model.SearchEntityCharacter || result.Groups[1].EntityType != model.SearchEntitySystemNode {
		t.Fatalf("分组顺序应为人物在前，实际 %+v", result.Groups)
	}
	nodeItem := result.Groups[1].Items[0]
	expectedURL := "/system?category=" + url.QueryEscape(fixture.category.ID) + "&node=" + url.QueryEscape(fixture.nodeID)
	if nodeItem.URL != expectedURL {
		t.Fatalf("系统节点链接不正确：%q，期望 %q", nodeItem.URL, expectedURL)
	}

	// 榜单副标题。
	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "天榜", "entity_types": "ranking"})
	group = findGroup(result, model.SearchEntityRanking)
	if group == nil || group.Total != 1 {
		t.Fatalf("榜单检索未命中：%+v", result.Groups)
	}
	if group.Items[0].Subtitle != "榜单 · 1 席 · 1 人在榜" {
		t.Fatalf("榜单副标题不正确：%q", group.Items[0].Subtitle)
	}

	// 删除后不再命中。
	h.deleteCharacter(fixture.sword.ID, "1")
	result = getSearch(t, h, fixture.work.ID, map[string]string{"q": "SWORD"})
	if result.Total != 0 || len(result.Groups) != 0 {
		t.Fatalf("删除后不应命中，实际 %+v", result)
	}
}

func TestSearchAPIFiltersErrorsAndPagination(t *testing.T) {
	h := newHarness(t)
	fixture := seedSearchFixture(t, h)

	filterCases := []map[string]string{
		{"q": "", "entity_types": "character", "faction_id": fixture.faction.ID},
		{"q": "", "entity_types": "character", "system_category_id": fixture.category.ID},
		{"q": "", "entity_types": "character", "system_node_id": fixture.nodeID},
		{"q": "", "entity_types": "character", "location_id": fixture.location.ID},
		{"q": "", "entity_types": "character", "ranking_id": fixture.ranking.ID},
		{"q": "小圣", "entity_types": "character", "review_status": "confirmed"},
	}
	for _, params := range filterCases {
		result := getSearch(t, h, fixture.work.ID, params)
		group := findGroup(result, model.SearchEntityCharacter)
		if group == nil || group.Total != 1 || group.Items[0].EntityID != fixture.character.ID {
			t.Fatalf("筛选 %+v 应命中人物 A，实际 %+v", params, result.Groups)
		}
	}
	// 仅筛选时 tokens 必须是空数组而不是 null。
	filterOnly := getSearch(t, h, fixture.work.ID, map[string]string{"entity_types": "character", "faction_id": fixture.faction.ID})
	if filterOnly.Query.Tokens == nil || len(filterOnly.Query.Tokens) != 0 {
		t.Fatalf("仅筛选时 tokens 应为空数组，实际 %+v", filterOnly.Query.Tokens)
	}

	// 失效筛选返回 422 字段级错误。
	values := url.Values{
		"work_id":      {fixture.work.ID},
		"entity_types": {"character"},
		"faction_id":   {"missing"},
	}
	rec := h.do(http.MethodGet, "/api/v1/search?"+values.Encode(), nil, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("失效筛选应返回 422，实际 %d：%s", rec.Code, rec.Body.String())
	}
	errorBody := decodeBody[struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}](t, rec)
	if errorBody.Error.Code != "VALIDATION_ERROR" || errorBody.Error.Details["field"] != "faction_id" {
		t.Fatalf("失效筛选错误体不正确：%+v", errorBody)
	}

	// 空查询且无收窄筛选返回 400。
	rec = h.do(http.MethodGet, "/api/v1/search?"+url.Values{"work_id": {fixture.work.ID}}.Encode(), nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("空查询应返回 400，实际 %d：%s", rec.Code, rec.Body.String())
	}

	// 无结果不返回空分组。
	result := getSearch(t, h, fixture.work.ID, map[string]string{"q": "不存在"})
	if result.Total != 0 || len(result.Groups) != 0 {
		t.Fatalf("无结果不应返回分组：%+v", result)
	}

	// 分页：25 条合计，页 1 为 20、页 2 为 5，无重复。
	for i := 1; i <= 25; i++ {
		id := fmt.Sprintf("PGS%03d", i)
		name := fmt.Sprintf("批量人物%02d", i)
		if err := h.tx.Exec(
			`INSERT INTO characters (id, work_id, name, name_key) VALUES (?, ?, ?, ?)`,
			id, fixture.work.ID, name, name,
		).Error; err != nil {
			t.Fatalf("播种分页数据失败：%v", err)
		}
	}
	if err := search.NewRefresher(h.tx).RefreshWork(context.Background(), fixture.work.ID); err != nil {
		t.Fatalf("刷新分页数据失败：%v", err)
	}
	page1 := getSearch(t, h, fixture.work.ID, map[string]string{"q": "批量", "entity_types": "character", "page": "1"})
	page2 := getSearch(t, h, fixture.work.ID, map[string]string{"q": "批量", "entity_types": "character", "page": "2"})
	group1 := findGroup(page1, model.SearchEntityCharacter)
	group2 := findGroup(page2, model.SearchEntityCharacter)
	if page1.Total != 25 || group1 == nil || group1.Total != 25 || len(group1.Items) != 20 {
		t.Fatalf("第 1 页结果不正确：total=%d groups=%+v", page1.Total, page1.Groups)
	}
	if group2 == nil || len(group2.Items) != 5 || group2.Total != 25 {
		t.Fatalf("第 2 页结果不正确：%+v", page2.Groups)
	}
	seen := make(map[string]bool, 25)
	for _, item := range append(group1.Items, group2.Items...) {
		if seen[item.EntityID] {
			t.Fatalf("分页出现重复项：%s", item.EntityID)
		}
		seen[item.EntityID] = true
	}

	// 当前页无条目的类型不返回空分组。
	h.createFaction(fixture.work.ID, "批量势力", nil, nil)
	all1 := getSearch(t, h, fixture.work.ID, map[string]string{"q": "批量", "page": "1"})
	all2 := getSearch(t, h, fixture.work.ID, map[string]string{"q": "批量", "page": "2"})
	if all1.Total != 26 || all2.Total != 26 {
		t.Fatalf("合计应为 26，实际 page1=%d page2=%d", all1.Total, all2.Total)
	}
	if findGroup(all2, model.SearchEntityFaction) != nil {
		t.Fatalf("第 2 页不应出现空势力分组：%+v", all2.Groups)
	}
	if len(all2.Groups) != 1 || len(all2.Groups[0].Items) != 6 {
		t.Fatalf("第 2 页应仅返回 6 条人物，实际 %+v", all2.Groups)
	}

	// 非法 page 返回 422；超大 page 不溢出且返回空切片。
	for _, raw := range []string{"abc", "0", "-2"} {
		rec = h.do(http.MethodGet, "/api/v1/search?"+url.Values{
			"work_id": {fixture.work.ID}, "q": {"批量"}, "page": {raw},
		}.Encode(), nil, nil)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("page=%s 应返回 422，实际 %d：%s", raw, rec.Code, rec.Body.String())
		}
	}
	huge := getSearch(t, h, fixture.work.ID, map[string]string{"q": "批量", "page": "9223372036854775807"})
	if huge.Total != 26 || len(huge.Groups) != 0 {
		t.Fatalf("超大 page 应返回空切片，实际 total=%d groups=%+v", huge.Total, huge.Groups)
	}

	// 非法 review_status 返回 422。
	rec = h.do(http.MethodGet, "/api/v1/search?"+url.Values{
		"work_id": {fixture.work.ID}, "q": {"批量"}, "review_status": {"bad"},
	}.Encode(), nil, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("非法 review_status 应返回 422，实际 %d：%s", rec.Code, rec.Body.String())
	}
}

func assertMatchedField(t *testing.T, item model.SearchItem, field, contains string) {
	t.Helper()
	for _, matched := range item.MatchedFields {
		if matched.Field == field && strings.Contains(matched.MatchedText, contains) {
			return
		}
	}
	t.Fatalf("结果 %s 缺少命中字段 %s（包含 %q）：%+v", item.EntityID, field, contains, item.MatchedFields)
}

//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"sort"
	"testing"
	"time"
)

type characterNamePayload struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type characterSystemRefPayload struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	CategoryType string `json:"category_type"`
	Node         struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Parent *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"parent"`
	} `json:"node"`
}

type characterPayload struct {
	ID            string                      `json:"id"`
	WorkID        string                      `json:"work_id"`
	Name          string                      `json:"name"`
	Names         []characterNamePayload      `json:"names"`
	Faction       *factionBrief               `json:"faction"`
	SystemSummary *string                     `json:"system_summary"`
	SystemRefs    []characterSystemRefPayload `json:"system_refs"`
	LocationLinks []struct{}                  `json:"location_links"`
	ReviewStatus  string                      `json:"review_status"`
	Version       int                         `json:"version"`
	UpdatedAt     time.Time                   `json:"updated_at"`
}

type characterListPayload struct {
	Items    []characterPayload `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}

func (h *harness) createCharacterNode(workID, categoryID, parentID, name string) string {
	h.t.Helper()
	body := map[string]any{"work_id": workID, "category_id": categoryID, "name": name}
	if parentID != "" {
		body["parent_id"] = parentID
	}
	rec := h.do(http.MethodPost, "/api/v1/system-nodes", body, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建体系节点失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(h.t, rec, &payload); err != nil {
		h.t.Fatalf("解析节点响应失败：%v", err)
	}
	return payload.ID
}

func (h *harness) createCharacter(workID, name string, names []characterNamePayload, refs []map[string]string) characterPayload {
	h.t.Helper()
	body := map[string]any{"work_id": workID, "name": name}
	if len(names) > 0 {
		body["names"] = names
	}
	if len(refs) > 0 {
		body["system_refs"] = refs
	}
	rec := h.do(http.MethodPost, "/api/v1/characters", body, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建人物失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[characterPayload](h.t, rec)
}

func (h *harness) listCharacters(workID, query string) characterListPayload {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/characters?work_id="+workID+"&"+query, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取人物列表失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[characterListPayload](h.t, rec)
}

func characterNames(items []characterPayload) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func TestCharacterListFiltersSortAndPaging(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("人物列表")
	category := h.listCategories(work.ID, nil).Items[0]
	parentNode := h.createCharacterNode(work.ID, category.ID, "", "帝境")
	childNode := h.createCharacterNode(work.ID, category.ID, parentNode, "五重天")

	alpha := h.createCharacter(work.ID, "Alpha", []characterNamePayload{{Kind: "alias", Value: "Shadow"}}, []map[string]string{
		{"category_id": category.ID, "node_id": childNode},
	})
	h.createCharacter(work.ID, "Beta", []characterNamePayload{{Kind: "title", Value: "Sword Immortal"}}, nil)
	h.createCharacter(work.ID, "Gamma", nil, nil)
	h.createCharacter(work.ID, "100%Fox", nil, nil)

	rec := h.do(http.MethodGet, "/api/v1/characters", nil, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	list := h.listCharacters(work.ID, "sort=name")
	if list.Total != 4 || list.Page != 1 || list.PageSize != 20 {
		t.Fatalf("分页元数据不符：%+v", list)
	}
	if got := fmt.Sprint(characterNames(list.Items)); got != "[100%Fox Alpha Beta Gamma]" {
		t.Fatalf("按名称升序不符：%s", got)
	}
	if list.Items[1].SystemSummary == nil || *list.Items[1].SystemSummary != "帝境·五重天" {
		t.Fatalf("境界摘要应为父·子：%+v", list.Items[1].SystemSummary)
	}

	list = h.listCharacters(work.ID, "sort=-name")
	if got := fmt.Sprint(characterNames(list.Items)); got != "[Gamma Beta Alpha 100%Fox]" {
		t.Fatalf("按名称降序不符：%s", got)
	}

	list = h.listCharacters(work.ID, "q=shadow")
	if list.Total != 1 || list.Items[0].ID != alpha.ID {
		t.Fatalf("q 应大小写不敏感命中别名，实际 %+v", list.Items)
	}
	list = h.listCharacters(work.ID, "q=SWORD")
	if list.Total != 1 || list.Items[0].Name != "Beta" {
		t.Fatalf("q 应命中称号，实际 %+v", list.Items)
	}
	list = h.listCharacters(work.ID, "q=%25")
	if list.Total != 1 || list.Items[0].Name != "100%Fox" {
		t.Fatalf("q 中的 %% 应被转义，实际 %+v", list.Items)
	}

	list = h.listCharacters(work.ID, "system_node_id="+childNode)
	if list.Total != 1 || list.Items[0].ID != alpha.ID {
		t.Fatalf("境界筛选应仅命中选中子节点，实际 %+v", list.Items)
	}
	list = h.listCharacters(work.ID, "system_node_id="+parentNode)
	if list.Total != 0 {
		t.Fatalf("境界筛选不应命中父节点下的子节点记录，实际 %d", list.Total)
	}

	if err := h.tx.Exec("UPDATE characters SET review_status = 'pending' WHERE id = ?", alpha.ID).Error; err != nil {
		t.Fatalf("设置待校对失败：%v", err)
	}
	list = h.listCharacters(work.ID, "review_status=pending")
	if list.Total != 1 || list.Items[0].ID != alpha.ID {
		t.Fatalf("待校对筛选不符：%+v", list.Items)
	}
	list = h.listCharacters(work.ID, "review_status=confirmed")
	if list.Total != 3 {
		t.Fatalf("已确认筛选应返回 3 条，实际 %d", list.Total)
	}

	list = h.listCharacters(work.ID, "sort=name&page=2&page_size=2")
	if list.Page != 2 || list.PageSize != 2 || list.Total != 4 || len(list.Items) != 2 {
		t.Fatalf("分页结果不符：%+v", list)
	}
	if got := fmt.Sprint(characterNames(list.Items)); got != "[Beta Gamma]" {
		t.Fatalf("第二页内容不符：%s", got)
	}
	list = h.listCharacters(work.ID, "sort=name&page=9&page_size=2")
	if len(list.Items) != 0 || list.Total != 4 {
		t.Fatalf("越界页应返回空 items 且 total 正确：%+v", list)
	}
}

func TestCharacterListPerformance(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("人物性能")

	err := h.tx.Exec(`
		INSERT INTO characters (id, work_id, name, name_key, review_status, version, created_at, updated_at)
		SELECT 'PERF' || lpad(gs::text, 4, '0'), ?, 'Perf ' || gs, lower('perf ' || gs),
		       'reviewed', 1, now(), now()
		FROM generate_series(1, 1000) AS gs`, work.ID).Error
	if err != nil {
		t.Fatalf("批量写入 1000 人物失败：%v", err)
	}

	samples := make([]time.Duration, 0, 20)
	for i := 0; i < 20; i++ {
		start := time.Now()
		list := h.listCharacters(work.ID, "sort=-updated_at&page_size=20")
		samples = append(samples, time.Since(start))
		if list.Total != 1000 || len(list.Items) != 20 {
			t.Fatalf("1000 人列表结果不符：total=%d items=%d", list.Total, len(list.Items))
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p95 := samples[18]
	if p95 > 500*time.Millisecond {
		t.Fatalf("1000 人列表 P95 应 ≤500ms，实际 %v", p95)
	}
	t.Logf("1000 人列表 P95=%v", p95)
}

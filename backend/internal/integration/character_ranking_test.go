//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestCharacterRankingEchoAndDeleteBlocked(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("人物榜单回显")
	alpha := h.createCharacter(work.ID, "司空摘天", nil, nil)
	ranking := h.createRanking(work.ID, "封神榜", 10)
	seat1 := h.rankingSeat(ranking.ID, 1)
	if _, apiErr := h.putSeat(ranking.ID, 1, seat1.Version, map[string]any{"character_id": alpha.ID}); apiErr != nil {
		t.Fatalf("指派失败：%+v", *apiErr)
	}

	// 列表回显名次。
	list := h.listCharacters(work.ID, "")
	var listed *characterPayload
	for i := range list.Items {
		if list.Items[i].ID == alpha.ID {
			listed = &list.Items[i]
		}
	}
	if listed == nil {
		t.Fatal("人物列表缺少目标人物")
	}
	if listed.RankingCount != 1 || len(listed.Rankings) != 1 || listed.Rankings[0].SeatNo != 1 {
		t.Fatalf("人物列表名次回显错误：%+v", listed)
	}
	if listed.Rankings[0].Name != "封神榜" || listed.Rankings[0].RankingID != ranking.ID {
		t.Fatalf("人物列表榜单摘要错误：%+v", listed.Rankings[0])
	}

	// 详情回显名次。
	detailRec := h.do(http.MethodGet, "/api/v1/characters/"+alpha.ID, nil, nil)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("读取人物详情失败：%s", detailRec.Body.String())
	}
	var detail struct {
		Rankings []struct {
			RankingID string `json:"ranking_id"`
			SeatNo    int    `json:"seat_no"`
		} `json:"rankings"`
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("解析人物详情失败：%v", err)
	}
	if len(detail.Rankings) != 1 || detail.Rankings[0].RankingID != ranking.ID {
		t.Fatalf("人物详情名次回显错误：%+v", detail.Rankings)
	}

	// 活跃榜单席位阻止删除人物。
	rec := h.do(http.MethodDelete, "/api/v1/characters/"+alpha.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(alpha.Version) + `"`,
	})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	var refs []struct {
		Type     string  `json:"type"`
		EntityID string  `json:"entity_id"`
		Name     *string `json:"name"`
		Field    string  `json:"field"`
		Value    string  `json:"value"`
		InTrash  bool    `json:"in_trash"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &refs); err != nil {
		t.Fatalf("解析引用 details 失败：%v", err)
	}
	if len(refs) != 1 || refs[0].Type != "ranking_seat" || refs[0].InTrash || refs[0].Value != "1" {
		t.Fatalf("引用阻断 details 错误：%+v", refs)
	}

	// 榜单进回收站后仍阻止人物删除，并标记 in_trash。
	delRanking := h.do(http.MethodDelete, "/api/v1/rankings/"+ranking.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(ranking.Version) + `"`,
	})
	if delRanking.Code != http.StatusOK {
		t.Fatalf("删除榜单失败：%s", delRanking.Body.String())
	}
	rec = h.do(http.MethodDelete, "/api/v1/characters/"+alpha.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(alpha.Version) + `"`,
	})
	envelope = assertErrorCode(t, rec, http.StatusConflict, "REFERENCE_CONFLICT")
	refs = nil
	if err := json.Unmarshal(envelope.Error.Details, &refs); err != nil {
		t.Fatalf("解析回收站引用 details 失败：%v", err)
	}
	if len(refs) != 1 || !refs[0].InTrash {
		t.Fatalf("回收站榜单引用应标记 in_trash：%+v", refs)
	}

	// 恢复榜单并移出人物后可删除。
	restore := h.do(http.MethodPost, "/api/v1/trash/ranking/"+ranking.ID+"/restore", map[string]any{}, nil)
	if restore.Code != http.StatusOK {
		t.Fatalf("恢复榜单失败：%s", restore.Body.String())
	}
	current := h.rankingSeat(ranking.ID, 1)
	if _, apiErr := h.putSeat(ranking.ID, 1, current.Version, map[string]any{"character_id": nil}); apiErr != nil {
		t.Fatalf("移出人物失败：%+v", *apiErr)
	}
	rec = h.do(http.MethodDelete, "/api/v1/characters/"+alpha.ID, nil, map[string]string{
		"If-Match": `"` + strconv.Itoa(alpha.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("移出后删除人物应成功（status=%d）：%s", rec.Code, rec.Body.String())
	}
}

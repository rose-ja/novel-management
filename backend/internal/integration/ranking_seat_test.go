//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

type moveSeatPayload struct {
	From seatPayload `json:"from"`
	To   seatPayload `json:"to"`
}

func (h *harness) rankingSeat(rankingID string, seatNo int) seatPayload {
	h.t.Helper()
	detail := h.getRanking(rankingID)
	for _, seat := range detail.Seats.Items {
		if seat.SeatNo == seatNo {
			return seat
		}
	}
	h.t.Fatalf("榜单 %s 未找到席位 %d", rankingID, seatNo)
	return seatPayload{}
}

func TestRankingSeatAssignMoveAndSwap(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("榜单席位")
	ranking := h.createRanking(work.ID, "天榜", 10)
	alpha := h.createCharacter(work.ID, "甲", nil, nil)
	beta := h.createCharacter(work.ID, "乙", nil, nil)

	seat1 := h.rankingSeat(ranking.ID, 1)
	assigned, apiErr := h.putSeat(ranking.ID, 1, seat1.Version, map[string]any{"character_id": alpha.ID})
	if apiErr != nil {
		t.Fatalf("指派失败：%+v", *apiErr)
	}
	if assigned.Character == nil || assigned.Character.ID != alpha.ID || assigned.Version != seat1.Version+1 {
		t.Fatalf("指派结果错误：%+v", assigned)
	}

	// 同榜重复指派。
	seat2 := h.rankingSeat(ranking.ID, 2)
	_, apiErr = h.putSeat(ranking.ID, 2, seat2.Version, map[string]any{"character_id": alpha.ID})
	if apiErr == nil || apiErr.Error.Code != "SEAT_CONFLICT" {
		t.Fatalf("重复指派应返回 SEAT_CONFLICT：%+v", apiErr)
	}
	var conflictDetails struct {
		SeatNo         int    `json:"seat_no"`
		CharacterID    string `json:"character_id"`
		ExistingSeatNo int    `json:"existing_seat_no"`
	}
	if err := json.Unmarshal(apiErr.Error.Details, &conflictDetails); err != nil {
		t.Fatalf("解析冲突 details 失败：%v", err)
	}
	if conflictDetails.ExistingSeatNo != 1 {
		t.Fatalf("冲突 details 应指向第 1 席：%+v", conflictDetails)
	}

	// 仅更新备注（character_id 省略）。
	withNote, apiErr := h.putSeat(ranking.ID, 1, assigned.Version, map[string]any{"note": "并列"})
	if apiErr != nil {
		t.Fatalf("更新备注失败：%+v", *apiErr)
	}
	if withNote.Character == nil || withNote.Character.ID != alpha.ID || withNote.Note == nil || *withNote.Note != "并列" {
		t.Fatalf("备注更新不应清空人物：%+v", withNote)
	}

	// 移动到空席。
	moved := h.moveSeat(ranking.ID, 1, withNote.Version, 5, "empty")
	if moved.code != http.StatusOK {
		t.Fatalf("移动到空席失败（status=%d）：%s", moved.code, moved.body)
	}
	var moveResult moveSeatPayload
	if err := json.Unmarshal([]byte(moved.body), &moveResult); err != nil {
		t.Fatalf("解析移动响应失败：%v", err)
	}
	if moveResult.From.Character != nil || moveResult.To.Character == nil || moveResult.To.Character.ID != alpha.ID {
		t.Fatalf("移动后席位错误：%+v", moveResult)
	}
	if moveResult.To.Note == nil || *moveResult.To.Note != "并列" {
		t.Fatalf("移动应携带备注：%+v", moveResult.To)
	}

	// 目标已占且 mode=empty。
	seat7 := h.rankingSeat(ranking.ID, 7)
	if _, apiErr = h.putSeat(ranking.ID, 7, seat7.Version, map[string]any{"character_id": beta.ID}); apiErr != nil {
		t.Fatalf("指派第二人失败：%+v", *apiErr)
	}
	occupiedMove := h.moveSeat(ranking.ID, 5, moveResult.To.Version, 7, "empty")
	if occupiedMove.code != http.StatusConflict {
		t.Fatalf("移动到已占席应 409：%d %s", occupiedMove.code, occupiedMove.body)
	}

	// 交换。
	swap := h.moveSeat(ranking.ID, 5, moveResult.To.Version, 7, "swap")
	if swap.code != http.StatusOK {
		t.Fatalf("交换失败（status=%d）：%s", swap.code, swap.body)
	}
	if err := json.Unmarshal([]byte(swap.body), &moveResult); err != nil {
		t.Fatalf("解析交换响应失败：%v", err)
	}
	if moveResult.From.Character == nil || moveResult.From.Character.ID != beta.ID {
		t.Fatalf("交换后第 5 席应为乙：%+v", moveResult.From)
	}
	if moveResult.To.Character == nil || moveResult.To.Character.ID != alpha.ID {
		t.Fatalf("交换后第 7 席应为甲：%+v", moveResult.To)
	}

	// 越界。
	outOfRange := h.moveSeat(ranking.ID, 5, moveResult.From.Version, 30, "empty")
	if outOfRange.code != http.StatusBadRequest {
		t.Fatalf("越界移动应 400：%d %s", outOfRange.code, outOfRange.body)
	}
	rec := h.do(http.MethodPut, "/api/v1/rankings/"+ranking.ID+"/seats/abc", map[string]any{}, map[string]string{
		"If-Match": `"1"`,
	})
	assertErrorCode(t, rec, http.StatusBadRequest, "SEAT_OUT_OF_RANGE")

	// 清空席位。
	cleared, apiErr := h.putSeat(ranking.ID, 7, moveResult.To.Version, map[string]any{"character_id": nil})
	if apiErr != nil {
		t.Fatalf("清空席位失败：%+v", *apiErr)
	}
	if cleared.Character != nil || cleared.Note != nil {
		t.Fatalf("清空应同时清备注：%+v", cleared)
	}
}

func TestRankingShrinkBlockedByOccupiedSeat(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("榜单缩容")
	ranking := h.createRanking(work.ID, "地榜", 10)
	alpha := h.createCharacter(work.ID, "甲", nil, nil)

	seat5 := h.rankingSeat(ranking.ID, 5)
	if _, apiErr := h.putSeat(ranking.ID, 5, seat5.Version, map[string]any{"character_id": alpha.ID}); apiErr != nil {
		t.Fatalf("指派失败：%+v", *apiErr)
	}

	rec := h.do(http.MethodPatch, "/api/v1/rankings/"+ranking.ID, map[string]any{"seat_count": 4}, map[string]string{
		"If-Match": `"` + strconv.Itoa(ranking.Version) + `"`,
	})
	envelope := assertErrorCode(t, rec, http.StatusConflict, "SEAT_SHRINK_OCCUPIED")
	var details struct {
		FromSeatCount int `json:"from_seat_count"`
		ToSeatCount   int `json:"to_seat_count"`
		Items         []struct {
			SeatNo      int    `json:"seat_no"`
			CharacterID string `json:"character_id"`
			Name        string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(envelope.Error.Details, &details); err != nil {
		t.Fatalf("解析缩容 details 失败：%v", err)
	}
	if details.FromSeatCount != 10 || details.ToSeatCount != 4 || len(details.Items) != 1 || details.Items[0].SeatNo != 5 {
		t.Fatalf("缩容 details 错误：%+v", details)
	}

	// 清空尾部后缩容成功。
	current := h.rankingSeat(ranking.ID, 5)
	if _, apiErr := h.putSeat(ranking.ID, 5, current.Version, map[string]any{"character_id": nil}); apiErr != nil {
		t.Fatalf("清空失败：%+v", *apiErr)
	}
	rec = h.do(http.MethodPatch, "/api/v1/rankings/"+ranking.ID, map[string]any{"seat_count": 4}, map[string]string{
		"If-Match": `"` + strconv.Itoa(ranking.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("缩容失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	shrunk := decodeBody[rankingPayload](t, rec)
	if shrunk.SeatCount != 4 || shrunk.Seats.Total != 4 {
		t.Fatalf("缩容后席位错误：%+v", shrunk)
	}
}

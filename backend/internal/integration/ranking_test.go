//go:build integration

package integration

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

type rankingTopSeatPayload struct {
	SeatNo    int `json:"seat_no"`
	Character *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"character"`
}

type seatPayload struct {
	SeatNo    int               `json:"seat_no"`
	Character *characterPayload `json:"character"`
	Note      *string           `json:"note"`
	Version   int               `json:"version"`
}

type seatPagePayload struct {
	Items    []seatPayload `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
}

type rankingPayload struct {
	ID            string                  `json:"id"`
	WorkID        string                  `json:"work_id"`
	Name          string                  `json:"name"`
	SeatCount     int                     `json:"seat_count"`
	OccupiedCount int                     `json:"occupied_count"`
	EmptyCount    int                     `json:"empty_count"`
	TopSeats      []rankingTopSeatPayload `json:"top_seats"`
	ReviewStatus  string                  `json:"review_status"`
	Version       int                     `json:"version"`
	UpdatedAt     time.Time               `json:"updated_at"`
	Seats         seatPagePayload         `json:"seats"`
	UndoToken     string                  `json:"undo_token"`
}

type rankingListPayload struct {
	Items    []rankingPayload `json:"items"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Total    int64            `json:"total"`
}

func (h *harness) createRanking(workID, name string, seatCount int) rankingPayload {
	h.t.Helper()
	rec := h.do(http.MethodPost, "/api/v1/rankings", map[string]any{
		"work_id":    workID,
		"name":       name,
		"seat_count": seatCount,
	}, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建榜单失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location == "" {
		h.t.Fatal("创建榜单应返回 Location")
	}
	return decodeBody[rankingPayload](h.t, rec)
}

func (h *harness) getRanking(id string) rankingPayload {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/rankings/"+id, nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取榜单失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[rankingPayload](h.t, rec)
}

func (h *harness) putSeat(rankingID string, seatNo, version int, body map[string]any) (seatPayload, *errorEnvelope) {
	h.t.Helper()
	rec := h.do(http.MethodPut, "/api/v1/rankings/"+rankingID+"/seats/"+strconv.Itoa(seatNo), body, map[string]string{
		"If-Match": `"` + strconv.Itoa(version) + `"`,
	})
	if rec.Code == http.StatusOK {
		return decodeBody[seatPayload](h.t, rec), nil
	}
	envelope := decodeBody[errorEnvelope](h.t, rec)
	return seatPayload{}, &envelope
}

func (h *harness) moveSeat(rankingID string, seatNo, version int, target int, mode string) *httptestResponse {
	h.t.Helper()
	rec := h.do(http.MethodPost, "/api/v1/rankings/"+rankingID+"/seats/"+strconv.Itoa(seatNo)+"/move", map[string]any{
		"target_seat_no": target,
		"mode":           mode,
	}, map[string]string{
		"If-Match": `"` + strconv.Itoa(version) + `"`,
	})
	return &httptestResponse{code: rec.Code, body: rec.Body.String()}
}

type httptestResponse struct {
	code int
	body string
}

func TestRankingCreateListAndCapacity(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("榜单作品")

	created := h.createRanking(work.ID, "封神榜", 100)
	if created.SeatCount != 100 || created.OccupiedCount != 0 || created.EmptyCount != 100 {
		t.Fatalf("新建榜单计数错误：%+v", created)
	}
	if created.Seats.Total != 100 || len(created.Seats.Items) != 50 {
		t.Fatalf("详情默认应返回 50/100 席：total=%d items=%d", created.Seats.Total, len(created.Seats.Items))
	}
	if created.Seats.Items[0].SeatNo != 1 || created.Seats.Items[0].Character != nil {
		t.Fatalf("首个席位应为空席 1：%+v", created.Seats.Items[0])
	}

	// 大小写等价重名。
	rec := h.do(http.MethodPost, "/api/v1/rankings", map[string]any{
		"work_id": work.ID, "name": "  封神榜  ", "seat_count": 10,
	}, nil)
	assertErrorCode(t, rec, http.StatusConflict, "NAME_CONFLICT")

	// 席位数越界。
	rec = h.do(http.MethodPost, "/api/v1/rankings", map[string]any{
		"work_id": work.ID, "name": "越界榜", "seat_count": 201,
	}, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	// 列表摘要。
	listRec := h.do(http.MethodGet, "/api/v1/rankings?work_id="+work.ID, nil, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("榜单列表失败：%s", listRec.Body.String())
	}
	list := decodeBody[rankingListPayload](t, listRec)
	if list.Total != 1 || list.Items[0].ID != created.ID {
		t.Fatalf("榜单列表错误：%+v", list)
	}
	if len(list.Items[0].TopSeats) != 3 || list.Items[0].TopSeats[0].Character != nil {
		t.Fatalf("前三席应为 3 个空席：%+v", list.Items[0].TopSeats)
	}

	// 扩容到 120。
	rec = h.do(http.MethodPatch, "/api/v1/rankings/"+created.ID, map[string]any{"seat_count": 120}, map[string]string{
		"If-Match": `"` + strconv.Itoa(created.Version) + `"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("扩容失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	expanded := decodeBody[rankingPayload](t, rec)
	if expanded.SeatCount != 120 || expanded.EmptyCount != 120 {
		t.Fatalf("扩容后计数错误：%+v", expanded)
	}

	// 缺 If-Match。
	rec = h.do(http.MethodPatch, "/api/v1/rankings/"+created.ID, map[string]any{"seat_count": 130}, nil)
	assertErrorCode(t, rec, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")

	// 过期版本。
	rec = h.do(http.MethodPatch, "/api/v1/rankings/"+created.ID, map[string]any{"seat_count": 130}, map[string]string{
		"If-Match": `"` + strconv.Itoa(created.Version) + `"`,
	})
	assertErrorCode(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestRankingListSortAndSeatPageValidation(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("榜单排序")

	first := h.createRanking(work.ID, "B榜", 10)
	h.createRanking(work.ID, "A榜", 10)

	rec := h.do(http.MethodGet, "/api/v1/rankings?work_id="+work.ID+"&sort=name:asc", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("按名称排序失败：%s", rec.Body.String())
	}
	list := decodeBody[rankingListPayload](t, rec)
	if list.Items[0].Name != "A榜" || list.Items[1].Name != "B榜" {
		t.Fatalf("名称升序错误：%+v", list.Items)
	}

	rec = h.do(http.MethodGet, "/api/v1/rankings?work_id="+work.ID+"&sort=bad", nil, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	rec = h.do(http.MethodGet, "/api/v1/rankings/"+first.ID+"?page_size=20", nil, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	rec = h.do(http.MethodGet, "/api/v1/rankings/"+first.ID+"/seats?page_size=100", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("席位分页 100 失败：%s", rec.Body.String())
	}

	rec = h.do(http.MethodGet, "/api/v1/rankings/"+first.ID+"/seats?occupied=maybe", nil, nil)
	assertErrorCode(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	rec = h.do(http.MethodGet, "/api/v1/rankings/"+first.ID, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("读取榜单详情失败：%s", rec.Body.String())
	}
}

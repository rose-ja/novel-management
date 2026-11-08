package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Details   []any  `json:"details"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

func TestUnknownAPIReturnsJSONWithRequestID(t *testing.T) {
	router := newTestRouter(HealthDeps{})

	rec := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17860/api/v1/unknown", nil)
	request.Header.Set("X-Request-ID", "req-test-1")
	router.ServeHTTP(rec, request)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d", rec.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if body.Error.Code != "NOT_FOUND" || body.Error.Message == "" {
		t.Fatalf("错误体不符：%+v", body.Error)
	}
	if body.Error.RequestID != "req-test-1" {
		t.Fatalf("期望回传 request_id，实际 %q", body.Error.RequestID)
	}
	if body.Error.Details != nil {
		t.Fatalf("未提供 details 时不应出现该字段：%+v", body.Error.Details)
	}
}

func TestAbortErrorWithDetailsCarriesLocators(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(requestIDKey, "req-test-2")

	details := []map[string]any{{"type": "faction_leader", "entity_id": "01H", "name": "山海妖族"}}
	abortErrorWithDetails(c, http.StatusConflict, "REFERENCE_CONFLICT", "该人物仍被 1 处引用", details)

	if rec.Code != http.StatusConflict {
		t.Fatalf("期望 409，实际 %d", rec.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if len(body.Error.Details) != 1 || body.Error.Code != "REFERENCE_CONFLICT" || body.Error.RequestID != "req-test-2" {
		t.Fatalf("details 未按契约输出：%+v", body.Error)
	}
}

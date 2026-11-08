//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// AC-401：CHG-001 基线端点行为不变。
func TestBaselineEndpoints(t *testing.T) {
	h := newHarness(t)

	health := h.do(http.MethodGet, "/api/v1/health", nil, nil)
	if health.Code != http.StatusOK {
		t.Fatalf("健康检查失败（status=%d）：%s", health.Code, health.Body.String())
	}
	var payload struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := decodeJSON(t, health, &payload); err != nil {
		t.Fatalf("解析健康响应失败：%v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("健康状态应为 ok，实际 %q", payload.Status)
	}

	meta := h.do(http.MethodGet, "/api/v1/meta", nil, nil)
	if meta.Code != http.StatusOK {
		t.Fatalf("meta 失败（status=%d）", meta.Code)
	}

	missing := h.do(http.MethodGet, "/api/v1/unknown", nil, nil)
	assertErrorCode(t, missing, http.StatusNotFound, "NOT_FOUND")
}

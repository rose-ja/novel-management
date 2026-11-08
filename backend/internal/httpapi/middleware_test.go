package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostGuardRejectsUnknownHost(t *testing.T) {
	router := newTestRouter(HealthDeps{})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://evil.example/api/v1/health", nil))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际 %d", rec.Code)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	router := newTestRouter(HealthDeps{})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17860/api/v1/health", nil))

	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("缺少 Content-Security-Policy")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options 不符：%q", got)
	}
	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("缺少 X-Request-ID")
	}
}

func TestOriginGuardRejectsCrossOriginWrites(t *testing.T) {
	router := newTestRouter(HealthDeps{})

	cases := []struct {
		name   string
		method string
		origin string
		status int
	}{
		{name: "跨源写请求被拒", method: http.MethodPost, origin: "https://evil.example", status: http.StatusForbidden},
		{name: "同源写请求放行", method: http.MethodPost, origin: "http://127.0.0.1:17860", status: http.StatusNotFound},
		{name: "无 Origin 的写请求放行（curl）", method: http.MethodPost, status: http.StatusNotFound},
		{name: "跨源读请求不受影响", method: http.MethodGet, origin: "https://evil.example", status: http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://127.0.0.1:17860/api/v1/", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("期望 %d，实际 %d：%s", tc.status, rec.Code, rec.Body.String())
			}
		})
	}
}

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

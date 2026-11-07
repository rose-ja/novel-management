package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/config"
)

func newTestRouter(health HealthDeps) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouter(Options{
		Config: config.Config{Version: "test", AllowedHosts: []string{"127.0.0.1:17860"}},
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Health: health,
	})
}

type healthResponse struct {
	Status          string `json:"status"`
	Version         string `json:"version"`
	DB              string `json:"db"`
	Cache           string `json:"cache"`
	DataDirWritable bool   `json:"data_dir_writable"`
}

func TestHealthReturnsOKWhenDatabaseIsAvailable(t *testing.T) {
	router := newTestRouter(HealthDeps{
		DBPing:          func(context.Context) error { return nil },
		CachePing:       nil,
		DataDirWritable: func() bool { return true },
		Version:         "test",
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17860/api/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var body healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if body.Status != "ok" || body.DB != "ok" || body.Cache != "disabled" || !body.DataDirWritable {
		t.Fatalf("健康响应不符：%+v", body)
	}
}

func TestHealthReturns503AndDegradedWithoutDatabase(t *testing.T) {
	router := newTestRouter(HealthDeps{
		DBPing:          func(context.Context) error { return errors.New("db down") },
		CachePing:       func(context.Context) error { return errors.New("cache down") },
		DataDirWritable: func() bool { return false },
		Version:         "test",
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17860/api/v1/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("期望 503，实际 %d", rec.Code)
	}
	var body healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if body.Status != "degraded" || body.DB != "unavailable" || body.Cache != "degraded" {
		t.Fatalf("降级响应不符：%+v", body)
	}
}

func TestMetaExposesContractLimits(t *testing.T) {
	router := newTestRouter(HealthDeps{})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17860/api/v1/meta", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rec.Code)
	}
	var body struct {
		APIVersion string         `json:"api_version"`
		Limits     map[string]int `json:"limits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if body.APIVersion != "v1" || body.Limits["name"] != 200 || body.Limits["category_max"] != 10 {
		t.Fatalf("/meta 常量不符：%+v", body)
	}
}

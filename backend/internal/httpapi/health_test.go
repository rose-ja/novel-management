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
	want := map[string]int{
		"name":            200,
		"description":     20000,
		"chronicle":       2000,
		"relation_note":   500,
		"era_text":        100,
		"seat_max":        200,
		"category_max":    10,
		"upload_image_mb": 10,
	}
	for key, value := range want {
		if body.Limits[key] != value {
			t.Fatalf("/meta limits[%s] 期望 %d，实际 %d", key, value, body.Limits[key])
		}
	}
}

func TestSettingsExposesReadOnlyFields(t *testing.T) {
	router := newTestRouter(HealthDeps{})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:17860/api/v1/settings", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rec.Code)
	}
	var body struct {
		CurrentWorkID *string `json:"current_work_id"`
		DataDir       string  `json:"data_dir"`
		SnapshotKeep  int     `json:"snapshot_keep"`
		AppVersion    string  `json:"app_version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if body.CurrentWorkID != nil {
		t.Fatalf("CHG-002 之前 current_work_id 应为 null，实际 %v", *body.CurrentWorkID)
	}
	if body.AppVersion != "test" {
		t.Fatalf("版本不符：%q", body.AppVersion)
	}
}

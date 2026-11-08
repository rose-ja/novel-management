//go:build integration

// Package integration 覆盖 CHG-002 的端到端后端行为；需要真实 PostgreSQL。
// 每个用例在独立事务中运行并回滚，因此不会污染应用数据库。
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/database"
	"novel-management/backend/internal/httpapi"
	"novel-management/backend/internal/migrations"
	"novel-management/backend/internal/service"
)

var testDB *gorm.DB

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestMain(m *testing.M) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "SKIP integration: TEST_DATABASE_URL 未设置")
		fmt.Fprintln(os.Stderr, `示例：docker compose up -d --wait db; `+
			`$env:TEST_DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"; `+
			`go test -tags=integration ./internal/integration/...`)
		os.Exit(0)
	}

	gin.SetMode(gin.TestMode)
	scratchDSN, err := ensureScratchDatabase(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "准备测试数据库失败：%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	scratch, err := database.Open(ctx, scratchDSN, 30*time.Second, discardLogger())
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接测试数据库失败：%v\n", err)
		os.Exit(1)
	}
	if err := migrations.Up(ctx, scratch.SQL); err != nil {
		fmt.Fprintf(os.Stderr, "迁移测试数据库失败：%v\n", err)
		_ = scratch.SQL.Close()
		os.Exit(1)
	}

	testDB = scratch.Gorm
	code := m.Run()
	_ = scratch.SQL.Close()
	os.Exit(code)
}

// ensureScratchDatabase 在同一个 PostgreSQL 实例上创建独立库 {db}_it，
// 避免集成测试清空或污染应用正式数据。
func ensureScratchDatabase(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	base := strings.TrimPrefix(u.Path, "/")
	if base == "" {
		return "", errors.New("TEST_DATABASE_URL 缺少数据库名")
	}

	admin := *u
	admin.Path = "/postgres"
	adminURL := admin.String()

	db, err := database.Open(context.Background(), adminURL, 10*time.Second, discardLogger())
	if err != nil {
		return "", err
	}
	defer func() { _ = db.SQL.Close() }()

	scratch := base + "_it"
	quoted := `"` + strings.ReplaceAll(scratch, `"`, `""`) + `"`
	ctx := context.Background()
	_, _ = db.SQL.ExecContext(ctx,
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()", scratch)
	if _, err := db.SQL.ExecContext(ctx, "DROP DATABASE IF EXISTS "+quoted); err != nil {
		return "", err
	}
	if _, err := db.SQL.ExecContext(ctx, "CREATE DATABASE "+quoted); err != nil {
		return "", err
	}

	result := *u
	result.Path = "/" + scratch
	return result.String(), nil
}

type flakyDumper struct {
	fail bool
}

func (d *flakyDumper) Dump(_ context.Context, _ string, outPath string) error {
	if d.fail {
		return errors.New("模拟 pg_dump 失败")
	}
	return os.WriteFile(outPath, []byte("fake-dump"), 0o644)
}

type harness struct {
	t         *testing.T
	tx        *gorm.DB
	engine    *gin.Engine
	dataDir   string
	dumper    *flakyDumper
	snapshots *service.Snapshot
	works     *service.Works
	trash     *service.Trash
	settings  *service.Settings
	system    *service.System
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	tx := testDB.Begin()
	if tx.Error != nil {
		t.Fatalf("开启测试事务失败：%v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "images"), 0o755); err != nil {
		t.Fatalf("准备数据目录失败：%v", err)
	}

	dumper := &flakyDumper{}
	cfg := config.Config{
		DataDir:      dataDir,
		WebDir:       t.TempDir(),
		DatabaseURL:  "postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable",
		AllowedHosts: []string{"127.0.0.1:17860"},
		SnapshotKeep: 3,
		Version:      "integration",
	}

	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败：%v", err)
	}

	snapshots := service.NewSnapshot(tx, cfg, discardLogger(), dumper)
	works := service.NewWorks(tx, snapshots)
	trash := service.NewTrash(tx, snapshots)
	settings := service.NewSettings(tx)
	system := service.NewSystem(tx)
	engine := httpapi.NewRouter(httpapi.Options{
		Config: cfg,
		Log:    discardLogger(),
		Health: httpapi.HealthDeps{
			DBPing:          sqlDB.PingContext,
			DataDirWritable: func() bool { return true },
			Version:         cfg.Version,
		},
		Works:     works,
		Trash:     trash,
		Snapshots: snapshots,
		Settings:  settings,
		System:    system,
	})

	return &harness{
		t:         t,
		tx:        tx,
		engine:    engine,
		dataDir:   dataDir,
		dumper:    dumper,
		snapshots: snapshots,
		works:     works,
		trash:     trash,
		settings:  settings,
		system:    system,
	}
}

func (h *harness) do(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	h.t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("序列化请求体失败：%v", err)
		}
		reader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, "http://127.0.0.1:17860"+path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	h.engine.ServeHTTP(rec, req)
	return rec
}

type workPayload struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Version     int     `json:"version"`
	DeletedAt   *string `json:"deleted_at"`
}

type deletePayload struct {
	Deleted    bool   `json:"deleted"`
	UndoToken  string `json:"undo_token"`
	SnapshotID string `json:"snapshot_id"`
}

type workPage struct {
	Items    []workPayload `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int64         `json:"total"`
}

type trashPage struct {
	Items []struct {
		RootType      string  `json:"root_type"`
		RootID        string  `json:"root_id"`
		Name          string  `json:"name"`
		AffectedCount int64   `json:"affected_count"`
		DeletedAt     *string `json:"deleted_at"`
	} `json:"items"`
	Total int64 `json:"total"`
}

type undoPayload struct {
	Restored bool   `json:"restored"`
	RootType string `json:"root_type"`
	RootID   string `json:"root_id"`
}

type restorePayload struct {
	RootType      string `json:"root_type"`
	RootID        string `json:"root_id"`
	Name          string `json:"name"`
	RestoredCount int64  `json:"restored_count"`
}

type purgeAllPayload struct {
	Purged     int64  `json:"purged"`
	SnapshotID string `json:"snapshot_id"`
}

type errorEnvelope struct {
	Error struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Details json.RawMessage `json:"details"`
	} `json:"error"`
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应失败（status=%d）：%v，body=%s", rec.Code, err, rec.Body.String())
	}
	return out
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dst any) error {
	t.Helper()
	return json.Unmarshal(rec.Body.Bytes(), dst)
}

func (h *harness) createWork(name string) workPayload {
	h.t.Helper()
	rec := h.do(http.MethodPost, "/api/v1/works", map[string]any{"name": name}, nil)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("创建作品失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[workPayload](h.t, rec)
}

func (h *harness) deleteWork(id string, confirmName string, version string) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.do(http.MethodDelete, "/api/v1/works/"+id, map[string]any{"confirm_name": confirmName}, map[string]string{
		"If-Match": `"` + version + `"`,
	})
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) errorEnvelope {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("期望状态码 %d，实际 %d：%s", status, rec.Code, rec.Body.String())
	}
	envelope := decodeBody[errorEnvelope](t, rec)
	if envelope.Error.Code != code {
		t.Fatalf("期望错误码 %s，实际 %s：%s", code, envelope.Error.Code, rec.Body.String())
	}
	return envelope
}

func (h *harness) trashTotal() int64 {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/trash", nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取回收站失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	return decodeBody[trashPage](h.t, rec).Total
}

func (h *harness) currentWorkID() *string {
	h.t.Helper()
	rec := h.do(http.MethodGet, "/api/v1/settings", nil, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("读取设置失败（status=%d）：%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		CurrentWorkID *string `json:"current_work_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		h.t.Fatalf("解析设置响应失败：%v", err)
	}
	return payload.CurrentWorkID
}

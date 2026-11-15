package service

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"novel-management/backend/internal/config"
)

func newTestExport(dataDir string) *Export {
	return &Export{
		cfg:    config.Config{DataDir: dataDir, Version: "0.1.0"},
		log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		dumper: nil,
		now:    time.Now,
	}
}

func TestUniqueFileNameAddsSuffixWhenCollision(t *testing.T) {
	dataDir := t.TempDir()
	svc := newTestExport(dataDir)
	if err := os.MkdirAll(filepath.Join(dataDir, "exports"), 0o755); err != nil {
		t.Fatalf("准备目录失败：%v", err)
	}
	base := "novel-settings-20261115-120000.zip"
	if err := os.WriteFile(filepath.Join(dataDir, "exports", base), []byte("old"), 0o644); err != nil {
		t.Fatalf("准备旧包失败：%v", err)
	}

	got := svc.uniqueFileName(base, "01TESTEXPORTID0000000000")
	want := "novel-settings-20261115-120000-000000.zip"
	if got != want {
		t.Fatalf("冲突文件名不符：%q，期望 %q", got, want)
	}
	if svc.uniqueFileName("novel-settings-20261115-130000.zip", "01TESTEXPORTID0000000000") !=
		"novel-settings-20261115-130000.zip" {
		t.Fatal("无冲突时应保持原名")
	}
}

func TestReadmeContentIncludesTrashAndRestoreCommand(t *testing.T) {
	content := readmeContent(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC), false, "backup.zip")
	for _, want := range []string{"包含回收站数据：否", "docker compose run --rm cli restore /data/exports/backup.zip --yes", "checksums.sha256"} {
		if !strings.Contains(content, want) {
			t.Fatalf("README 缺少 %q：%s", want, content)
		}
	}
}

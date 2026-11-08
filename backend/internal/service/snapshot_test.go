package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/model"
)

type fakeDumper struct {
	err error
}

func (f fakeDumper) Dump(_ context.Context, _ string, outPath string) error {
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(outPath, []byte("fake-dump"), 0o644)
}

func newTestSnapshot(t *testing.T, dumper Dumper) (*Snapshot, string) {
	t.Helper()
	dataDir := t.TempDir()
	return &Snapshot{
		cfg:    config.Config{DataDir: dataDir, DatabaseURL: "postgres://user:pass@localhost:5432/db"},
		log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		dumper: dumper,
		now:    time.Now,
	}, dataDir
}

func TestSnapshotProduceWritesFiveArtifacts(t *testing.T) {
	svc, dataDir := newTestSnapshot(t, fakeDumper{})
	imagesDir := filepath.Join(dataDir, "images", "nested")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		t.Fatalf("准备图片目录失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "a.png"), []byte("image"), 0o644); err != nil {
		t.Fatalf("准备图片失败：%v", err)
	}

	dir := filepath.Join(dataDir, "snapshots", "20261108T010203Z-work_delete")
	record := &model.Snapshot{
		ID:        "01TESTSNAPSHOT0000000000",
		Reason:    model.SnapshotReasonWorkDelete,
		CreatedAt: time.Date(2026, 11, 8, 1, 2, 3, 0, time.UTC),
	}
	if err := svc.produce(context.Background(), dir, record); err != nil {
		t.Fatalf("生成快照失败：%v", err)
	}

	for _, name := range []string{"app.dump", "images", "snapshot.json", "manifest.json", "checksums.sha256"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("缺少快照文件 %s：%v", name, err)
		}
	}
	if record.ImageCount != 1 {
		t.Fatalf("图片数量应为 1，实际 %d", record.ImageCount)
	}
	if record.DBBytes != int64(len("fake-dump")) {
		t.Fatalf("dump 大小不符：%d", record.DBBytes)
	}

	checksums, err := os.ReadFile(filepath.Join(dir, "checksums.sha256"))
	if err != nil {
		t.Fatalf("读取校验和失败：%v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(checksums)), "\n")
	if len(lines) != 4 {
		t.Fatalf("校验和应覆盖 4 个文件（不含 checksums 自身），实际 %d：%v", len(lines), lines)
	}
	for _, line := range lines {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			t.Fatalf("校验和行格式不符：%q", line)
		}
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(parts[1])))
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", parts[1], err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != parts[0] {
			t.Fatalf("%s 校验和不匹配", parts[1])
		}
	}

	var meta snapshotMeta
	data, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		t.Fatalf("读取 snapshot.json 失败：%v", err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("解析 snapshot.json 失败：%v", err)
	}
	if meta.ID != record.ID || meta.Reason != model.SnapshotReasonWorkDelete {
		t.Fatalf("snapshot.json 内容不符：%+v", meta)
	}
}

func TestSnapshotProduceFailsWhenDumpFails(t *testing.T) {
	svc, dataDir := newTestSnapshot(t, fakeDumper{err: errors.New("pg_dump 退出码 1")})
	record := &model.Snapshot{ID: "01TESTSNAPSHOTFAIL00000000", Reason: model.SnapshotReasonWorkDelete}

	err := svc.produce(context.Background(), filepath.Join(dataDir, "snapshots", "fail"), record)
	if err == nil {
		t.Fatal("dump 失败应返回错误")
	}
	if _, statErr := os.Stat(filepath.Join(dataDir, "snapshots", "fail", "manifest.json")); !os.IsNotExist(statErr) {
		t.Fatal("失败时不应写出 manifest.json")
	}
}

func TestSnapshotDirUsesTimestampAndReason(t *testing.T) {
	svc, _ := newTestSnapshot(t, fakeDumper{})
	created := time.Date(2026, 11, 8, 9, 30, 15, 0, time.UTC)
	got := svc.snapshotDir(created, model.SnapshotReasonWorkDelete, "01TESTID")
	want := filepath.Join(svc.cfg.DataDir, "snapshots", "20261108T093015Z-work_delete")
	if got != want {
		t.Fatalf("快照目录不符：%q", got)
	}
}

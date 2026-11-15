package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"novel-management/backend/internal/backup"
	"novel-management/backend/internal/config"
	"novel-management/backend/internal/model"
)

type fakeSnapshotCreator struct {
	path string
	err  error
}

func (f fakeSnapshotCreator) Create(context.Context, string) (*model.Snapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &model.Snapshot{ID: "01TESTSNAPSHOT0000000000", Path: f.path}, nil
}

type fakeRestoreRunner struct {
	calls int
	err   error
}

func (f *fakeRestoreRunner) Restore(context.Context, string, string) error {
	f.calls++
	return f.err
}

func buildRestoreZip(t *testing.T, schemaVersion int, tamper bool) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("准备目录失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", "app.dump"), []byte("dump"), 0o644); err != nil {
		t.Fatalf("写 dump 失败：%v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o755); err != nil {
		t.Fatalf("准备图片目录失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "images", "a.jpg"), []byte("image"), 0o644); err != nil {
		t.Fatalf("写图片失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.txt"), []byte("readme"), 0o644); err != nil {
		t.Fatalf("写 README 失败：%v", err)
	}
	manifest := backup.ExportManifest{
		ExportVersion: backup.ExportVersion,
		SchemaVersion: schemaVersion,
		AppVersion:    "test",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		IncludesTrash: true,
		Counts:        map[string]int64{"works": 1},
	}
	if err := backup.WriteJSON(filepath.Join(root, "manifest.json"), manifest); err != nil {
		t.Fatalf("写 manifest 失败：%v", err)
	}
	entries, err := backup.ListFiles(root, nil)
	if err != nil {
		t.Fatalf("列举文件失败：%v", err)
	}
	if err := backup.WriteChecksums(root, filepath.Join(root, backup.ChecksumFile), entries); err != nil {
		t.Fatalf("写校验和失败：%v", err)
	}
	if tamper {
		if err := os.WriteFile(filepath.Join(root, "data", "app.dump"), []byte("tampered"), 0o644); err != nil {
			t.Fatalf("篡改 dump 失败：%v", err)
		}
	}
	entries, err = backup.ListFiles(root, nil)
	if err != nil {
		t.Fatalf("列举文件失败：%v", err)
	}
	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	if err := backup.WriteZip(zipPath, root, entries); err != nil {
		t.Fatalf("写 zip 失败：%v", err)
	}
	return zipPath
}

func newTestRestore(t *testing.T, runner RestoreRunner) *Restore {
	t.Helper()
	return &Restore{
		cfg:       config.Config{DataDir: t.TempDir(), Version: "test"},
		log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		snapshots: fakeSnapshotCreator{path: filepath.Join(t.TempDir(), "snap")},
		runner:    runner,
		now:       time.Now,
	}
}

func TestPGEnvKeepsPasswordOutOfArguments(t *testing.T) {
	env, database, err := pgEnv("postgres://novel:secret@db:5432/novel?sslmode=disable")
	if err != nil {
		t.Fatalf("解析连接串失败：%v", err)
	}
	if database != "novel" {
		t.Fatalf("数据库名不符：%s", database)
	}
	want := map[string]bool{"PGHOST=db": true, "PGPORT=5432": true, "PGUSER=novel": true, "PGPASSWORD=secret": true, "PGSSLMODE=disable": true}
	for _, item := range env {
		delete(want, item)
	}
	if len(want) > 0 {
		t.Fatalf("环境变量缺少：%v", want)
	}
}

func TestRestoreArchiveRejectsNewerSchema(t *testing.T) {
	runner := &fakeRestoreRunner{}
	svc := newTestRestore(t, runner)
	zipPath := buildRestoreZip(t, 99, false)

	_, err := svc.RestoreArchive(context.Background(), zipPath, RestoreOptions{Force: true})
	if !errors.Is(err, ErrRestoreInvalid) {
		t.Fatalf("应返回 ErrRestoreInvalid，实际 %v", err)
	}
	if runner.calls != 0 {
		t.Fatal("版本不兼容时不应调用 pg_restore")
	}
}

func TestRestoreArchiveRejectsChecksumMismatch(t *testing.T) {
	runner := &fakeRestoreRunner{}
	svc := newTestRestore(t, runner)
	zipPath := buildRestoreZip(t, 10, true)

	_, err := svc.RestoreArchive(context.Background(), zipPath, RestoreOptions{Force: true})
	if !errors.Is(err, ErrRestoreInvalid) {
		t.Fatalf("应返回 ErrRestoreInvalid，实际 %v", err)
	}
	if runner.calls != 0 {
		t.Fatal("校验失败时不应调用 pg_restore")
	}
}

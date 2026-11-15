//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

type fakeRestoreRunner struct {
	calls int
	err   error
}

func (f *fakeRestoreRunner) Restore(context.Context, string, string) error {
	f.calls++
	return f.err
}

func (h *harness) restoreService(runner service.RestoreRunner) *service.Restore {
	h.t.Helper()
	return service.NewRestore(h.tx, h.cfg, discardLogger(), h.snapshots, runner)
}

func TestRestoreArchiveRestoresImagesAndWritesSnapshot(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("恢复作品")
	h.createMediaAsset(work.ID)
	run := h.runExport(true)
	zipPath := filepath.Join(h.dataDir, "exports", *run.FileName)

	imagePath := filepath.Join(h.dataDir, "images", work.ID, "test.jpg")
	if err := os.Remove(imagePath); err != nil {
		t.Fatalf("删除测试图片失败：%v", err)
	}

	runner := &fakeRestoreRunner{}
	restore := h.restoreService(runner)
	summary, err := restore.RestoreArchive(context.Background(), zipPath, service.RestoreOptions{Force: true})
	if err != nil {
		t.Fatalf("恢复失败：%v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("pg_restore 应调用 1 次，实际 %d", runner.calls)
	}
	if summary.Counts["works"] != 1 || summary.Counts["characters"] != 0 {
		t.Fatalf("恢复摘要计数不符：%+v", summary.Counts)
	}
	if _, err := os.Stat(imagePath); err != nil {
		t.Fatalf("图片未被恢复：%v", err)
	}

	settings, err := h.settings.Get(context.Background())
	if err != nil {
		t.Fatalf("读取设置失败：%v", err)
	}
	if settings.LastRestoreAt == nil {
		t.Fatal("恢复后应写入 last_restore_at")
	}

	snapshots, err := h.snapshots.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("读取快照失败：%v", err)
	}
	var restoredSnapshot *model.Snapshot
	for index := range snapshots {
		if snapshots[index].Reason == model.SnapshotReasonCLIRestore {
			restoredSnapshot = &snapshots[index]
			break
		}
	}
	if restoredSnapshot == nil {
		t.Fatal("恢复后应保留 cli_restore 快照记录")
	}
	if _, err := os.Stat(restoredSnapshot.Path); err != nil {
		t.Fatalf("恢复前快照目录不存在：%v", err)
	}

	second, err := restore.RestoreSnapshot(context.Background(), restoredSnapshot.Path, service.RestoreOptions{Force: true})
	if err != nil {
		t.Fatalf("快照恢复失败：%v", err)
	}
	if second.SnapshotPath == "" || runner.calls != 2 {
		t.Fatalf("快照恢复摘要不符：%+v，runner=%d", second, runner.calls)
	}
}

func TestRestoreRequiresForceOnNonEmptyInstance(t *testing.T) {
	h := newHarness(t)
	h.createWork("非空实例")
	run := h.runExport(true)
	zipPath := filepath.Join(h.dataDir, "exports", *run.FileName)

	runner := &fakeRestoreRunner{}
	restore := h.restoreService(runner)
	_, err := restore.RestoreArchive(context.Background(), zipPath, service.RestoreOptions{})
	if !errors.Is(err, service.ErrRestoreNotEmpty) {
		t.Fatalf("非空实例应拒绝恢复，实际 %v", err)
	}
	if runner.calls != 0 {
		t.Fatal("拒绝恢复时不应调用 pg_restore")
	}
}

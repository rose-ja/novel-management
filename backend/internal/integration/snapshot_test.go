//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"novel-management/backend/internal/model"
)

func TestSnapshotRetentionKeepsLatestThree(t *testing.T) {
	h := newHarness(t)

	for i := 0; i < 4; i++ {
		if _, err := h.snapshots.Create(context.Background(), model.SnapshotReasonWorkDelete); err != nil {
			t.Fatalf("第 %d 次快照失败：%v", i+1, err)
		}
	}

	items, err := h.snapshots.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("读取快照记录失败：%v", err)
	}
	if len(items) != 3 {
		t.Fatalf("保留策略应只留 3 条记录，实际 %d", len(items))
	}

	entries, err := os.ReadDir(filepath.Join(h.dataDir, "snapshots"))
	if err != nil {
		t.Fatalf("读取快照目录失败：%v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("目录数应与记录一致（3），实际 %d", len(entries))
	}

	for _, item := range items {
		for _, name := range []string{"app.dump", "images", "snapshot.json", "manifest.json", "checksums.sha256"} {
			if _, err := os.Stat(filepath.Join(item.Path, name)); err != nil {
				t.Fatalf("快照 %s 缺少 %s：%v", item.ID, name, err)
			}
		}
	}
}

func TestSnapshotFailureIsRecordedAndRejected(t *testing.T) {
	h := newHarness(t)
	h.dumper.fail = true

	if _, err := h.snapshots.Create(context.Background(), model.SnapshotReasonWorkDelete); err == nil {
		t.Fatal("快照失败应返回错误")
	} else if !errors.Is(err, model.ErrSnapshotFailed) {
		t.Fatalf("错误应为 ErrSnapshotFailed，实际 %v", err)
	}

	var failed int64
	if err := h.tx.Table("snapshots").Where("status = ?", model.SnapshotStatusFailed).Count(&failed).Error; err != nil {
		t.Fatalf("统计失败快照失败：%v", err)
	}
	if failed != 1 {
		t.Fatalf("应记录 1 条失败快照，实际 %d", failed)
	}
}

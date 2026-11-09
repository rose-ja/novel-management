//go:build integration

package integration

import (
	"context"
	"testing"

	"novel-management/backend/internal/migrations"
)

func tableExists(t *testing.T, name string) bool {
	t.Helper()
	var exists bool
	err := testDB.Raw(
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = ?)",
		name,
	).Scan(&exists).Error
	if err != nil {
		t.Fatalf("检查表 %s 失败：%v", name, err)
	}
	return exists
}

func TestMigrationRoundTrip(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败：%v", err)
	}

	tables := []string{
		"works", "system_categories", "system_nodes", "character_system_refs",
		"characters", "character_names",
		"app_settings", "snapshots", "undo_tokens",
	}
	for _, table := range tables {
		if !tableExists(t, table) {
			t.Fatalf("迁移 up 后缺少表 %s", table)
		}
	}

	var version int64
	if err := testDB.Raw("SELECT max(version_id) FROM goose_db_version").Scan(&version).Error; err != nil {
		t.Fatalf("读取 goose_db_version 失败：%v", err)
	}
	if version != 4 {
		t.Fatalf("迁移版本应为 4，实际 %d", version)
	}

	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0004 失败：%v", err)
	}
	for _, table := range []string{"characters", "character_names"} {
		if tableExists(t, table) {
			t.Fatalf("回滚 0004 后表 %s 仍然存在", table)
		}
	}
	if !tableExists(t, "character_system_refs") {
		t.Fatal("回滚 0004 不应删除表 character_system_refs")
	}

	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0003 失败：%v", err)
	}
	for _, table := range []string{"system_nodes", "character_system_refs"} {
		if tableExists(t, table) {
			t.Fatalf("回滚 0003 后表 %s 仍然存在", table)
		}
	}
	for _, table := range []string{"works", "system_categories"} {
		if !tableExists(t, table) {
			t.Fatalf("回滚 0003 不应删除表 %s", table)
		}
	}

	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0002 失败：%v", err)
	}
	for _, table := range tables {
		if tableExists(t, table) {
			t.Fatalf("回滚后表 %s 仍然存在", table)
		}
	}

	if err := migrations.Up(ctx, sqlDB); err != nil {
		t.Fatalf("重新迁移失败：%v", err)
	}
	if err := migrations.Up(ctx, sqlDB); err != nil {
		t.Fatalf("重复迁移失败：%v", err)
	}
	for _, table := range tables {
		if !tableExists(t, table) {
			t.Fatalf("重新 up 后缺少表 %s", table)
		}
	}

	var settings int64
	if err := testDB.Table("app_settings").Where("id = 1").Count(&settings).Error; err != nil {
		t.Fatalf("检查 app_settings 失败：%v", err)
	}
	if settings != 1 {
		t.Fatalf("app_settings 应有且只有 1 行，实际 %d", settings)
	}
}

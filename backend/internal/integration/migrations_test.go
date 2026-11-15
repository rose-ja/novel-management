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

func columnExists(t *testing.T, table, column string) bool {
	t.Helper()
	var exists bool
	err := testDB.Raw(
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = ? AND column_name = ?)",
		table, column,
	).Scan(&exists).Error
	if err != nil {
		t.Fatalf("检查列 %s.%s 失败：%v", table, column, err)
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
		"characters", "character_names", "media_assets", "chronicles", "factions",
		"locations", "character_locations", "location_factions",
		"rankings", "ranking_seats", "search_documents", "export_runs",
		"app_settings", "snapshots", "undo_tokens",
	}
	for _, table := range tables {
		if !tableExists(t, table) {
			t.Fatalf("迁移 up 后缺少表 %s", table)
		}
	}
	if columnExists(t, "characters", "image_path") {
		t.Fatal("0005 后不应保留 characters.image_path")
	}

	var version int64
	if err := testDB.Raw("SELECT max(version_id) FROM goose_db_version").Scan(&version).Error; err != nil {
		t.Fatalf("读取 goose_db_version 失败：%v", err)
	}
	if version != migrations.LatestVersion {
		t.Fatalf("迁移版本应为 %d，实际 %d", migrations.LatestVersion, version)
	}

	seed := []string{
		`INSERT INTO works (id, name, name_key) VALUES ('MIGWORK', '迁移作品', '迁移作品')`,
		`INSERT INTO system_categories (id, work_id, name, name_key) VALUES ('MIGCAT', 'MIGWORK', '迁移类别', '迁移类别')`,
		`INSERT INTO system_nodes (id, work_id, category_id, name, name_key) VALUES ('MIGNODE', 'MIGWORK', 'MIGCAT', '迁移节点', '迁移节点')`,
		`INSERT INTO characters (id, work_id, name, name_key) VALUES ('MIGCHAR', 'MIGWORK', '迁移人物', '迁移人物')`,
		`INSERT INTO character_system_refs (id, work_id, character_id, category_id, node_id) VALUES ('MIGREF', 'MIGWORK', 'MIGCHAR', 'MIGCAT', 'MIGNODE')`,
		`INSERT INTO media_assets (id, work_id, entity_type, entity_id, original_path, mime, byte_size, sha256) VALUES ('MIGMEDIA', 'MIGWORK', 'character', 'MIGCHAR', 'images/MIGWORK/MIGMEDIA.jpg', 'image/jpeg', 1, repeat('a', 64))`,
		`INSERT INTO chronicles (id, work_id, owner_type, owner_id, text) VALUES ('MIGCHRON', 'MIGWORK', 'character', 'MIGCHAR', '迁移大事记')`,
		`INSERT INTO factions (id, work_id, name, name_key) VALUES ('MIGFAC', 'MIGWORK', '迁移势力', '迁移势力')`,
		`UPDATE characters SET faction_id = 'MIGFAC' WHERE id = 'MIGCHAR'`,
		`INSERT INTO chronicles (id, work_id, owner_type, owner_id, text) VALUES ('MIGFACHRON', 'MIGWORK', 'faction', 'MIGFAC', '迁移势力大事记')`,
		`INSERT INTO locations (id, work_id, name, name_key) VALUES ('MIGLOC', 'MIGWORK', '迁移地点', '迁移地点')`,
		`INSERT INTO character_locations (id, work_id, character_id, location_id, note) VALUES ('MIGCL', 'MIGWORK', 'MIGCHAR', 'MIGLOC', '出生地')`,
		`INSERT INTO location_factions (id, work_id, location_id, faction_id, note) VALUES ('MIGLF', 'MIGWORK', 'MIGLOC', 'MIGFAC', '驻点')`,
		`INSERT INTO rankings (id, work_id, name, name_key, seat_count) VALUES ('MIGRANK', 'MIGWORK', '迁移榜单', '迁移榜单', 3)`,
		`INSERT INTO ranking_seats (id, work_id, ranking_id, seat_no, character_id) VALUES ('MIGSEAT', 'MIGWORK', 'MIGRANK', 1, 'MIGCHAR')`,
	}
	for _, statement := range seed {
		if err := testDB.Exec(statement).Error; err != nil {
			t.Fatalf("播种迁移回滚数据失败：%v", err)
		}
	}

	// 0010 Down：删除导出任务表；检索文档保留。
	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0010 失败：%v", err)
	}
	if tableExists(t, "export_runs") {
		t.Fatal("回滚 0010 后表 export_runs 仍然存在")
	}
	if !tableExists(t, "search_documents") {
		t.Fatal("回滚 0010 不应删除检索文档表")
	}

	// 0009 Down：删除检索文档；其他表保留。
	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0009 失败：%v", err)
	}
	if tableExists(t, "search_documents") {
		t.Fatal("回滚 0009 后表 search_documents 仍然存在")
	}
	if !tableExists(t, "rankings") {
		t.Fatal("回滚 0009 不应删除榜单表")
	}

	// 0008 Down：删除榜单与席位；人物保留。
	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0008 失败：%v", err)
	}
	for _, table := range []string{"ranking_seats", "rankings"} {
		if tableExists(t, table) {
			t.Fatalf("回滚 0008 后表 %s 仍然存在", table)
		}
	}
	if !tableExists(t, "characters") {
		t.Fatal("回滚 0008 不应删除人物表")
	}

	// 0007 Down：删除地点与两类关系表；人物/势力本体保留。
	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0007 失败：%v", err)
	}
	for _, table := range []string{"location_factions", "character_locations", "locations"} {
		if tableExists(t, table) {
			t.Fatalf("回滚 0007 后表 %s 仍然存在", table)
		}
	}
	if !tableExists(t, "factions") || !tableExists(t, "characters") {
		t.Fatal("回滚 0007 不应删除势力或人物表")
	}

	// 0006 Down：删除势力与势力大事记，恢复 chronicles 仅 character，人物 faction_id 列值保留。
	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0006 失败：%v", err)
	}
	if tableExists(t, "factions") {
		t.Fatal("回滚 0006 后表 factions 仍然存在")
	}
	if !columnExists(t, "characters", "faction_id") {
		t.Fatal("回滚 0006 不应删除 characters.faction_id")
	}
	var keptFactionID string
	if err := testDB.Raw("SELECT faction_id FROM characters WHERE id = 'MIGCHAR'").Scan(&keptFactionID).Error; err != nil {
		t.Fatalf("读取 characters.faction_id 失败：%v", err)
	}
	if keptFactionID != "MIGFAC" {
		t.Fatalf("回滚 0006 后人物 faction_id 应保留 MIGFAC，实际 %q", keptFactionID)
	}
	var factionChronicles int64
	if err := testDB.Raw("SELECT count(*) FROM chronicles WHERE owner_type = 'faction'").Scan(&factionChronicles).Error; err != nil {
		t.Fatalf("统计势力大事记失败：%v", err)
	}
	if factionChronicles != 0 {
		t.Fatalf("回滚 0006 后势力大事记应清空，实际 %d 行", factionChronicles)
	}
	if err := testDB.Exec(`INSERT INTO chronicles (id, work_id, owner_type, owner_id, text) VALUES ('MIGFACHRON2', 'MIGWORK', 'faction', 'MIGFAC', '不应写入')`).Error; err == nil {
		t.Fatal("回滚 0006 后 chronicles.owner_type 应恢复为仅 character")
	}

	// 0005 Down：删除图片/大事记表，恢复 characters.image_path，人物与引用保留。
	if err := migrations.Down(ctx, sqlDB); err != nil {
		t.Fatalf("回滚 0005 失败：%v", err)
	}
	for _, table := range []string{"media_assets", "chronicles"} {
		if tableExists(t, table) {
			t.Fatalf("回滚 0005 后表 %s 仍然存在", table)
		}
	}
	if !columnExists(t, "characters", "image_path") {
		t.Fatal("回滚 0005 后应恢复 characters.image_path")
	}
	if !tableExists(t, "characters") {
		t.Fatal("回滚 0005 不应删除 characters")
	}

	// 0004 Down：清空引用并删除人物/别名。
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
	var leftoverRefs int64
	if err := testDB.Raw("SELECT count(*) FROM character_system_refs").Scan(&leftoverRefs).Error; err != nil {
		t.Fatalf("统计 character_system_refs 失败：%v", err)
	}
	if leftoverRefs != 0 {
		t.Fatalf("回滚 0004 后 character_system_refs 应清空，实际 %d 行", leftoverRefs)
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

//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/search"
)

// TestSearchRefresherBuildsAndCleansDocuments 验证五类文档、关联聚合、软删除与物理删除清理。
func TestSearchRefresherBuildsAndCleansDocuments(t *testing.T) {
	h := newHarness(t)
	work := h.createWork("检索刷新作品")

	seed := []struct {
		statement string
		args      []any
	}{
		{`INSERT INTO system_categories (id, work_id, name, name_key, category_type, is_builtin)
		 VALUES ('RFCAT', ?, '检索境界', '检索境界', 'normal', false)`, []any{work.ID}},
		{`INSERT INTO system_nodes (id, work_id, category_id, name, name_key)
		 VALUES ('RFNODE', ?, 'RFCAT', '圣者境', '圣者境')`, []any{work.ID}},
		{`INSERT INTO factions (id, work_id, name, name_key)
		 VALUES ('RFFAC', ?, '山海妖族', '山海妖族')`, []any{work.ID}},
		{`INSERT INTO locations (id, work_id, name, name_key, type_text, description)
		 VALUES ('RFLOC', ?, '青丘', '青丘', '山域', '狐族祖地')`, []any{work.ID}},
		{`INSERT INTO rankings (id, work_id, name, name_key, seat_count)
		 VALUES ('RFRANK', ?, '天榜', '天榜', 10)`, []any{work.ID}},
		{`INSERT INTO characters (id, work_id, name, name_key, description, faction_id)
		 VALUES ('RFCHAR', ?, '司空摘天', '司空摘天', '摘星者', 'RFFAC')`, []any{work.ID}},
		{`INSERT INTO characters (id, work_id, name, name_key)
		 VALUES ('RFOTHER', ?, '路人甲', '路人甲')`, []any{work.ID}},
		{`INSERT INTO character_names (id, character_id, work_id, kind, value, value_key, sort_order)
		 VALUES ('RFN1', 'RFCHAR', ?, 'alias', '小圣', '小圣', 0),
		        ('RFN2', 'RFCHAR', ?, 'alias', '摘天', '摘天', 1),
		        ('RFN3', 'RFCHAR', ?, 'title', '神逆', '神逆', 2)`, []any{work.ID, work.ID, work.ID}},
		{`INSERT INTO character_system_refs (id, work_id, character_id, category_id, node_id)
		 VALUES ('RFREF', ?, 'RFCHAR', 'RFCAT', 'RFNODE')`, []any{work.ID}},
		{`INSERT INTO character_locations (id, work_id, character_id, location_id, note)
		 VALUES ('RFCL', ?, 'RFCHAR', 'RFLOC', '出生地')`, []any{work.ID}},
		{`INSERT INTO location_factions (id, work_id, location_id, faction_id, note)
		 VALUES ('RFLF', ?, 'RFLOC', 'RFFAC', '驻点')`, []any{work.ID}},
		{`INSERT INTO ranking_seats (id, work_id, ranking_id, seat_no, character_id)
		 VALUES ('RFSEAT', ?, 'RFRANK', 1, 'RFCHAR')`, []any{work.ID}},
	}
	for _, item := range seed {
		if err := h.tx.Exec(item.statement, item.args...).Error; err != nil {
			t.Fatalf("播种检索数据失败：%v", err)
		}
	}
	if err := h.tx.Exec(`UPDATE factions SET leader_character_id = 'RFCHAR' WHERE id = 'RFFAC'`).Error; err != nil {
		t.Fatalf("设置势力首领失败：%v", err)
	}

	ctx := context.Background()
	refresher := search.NewRefresher(h.tx)
	if err := refresher.RefreshWork(ctx, work.ID); err != nil {
		t.Fatalf("刷新检索文档失败：%v", err)
	}

	docs, err := loadSearchDocs(h, work.ID)
	if err != nil {
		t.Fatalf("读取检索文档失败：%v", err)
	}
	if len(docs) != 6 {
		t.Fatalf("应有 6 条文档（2 人物 + 势力 + 地点 + 榜单 + 节点），实际 %d", len(docs))
	}
	character := docs[model.SearchEntityCharacter+":RFCHAR"]
	if character == nil {
		t.Fatal("缺少人物检索文档")
	}
	assertContains(t, "别名", character.AliasText, "小圣", "摘天")
	assertContains(t, "称号", character.TitleText, "神逆")
	assertContains(t, "人物关联", character.RelationText, "山海妖族", "青丘", "天榜", "圣者境", "检索境界")
	if character.IsDeleted {
		t.Fatal("活跃人物文档不应标记删除")
	}
	assertContains(t, "势力关联", docs[model.SearchEntityFaction+":RFFAC"].RelationText, "司空摘天")
	assertContains(t, "地点关联", docs[model.SearchEntityLocation+":RFLOC"].RelationText, "司空摘天", "山海妖族")
	assertContains(t, "榜单关联", docs[model.SearchEntityRanking+":RFRANK"].RelationText, "司空摘天")
	assertContains(t, "节点关联", docs[model.SearchEntitySystemNode+":RFNODE"].RelationText, "检索境界")

	// 软删除后文档保留但 is_deleted=true。
	if err := h.tx.Exec(
		`UPDATE characters SET deleted_at = now(), deleted_root_id = id WHERE id = 'RFCHAR'`,
	).Error; err != nil {
		t.Fatalf("软删除人物失败：%v", err)
	}
	if err := refresher.RefreshWork(ctx, work.ID); err != nil {
		t.Fatalf("软删除后刷新失败：%v", err)
	}
	docs, err = loadSearchDocs(h, work.ID)
	if err != nil {
		t.Fatalf("读取检索文档失败：%v", err)
	}
	if !docs[model.SearchEntityCharacter+":RFCHAR"].IsDeleted {
		t.Fatal("软删除人物文档应标记 is_deleted=true")
	}

	// 恢复后 is_deleted=false。
	if err := h.tx.Exec(
		`UPDATE characters SET deleted_at = NULL, deleted_root_id = NULL WHERE id = 'RFCHAR'`,
	).Error; err != nil {
		t.Fatalf("恢复人物失败：%v", err)
	}
	if err := refresher.RefreshWork(ctx, work.ID); err != nil {
		t.Fatalf("恢复后刷新失败：%v", err)
	}
	docs, err = loadSearchDocs(h, work.ID)
	if err != nil {
		t.Fatalf("读取检索文档失败：%v", err)
	}
	if docs[model.SearchEntityCharacter+":RFCHAR"].IsDeleted {
		t.Fatal("恢复人物文档应标记 is_deleted=false")
	}

	// 物理删除人物后文档应被清理。
	if err := h.tx.Exec(`UPDATE factions SET leader_character_id = NULL WHERE id = 'RFFAC'`).Error; err != nil {
		t.Fatalf("解除势力首领失败：%v", err)
	}
	if err := h.tx.Exec(`DELETE FROM ranking_seats WHERE character_id = 'RFCHAR'`).Error; err != nil {
		t.Fatalf("删除席位失败：%v", err)
	}
	if err := h.tx.Exec(`DELETE FROM characters WHERE id = 'RFCHAR'`).Error; err != nil {
		t.Fatalf("物理删除人物失败：%v", err)
	}
	if err := refresher.RefreshWork(ctx, work.ID); err != nil {
		t.Fatalf("物理删除后刷新失败：%v", err)
	}
	docs, err = loadSearchDocs(h, work.ID)
	if err != nil {
		t.Fatalf("读取检索文档失败：%v", err)
	}
	if docs[model.SearchEntityCharacter+":RFCHAR"] != nil {
		t.Fatal("物理删除后人物文档应被清理")
	}
	if len(docs) != 5 {
		t.Fatalf("清理后应有 5 条文档，实际 %d", len(docs))
	}

	consistency, err := refresher.Consistency(ctx, work.ID)
	if err != nil {
		t.Fatalf("一致性检查失败：%v", err)
	}
	if consistency.Drift != 0 {
		t.Fatalf("一致性漂移应为 0，实际 %d（%+v）", consistency.Drift, consistency.ByType)
	}
}

func loadSearchDocs(h *harness, workID string) (map[string]*model.SearchDocument, error) {
	var rows []model.SearchDocument
	if err := h.tx.Where("work_id = ?", workID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]*model.SearchDocument, len(rows))
	for i := range rows {
		out[rows[i].EntityType+":"+rows[i].EntityID] = &rows[i]
	}
	return out, nil
}

func assertContains(t *testing.T, label string, value *string, expected ...string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s 文本为空，期望包含 %s", label, strings.Join(expected, "、"))
	}
	for _, item := range expected {
		if !strings.Contains(*value, item) {
			t.Fatalf("%s 文本 %q 缺少 %q", label, *value, item)
		}
	}
}

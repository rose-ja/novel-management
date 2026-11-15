// Package search 维护与查询 search_documents 检索文档。
package search

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
)

// Refresher 在同一事务内重建检索文档；重建粒度为整个作品。
type Refresher struct {
	db *gorm.DB
}

func NewRefresher(db *gorm.DB) *Refresher {
	return &Refresher{db: db}
}

// RefreshWork 重建指定作品的全部检索文档，并清理源实体已物理不存在的水位。
func (r *Refresher) RefreshWork(ctx context.Context, workID string) error {
	if workID == "" {
		return nil
	}
	db := r.db.WithContext(ctx)
	for _, statement := range refreshStatements {
		if err := db.Exec(statement, workID).Error; err != nil {
			return fmt.Errorf("刷新检索文档失败：%w", err)
		}
	}
	if err := db.Exec(cleanupStatement, workID).Error; err != nil {
		return fmt.Errorf("清理检索文档失败：%w", err)
	}
	return nil
}

// RebuildAll 按作品逐个全量重建；供启动与 CLI reindex 使用。
func (r *Refresher) RebuildAll(ctx context.Context) (int, error) {
	var workIDs []string
	if err := r.db.WithContext(ctx).Model(&model.Work{}).Order("id").Pluck("id", &workIDs).Error; err != nil {
		return 0, err
	}
	for _, workID := range workIDs {
		if err := r.RefreshWork(ctx, workID); err != nil {
			return 0, err
		}
	}
	return len(workIDs), nil
}

// Consistency 比较文档数与源实体数；deleted 实体也计入，漂移即代表文档缺失或残留。
func (r *Refresher) Consistency(ctx context.Context, workID string) (*model.SearchConsistency, error) {
	db := r.db.WithContext(ctx)
	docs := make(map[string]int64)
	var docRows []struct {
		EntityType string
		Count      int64
	}
	if err := db.Model(&model.SearchDocument{}).
		Select("entity_type, count(*) AS count").
		Where("work_id = ?", workID).
		Group("entity_type").
		Scan(&docRows).Error; err != nil {
		return nil, err
	}
	for _, row := range docRows {
		docs[row.EntityType] = row.Count
	}

	entities := make(map[string]int64)
	sources := []struct {
		entityType string
		table      string
	}{
		{model.SearchEntityCharacter, "characters"},
		{model.SearchEntityFaction, "factions"},
		{model.SearchEntityLocation, "locations"},
		{model.SearchEntityRanking, "rankings"},
		{model.SearchEntitySystemNode, "system_nodes"},
	}
	for _, source := range sources {
		var count int64
		if err := db.Table(source.table).Where("work_id = ?", workID).Count(&count).Error; err != nil {
			return nil, err
		}
		entities[source.entityType] = count
	}

	result := &model.SearchConsistency{WorkID: workID}
	for _, entityType := range model.SearchEntityTypes() {
		stat := model.SearchTypeStat{
			EntityType: entityType,
			Docs:       docs[entityType],
			Entities:   entities[entityType],
		}
		result.ByType = append(result.ByType, stat)
		result.TotalDocs += stat.Docs
		result.TotalEntities += stat.Entities
		result.Drift += stat.Docs - stat.Entities
	}
	return result, nil
}

var refreshStatements = []string{
	refreshCharactersSQL,
	refreshFactionsSQL,
	refreshLocationsSQL,
	refreshRankingsSQL,
	refreshSystemNodesSQL,
}

const refreshCharactersSQL = `
INSERT INTO search_documents (
    id, work_id, entity_type, entity_id, name_text, name_key,
    alias_text, title_text, desc_text, relation_text,
    review_status, updated_at, is_deleted
)
SELECT
    substr(replace(gen_random_uuid()::text, '-', ''), 1, 26),
    c.work_id,
    'character',
    c.id,
    c.name,
    c.name_key,
    NULLIF((SELECT string_agg(cn.value, ' ' ORDER BY cn.sort_order)
            FROM character_names cn
            WHERE cn.character_id = c.id AND cn.kind = 'alias'), ''),
    NULLIF((SELECT string_agg(cn.value, ' ' ORDER BY cn.sort_order)
            FROM character_names cn
            WHERE cn.character_id = c.id AND cn.kind = 'title'), ''),
    NULLIF(c.description, ''),
    NULLIF(concat_ws(' ',
        (SELECT f.name FROM factions f
         WHERE f.id = c.faction_id AND f.deleted_at IS NULL),
        (SELECT string_agg(l.name, ' ' ORDER BY l.name)
         FROM character_locations cl
         JOIN locations l ON l.id = cl.location_id AND l.deleted_at IS NULL
         WHERE cl.character_id = c.id),
        (SELECT string_agg(r.name, ' ' ORDER BY r.name)
         FROM ranking_seats rs
         JOIN rankings r ON r.id = rs.ranking_id AND r.deleted_at IS NULL
         WHERE rs.character_id = c.id),
        (SELECT string_agg(x.name, ' ' ORDER BY x.name) FROM (
            SELECT DISTINCT sn.name
            FROM character_system_refs csr
            JOIN system_nodes sn ON sn.id = csr.node_id AND sn.deleted_at IS NULL
            WHERE csr.character_id = c.id
            UNION
            SELECT DISTINCT pn.name
            FROM character_system_refs csr
            JOIN system_nodes sn ON sn.id = csr.node_id AND sn.deleted_at IS NULL
            JOIN system_nodes pn ON pn.id = sn.parent_id AND pn.deleted_at IS NULL
            WHERE csr.character_id = c.id
            UNION
            SELECT DISTINCT sc.name
            FROM character_system_refs csr
            JOIN system_categories sc ON sc.id = csr.category_id AND sc.deleted_at IS NULL
            WHERE csr.character_id = c.id
        ) x)
    ), ''),
    c.review_status,
    c.updated_at,
    (c.deleted_at IS NOT NULL OR w.deleted_at IS NOT NULL)
FROM characters c
JOIN works w ON w.id = c.work_id
WHERE c.work_id = $1
ON CONFLICT (work_id, entity_type, entity_id) DO UPDATE SET
    name_text = EXCLUDED.name_text,
    name_key = EXCLUDED.name_key,
    alias_text = EXCLUDED.alias_text,
    title_text = EXCLUDED.title_text,
    desc_text = EXCLUDED.desc_text,
    relation_text = EXCLUDED.relation_text,
    review_status = EXCLUDED.review_status,
    updated_at = EXCLUDED.updated_at,
    is_deleted = EXCLUDED.is_deleted
WHERE search_documents.name_text IS DISTINCT FROM EXCLUDED.name_text
   OR search_documents.name_key IS DISTINCT FROM EXCLUDED.name_key
   OR search_documents.alias_text IS DISTINCT FROM EXCLUDED.alias_text
   OR search_documents.title_text IS DISTINCT FROM EXCLUDED.title_text
   OR search_documents.desc_text IS DISTINCT FROM EXCLUDED.desc_text
   OR search_documents.relation_text IS DISTINCT FROM EXCLUDED.relation_text
   OR search_documents.review_status IS DISTINCT FROM EXCLUDED.review_status
   OR search_documents.updated_at IS DISTINCT FROM EXCLUDED.updated_at
   OR search_documents.is_deleted IS DISTINCT FROM EXCLUDED.is_deleted
`

const refreshFactionsSQL = `
INSERT INTO search_documents (
    id, work_id, entity_type, entity_id, name_text, name_key,
    alias_text, title_text, desc_text, relation_text,
    review_status, updated_at, is_deleted
)
SELECT
    substr(replace(gen_random_uuid()::text, '-', ''), 1, 26),
    f.work_id,
    'faction',
    f.id,
    f.name,
    f.name_key,
    NULL,
    NULL,
    NULLIF(f.description, ''),
    NULLIF(concat_ws(' ',
        (SELECT p.name FROM factions p
         WHERE p.id = f.parent_id AND p.deleted_at IS NULL),
        (SELECT c.name FROM characters c
         WHERE c.id = f.leader_character_id AND c.deleted_at IS NULL)
    ), ''),
    f.review_status,
    f.updated_at,
    (f.deleted_at IS NOT NULL OR w.deleted_at IS NOT NULL)
FROM factions f
JOIN works w ON w.id = f.work_id
WHERE f.work_id = $1
ON CONFLICT (work_id, entity_type, entity_id) DO UPDATE SET
    name_text = EXCLUDED.name_text,
    name_key = EXCLUDED.name_key,
    desc_text = EXCLUDED.desc_text,
    relation_text = EXCLUDED.relation_text,
    review_status = EXCLUDED.review_status,
    updated_at = EXCLUDED.updated_at,
    is_deleted = EXCLUDED.is_deleted
WHERE search_documents.name_text IS DISTINCT FROM EXCLUDED.name_text
   OR search_documents.name_key IS DISTINCT FROM EXCLUDED.name_key
   OR search_documents.desc_text IS DISTINCT FROM EXCLUDED.desc_text
   OR search_documents.relation_text IS DISTINCT FROM EXCLUDED.relation_text
   OR search_documents.review_status IS DISTINCT FROM EXCLUDED.review_status
   OR search_documents.updated_at IS DISTINCT FROM EXCLUDED.updated_at
   OR search_documents.is_deleted IS DISTINCT FROM EXCLUDED.is_deleted
`

const refreshLocationsSQL = `
INSERT INTO search_documents (
    id, work_id, entity_type, entity_id, name_text, name_key,
    alias_text, title_text, desc_text, relation_text,
    review_status, updated_at, is_deleted
)
SELECT
    substr(replace(gen_random_uuid()::text, '-', ''), 1, 26),
    l.work_id,
    'location',
    l.id,
    l.name,
    l.name_key,
    NULL,
    NULL,
    NULLIF(concat_ws(' ', l.type_text, l.description), ''),
    NULLIF(concat_ws(' ',
        (SELECT p.name FROM locations p
         WHERE p.id = l.parent_id AND p.deleted_at IS NULL),
        (SELECT string_agg(c.name, ' ' ORDER BY c.name)
         FROM character_locations cl
         JOIN characters c ON c.id = cl.character_id AND c.deleted_at IS NULL
         WHERE cl.location_id = l.id),
        (SELECT string_agg(f.name, ' ' ORDER BY f.name)
         FROM location_factions lf
         JOIN factions f ON f.id = lf.faction_id AND f.deleted_at IS NULL
         WHERE lf.location_id = l.id)
    ), ''),
    l.review_status,
    l.updated_at,
    (l.deleted_at IS NOT NULL OR w.deleted_at IS NOT NULL)
FROM locations l
JOIN works w ON w.id = l.work_id
WHERE l.work_id = $1
ON CONFLICT (work_id, entity_type, entity_id) DO UPDATE SET
    name_text = EXCLUDED.name_text,
    name_key = EXCLUDED.name_key,
    desc_text = EXCLUDED.desc_text,
    relation_text = EXCLUDED.relation_text,
    review_status = EXCLUDED.review_status,
    updated_at = EXCLUDED.updated_at,
    is_deleted = EXCLUDED.is_deleted
WHERE search_documents.name_text IS DISTINCT FROM EXCLUDED.name_text
   OR search_documents.name_key IS DISTINCT FROM EXCLUDED.name_key
   OR search_documents.desc_text IS DISTINCT FROM EXCLUDED.desc_text
   OR search_documents.relation_text IS DISTINCT FROM EXCLUDED.relation_text
   OR search_documents.review_status IS DISTINCT FROM EXCLUDED.review_status
   OR search_documents.updated_at IS DISTINCT FROM EXCLUDED.updated_at
   OR search_documents.is_deleted IS DISTINCT FROM EXCLUDED.is_deleted
`

const refreshRankingsSQL = `
INSERT INTO search_documents (
    id, work_id, entity_type, entity_id, name_text, name_key,
    alias_text, title_text, desc_text, relation_text,
    review_status, updated_at, is_deleted
)
SELECT
    substr(replace(gen_random_uuid()::text, '-', ''), 1, 26),
    r.work_id,
    'ranking',
    r.id,
    r.name,
    r.name_key,
    NULL,
    NULL,
    NULL,
    NULLIF((SELECT string_agg(c.name, ' ' ORDER BY rs.seat_no)
            FROM ranking_seats rs
            JOIN characters c ON c.id = rs.character_id AND c.deleted_at IS NULL
            WHERE rs.ranking_id = r.id), ''),
    r.review_status,
    r.updated_at,
    (r.deleted_at IS NOT NULL OR w.deleted_at IS NOT NULL)
FROM rankings r
JOIN works w ON w.id = r.work_id
WHERE r.work_id = $1
ON CONFLICT (work_id, entity_type, entity_id) DO UPDATE SET
    name_text = EXCLUDED.name_text,
    name_key = EXCLUDED.name_key,
    relation_text = EXCLUDED.relation_text,
    review_status = EXCLUDED.review_status,
    updated_at = EXCLUDED.updated_at,
    is_deleted = EXCLUDED.is_deleted
WHERE search_documents.name_text IS DISTINCT FROM EXCLUDED.name_text
   OR search_documents.name_key IS DISTINCT FROM EXCLUDED.name_key
   OR search_documents.relation_text IS DISTINCT FROM EXCLUDED.relation_text
   OR search_documents.review_status IS DISTINCT FROM EXCLUDED.review_status
   OR search_documents.updated_at IS DISTINCT FROM EXCLUDED.updated_at
   OR search_documents.is_deleted IS DISTINCT FROM EXCLUDED.is_deleted
`

const refreshSystemNodesSQL = `
INSERT INTO search_documents (
    id, work_id, entity_type, entity_id, name_text, name_key,
    alias_text, title_text, desc_text, relation_text,
    review_status, updated_at, is_deleted
)
SELECT
    substr(replace(gen_random_uuid()::text, '-', ''), 1, 26),
    n.work_id,
    'system_node',
    n.id,
    n.name,
    n.name_key,
    NULL,
    NULL,
    NULL,
    NULLIF(concat_ws(' ',
        (SELECT sc.name FROM system_categories sc
         WHERE sc.id = n.category_id AND sc.deleted_at IS NULL),
        (SELECT p.name FROM system_nodes p
         WHERE p.id = n.parent_id AND p.deleted_at IS NULL)
    ), ''),
    n.review_status,
    n.updated_at,
    (n.deleted_at IS NOT NULL OR w.deleted_at IS NOT NULL OR sc.deleted_at IS NOT NULL)
FROM system_nodes n
JOIN works w ON w.id = n.work_id
JOIN system_categories sc ON sc.id = n.category_id
WHERE n.work_id = $1
ON CONFLICT (work_id, entity_type, entity_id) DO UPDATE SET
    name_text = EXCLUDED.name_text,
    name_key = EXCLUDED.name_key,
    relation_text = EXCLUDED.relation_text,
    review_status = EXCLUDED.review_status,
    updated_at = EXCLUDED.updated_at,
    is_deleted = EXCLUDED.is_deleted
WHERE search_documents.name_text IS DISTINCT FROM EXCLUDED.name_text
   OR search_documents.name_key IS DISTINCT FROM EXCLUDED.name_key
   OR search_documents.relation_text IS DISTINCT FROM EXCLUDED.relation_text
   OR search_documents.review_status IS DISTINCT FROM EXCLUDED.review_status
   OR search_documents.updated_at IS DISTINCT FROM EXCLUDED.updated_at
   OR search_documents.is_deleted IS DISTINCT FROM EXCLUDED.is_deleted
`

const cleanupStatement = `
DELETE FROM search_documents d
WHERE d.work_id = $1
  AND NOT EXISTS (
      SELECT 1 FROM characters x
      WHERE d.entity_type = 'character' AND x.id = d.entity_id
      UNION ALL
      SELECT 1 FROM factions x
      WHERE d.entity_type = 'faction' AND x.id = d.entity_id
      UNION ALL
      SELECT 1 FROM locations x
      WHERE d.entity_type = 'location' AND x.id = d.entity_id
      UNION ALL
      SELECT 1 FROM rankings x
      WHERE d.entity_type = 'ranking' AND x.id = d.entity_id
      UNION ALL
      SELECT 1 FROM system_nodes x
      WHERE d.entity_type = 'system_node' AND x.id = d.entity_id
  )
`

package search

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
)

// Filters 是查询层的已校验筛选条件。
type Filters struct {
	EntityTypes      []string
	FactionID        *string
	FactionScope     string
	SystemCategoryID *string
	SystemNodeID     *string
	LocationID       *string
	RankingID        *string
	ReviewStatus     *string
}

// Query 是一次检索查询。
type Query struct {
	WorkID  string
	Tokens  []string
	Filters Filters
}

// FindMatches 返回全部命中文档（未分页），按 updated_at DESC, id ASC 稳定排序。
func FindMatches(ctx context.Context, db *gorm.DB, q Query) ([]model.SearchDocument, error) {
	where := []string{"d.work_id = ?", "d.is_deleted = false"}
	args := []any{q.WorkID}

	if len(q.Filters.EntityTypes) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(q.Filters.EntityTypes)), ",")
		where = append(where, "d.entity_type IN ("+placeholders+")")
		for _, entityType := range q.Filters.EntityTypes {
			args = append(args, entityType)
		}
	}

	for _, token := range q.Tokens {
		pattern := likePattern(token)
		where = append(where, `(d.name_text ILIKE ? ESCAPE '\'
			OR COALESCE(d.alias_text, '') ILIKE ? ESCAPE '\'
			OR COALESCE(d.title_text, '') ILIKE ? ESCAPE '\'
			OR COALESCE(d.desc_text, '') ILIKE ? ESCAPE '\'
			OR COALESCE(d.relation_text, '') ILIKE ? ESCAPE '\')`)
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}

	if q.Filters.FactionID != nil {
		if q.Filters.FactionScope == model.SearchFactionScopeSubtree {
			where = append(where, `(d.entity_type <> 'character' OR EXISTS (
				SELECT 1 FROM characters c
				WHERE c.id = d.entity_id AND c.faction_id IN (
					WITH RECURSIVE faction_subtree AS (
						SELECT id FROM factions WHERE id = ? AND work_id = ? AND deleted_at IS NULL
						UNION ALL
						SELECT f.id FROM factions f
						JOIN faction_subtree fs ON f.parent_id = fs.id
						WHERE f.deleted_at IS NULL
					)
					SELECT id FROM faction_subtree
				)
			))`)
			args = append(args, *q.Filters.FactionID, q.WorkID)
		} else {
			where = append(where, `(d.entity_type <> 'character' OR EXISTS (
				SELECT 1 FROM characters c WHERE c.id = d.entity_id AND c.faction_id = ?
			))`)
			args = append(args, *q.Filters.FactionID)
		}
	}
	if q.Filters.SystemCategoryID != nil {
		where = append(where, `(d.entity_type <> 'character' OR EXISTS (
			SELECT 1 FROM character_system_refs csr
			WHERE csr.character_id = d.entity_id AND csr.category_id = ?
		))`)
		args = append(args, *q.Filters.SystemCategoryID)
	}
	if q.Filters.SystemNodeID != nil {
		where = append(where, `(d.entity_type <> 'character' OR EXISTS (
			SELECT 1 FROM character_system_refs csr
			WHERE csr.character_id = d.entity_id
			  AND (csr.node_id = ? OR csr.node_id IN (
			      SELECT id FROM system_nodes WHERE parent_id = ? AND deleted_at IS NULL
			  ))
		))`)
		args = append(args, *q.Filters.SystemNodeID, *q.Filters.SystemNodeID)
	}
	if q.Filters.LocationID != nil {
		where = append(where, `(d.entity_type <> 'character' OR EXISTS (
			SELECT 1 FROM character_locations cl
			WHERE cl.character_id = d.entity_id AND cl.location_id = ?
		))`)
		args = append(args, *q.Filters.LocationID)
	}
	if q.Filters.RankingID != nil {
		where = append(where, `(d.entity_type <> 'character' OR EXISTS (
			SELECT 1 FROM ranking_seats rs
			WHERE rs.character_id = d.entity_id AND rs.ranking_id = ?
		))`)
		args = append(args, *q.Filters.RankingID)
	}
	if q.Filters.ReviewStatus != nil {
		where = append(where, "d.review_status = ?")
		args = append(args, *q.Filters.ReviewStatus)
	}

	statement := `SELECT d.* FROM search_documents d WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY d.updated_at DESC, d.id ASC`
	var docs []model.SearchDocument
	if err := db.WithContext(ctx).Raw(statement, args...).Scan(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// RankDocs 按名称 > 别名/称号 > 简介 > 关联实体排序，同分按 updated_at DESC、id ASC。
func RankDocs(docs []model.SearchDocument, tokens []string) {
	sort.SliceStable(docs, func(i, j int) bool {
		left, right := Score(docs[i], tokens), Score(docs[j], tokens)
		if left != right {
			return left > right
		}
		if !docs[i].UpdatedAt.Equal(docs[j].UpdatedAt) {
			return docs[i].UpdatedAt.After(docs[j].UpdatedAt)
		}
		return docs[i].ID < docs[j].ID
	})
}

// Score 返回文档的最高命中权重：名称 4、别名/称号 3、简介 2、关联实体 1。
func Score(doc model.SearchDocument, tokens []string) int {
	if len(tokens) == 0 {
		return 1
	}
	if fieldContains(doc.NameText, tokens) {
		return 4
	}
	if fieldContains(pointerText(doc.AliasText), tokens) || fieldContains(pointerText(doc.TitleText), tokens) {
		return 3
	}
	if fieldContains(pointerText(doc.DescText), tokens) {
		return 2
	}
	return 1
}

// BuildMatchedFields 返回命中字段与命中原文窗口；无关键词时返回空。
func BuildMatchedFields(doc model.SearchDocument, tokens []string) ([]model.SearchMatchedField, string) {
	if len(tokens) == 0 {
		return []model.SearchMatchedField{}, ""
	}
	fields := []struct {
		name  string
		label string
		value string
	}{
		{"name", "名称", doc.NameText},
		{"alias", "别名", pointerText(doc.AliasText)},
		{"title", "称号", pointerText(doc.TitleText)},
		{"desc", "简介", pointerText(doc.DescText)},
		{"relation", "关联实体", pointerText(doc.RelationText)},
	}
	matched := make([]model.SearchMatchedField, 0, len(fields))
	snippet := ""
	for _, field := range fields {
		text, ok := matchedWindow(field.value, tokens)
		if !ok {
			continue
		}
		matched = append(matched, model.SearchMatchedField{Field: field.name, MatchedText: text})
		if snippet == "" {
			snippet = field.label + "：" + text
		}
	}
	return matched, snippet
}

func likePattern(token string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(token) + "%"
}

func fieldContains(text string, tokens []string) bool {
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	for _, token := range tokens {
		if strings.Contains(lower, strings.ToLower(token)) {
			return true
		}
	}
	return false
}

func pointerText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// matchedWindow 返回命中词附近的原文窗口；短文本原样返回。
func matchedWindow(text string, tokens []string) (string, bool) {
	if text == "" {
		return "", false
	}
	lower := strings.ToLower(text)
	byteIndex := -1
	for _, token := range tokens {
		if idx := strings.Index(lower, strings.ToLower(token)); idx >= 0 && (byteIndex < 0 || idx < byteIndex) {
			byteIndex = idx
		}
	}
	if byteIndex < 0 {
		return "", false
	}
	runes := []rune(text)
	if len(runes) <= 60 {
		return text, true
	}
	matchRune := utf8.RuneCountInString(text[:byteIndex])
	start := matchRune - 10
	if start < 0 {
		start = 0
	}
	end := start + 60
	if end > len(runes) {
		end = len(runes)
		start = end - 60
	}
	window := string(runes[start:end])
	if start > 0 {
		window = "…" + window
	}
	if end < len(runes) {
		window += "…"
	}
	return window, true
}

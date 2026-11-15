package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
	"novel-management/backend/internal/search"
)

// Search 负责检索参数校验、命中组装与分页分组。
type Search struct {
	db         *gorm.DB
	characters *Characters
}

func NewSearch(db *gorm.DB, characters *Characters) *Search {
	return &Search{db: db, characters: characters}
}

// SearchInput 是 GET /search 的参数。
type SearchInput struct {
	WorkID           string
	Query            string
	EntityTypes      string
	FactionID        string
	FactionScope     string
	SystemCategoryID string
	SystemNodeID     string
	LocationID       string
	RankingID        string
	ReviewStatus     string
	Page             int
}

// Search 执行检索并返回分组结果。
func (s *Search) Search(ctx context.Context, in SearchInput) (*model.SearchResult, error) {
	workID := strings.TrimSpace(in.WorkID)
	if workID == "" {
		return nil, &model.ValidationError{Field: "work_id", Message: "work_id 不能为空"}
	}
	if _, err := repository.NewWorks(s.db).Get(ctx, workID); err != nil {
		return nil, err
	}

	query := strings.TrimSpace(in.Query)
	if utf8.RuneCountInString(query) > 200 {
		return nil, &model.ValidationError{Field: "q", Message: "关键词不能超过 200 个字符"}
	}
	tokens := strings.Fields(query)

	entityTypes, err := parseSearchEntityTypes(in.EntityTypes)
	if err != nil {
		return nil, err
	}
	scope := strings.TrimSpace(in.FactionScope)
	if scope == "" {
		scope = model.SearchFactionScopeDirect
	}
	if scope != model.SearchFactionScopeDirect && scope != model.SearchFactionScopeSubtree {
		return nil, &model.ValidationError{Field: "faction_scope", Message: "faction_scope 仅支持 direct 或 subtree"}
	}
	reviewStatus, err := parseSearchReviewStatus(in.ReviewStatus)
	if err != nil {
		return nil, err
	}

	filters := search.Filters{
		EntityTypes:  entityTypes,
		FactionScope: scope,
		ReviewStatus: reviewStatus,
	}
	if err := s.validateFilters(ctx, workID, in, &filters); err != nil {
		return nil, err
	}
	if len(tokens) == 0 && !hasNarrowingFilter(filters) {
		return nil, model.ErrSearchQueryRequired
	}

	page := in.Page
	if page < 1 {
		page = 1
	}

	docs, err := search.FindMatches(ctx, s.db, search.Query{WorkID: workID, Tokens: tokens, Filters: filters})
	if err != nil {
		return nil, err
	}
	search.RankDocs(docs, tokens)

	total := int64(len(docs))
	start := (page - 1) * model.SearchPageSize
	if start > len(docs) {
		start = len(docs)
	}
	end := start + model.SearchPageSize
	if end > len(docs) {
		end = len(docs)
	}
	pageDocs := docs[start:end]

	totals := make(map[string]int64, len(model.SearchEntityTypes()))
	for _, doc := range docs {
		totals[doc.EntityType]++
	}
	items, err := s.buildItems(ctx, pageDocs, tokens)
	if err != nil {
		return nil, err
	}

	groups := make([]model.SearchGroup, 0, len(totals))
	for _, entityType := range model.SearchEntityTypes() {
		if totals[entityType] == 0 {
			continue
		}
		group := model.SearchGroup{
			EntityType: entityType,
			Total:      totals[entityType],
			Items:      items[entityType],
		}
		if group.Items == nil {
			group.Items = []model.SearchItem{}
		}
		groups = append(groups, group)
	}

	apiReview := ""
	if filters.ReviewStatus != nil {
		apiReview = apiReviewStatus(*filters.ReviewStatus)
	}
	return &model.SearchResult{
		Groups:   groups,
		Total:    total,
		Page:     page,
		PageSize: model.SearchPageSize,
		Query: model.SearchQueryView{
			Q:      query,
			Tokens: tokens,
			Filters: model.SearchFilters{
				EntityTypes:      filters.EntityTypes,
				FactionID:        filters.FactionID,
				FactionScope:     filters.FactionScope,
				SystemCategoryID: filters.SystemCategoryID,
				SystemNodeID:     filters.SystemNodeID,
				LocationID:       filters.LocationID,
				RankingID:        filters.RankingID,
				ReviewStatus:     optionalString(apiReview),
			},
		},
	}, nil
}

func (s *Search) validateFilters(ctx context.Context, workID string, in SearchInput, filters *search.Filters) error {
	if raw := strings.TrimSpace(in.FactionID); raw != "" {
		faction, err := repository.NewFactions(s.db).GetActiveByWork(ctx, workID, raw)
		if err != nil {
			return invalidSearchFilter("faction_id", err)
		}
		filters.FactionID = &faction.ID
	}
	if raw := strings.TrimSpace(in.SystemCategoryID); raw != "" {
		category, err := repository.NewSystem(s.db).GetCategory(ctx, raw)
		if err != nil {
			return invalidSearchFilter("system_category_id", err)
		}
		if category.WorkID != workID {
			return invalidSearchFilter("system_category_id", model.ErrNotFound)
		}
		filters.SystemCategoryID = &category.ID
	}
	if raw := strings.TrimSpace(in.SystemNodeID); raw != "" {
		node, err := repository.NewSystem(s.db).GetNode(ctx, raw)
		if err != nil {
			return invalidSearchFilter("system_node_id", err)
		}
		if node.WorkID != workID {
			return invalidSearchFilter("system_node_id", model.ErrNotFound)
		}
		filters.SystemNodeID = &node.ID
	}
	if raw := strings.TrimSpace(in.LocationID); raw != "" {
		location, err := repository.NewLocations(s.db).Get(ctx, raw)
		if err != nil {
			return invalidSearchFilter("location_id", err)
		}
		if location.WorkID != workID {
			return invalidSearchFilter("location_id", model.ErrNotFound)
		}
		filters.LocationID = &location.ID
	}
	if raw := strings.TrimSpace(in.RankingID); raw != "" {
		ranking, err := repository.NewRankings(s.db).Get(ctx, raw)
		if err != nil {
			return invalidSearchFilter("ranking_id", err)
		}
		if ranking.WorkID != workID {
			return invalidSearchFilter("ranking_id", model.ErrNotFound)
		}
		filters.RankingID = &ranking.ID
	}
	return nil
}

func parseSearchEntityTypes(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return model.SearchEntityTypes(), nil
	}
	allowed := make(map[string]struct{}, len(model.SearchEntityTypes()))
	for _, entityType := range model.SearchEntityTypes() {
		allowed[entityType] = struct{}{}
	}
	result := make([]string, 0, len(model.SearchEntityTypes()))
	seen := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		entityType := strings.TrimSpace(part)
		if entityType == "" {
			continue
		}
		if _, ok := allowed[entityType]; !ok {
			return nil, &model.ValidationError{Field: "entity_types", Message: "entity_types 含不支持的类型"}
		}
		if _, ok := seen[entityType]; ok {
			continue
		}
		seen[entityType] = struct{}{}
		result = append(result, entityType)
	}
	if len(result) == 0 {
		return nil, &model.ValidationError{Field: "entity_types", Message: "entity_types 不能为空"}
	}
	return result, nil
}

func parseSearchReviewStatus(raw string) (*string, error) {
	switch strings.TrimSpace(raw) {
	case "":
		return nil, nil
	case "confirmed":
		value := model.ReviewStatusReviewed
		return &value, nil
	case "pending":
		value := model.ReviewStatusPending
		return &value, nil
	default:
		return nil, &model.ValidationError{Field: "review_status", Message: "review_status 仅支持 confirmed 或 pending"}
	}
}

func hasNarrowingFilter(filters search.Filters) bool {
	return filters.FactionID != nil || filters.SystemCategoryID != nil || filters.SystemNodeID != nil ||
		filters.LocationID != nil || filters.RankingID != nil || filters.ReviewStatus != nil
}

func invalidSearchFilter(field string, err error) error {
	if errors.Is(err, model.ErrNotFound) {
		return &model.ValidationError{Field: field, Message: "筛选条件已失效，请重新选择"}
	}
	return err
}

func apiReviewStatus(dbValue string) string {
	if dbValue == model.ReviewStatusReviewed {
		return "confirmed"
	}
	return dbValue
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

type factionSearchMeta struct {
	ID         string  `gorm:"column:id"`
	ParentName *string `gorm:"column:parent_name"`
	LeaderName *string `gorm:"column:leader_name"`
}

type locationSearchMeta struct {
	ID         string  `gorm:"column:id"`
	TypeText   *string `gorm:"column:type_text"`
	ParentName *string `gorm:"column:parent_name"`
}

type rankingSearchMeta struct {
	ID        string `gorm:"column:id"`
	SeatCount int    `gorm:"column:seat_count"`
	Occupied  int64  `gorm:"column:occupied"`
}

type nodeSearchMeta struct {
	ID           string  `gorm:"column:id"`
	CategoryID   string  `gorm:"column:category_id"`
	CategoryName string  `gorm:"column:category_name"`
	ParentName   *string `gorm:"column:parent_name"`
}

// buildItems 按当前页批量装配标题、副标题、高亮与链接，避免 N+1。
func (s *Search) buildItems(ctx context.Context, docs []model.SearchDocument, tokens []string) (map[string][]model.SearchItem, error) {
	result := make(map[string][]model.SearchItem, len(model.SearchEntityTypes()))
	if len(docs) == 0 {
		return result, nil
	}
	idsByType := make(map[string][]string, len(model.SearchEntityTypes()))
	for _, doc := range docs {
		idsByType[doc.EntityType] = append(idsByType[doc.EntityType], doc.EntityID)
	}

	charMeta := make(map[string]model.CharacterSummary)
	if ids := idsByType[model.SearchEntityCharacter]; len(ids) > 0 && s.characters != nil {
		characters, err := repository.NewCharacters(s.db).ListByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		summaries, err := s.characters.Summaries(ctx, characters)
		if err != nil {
			return nil, err
		}
		for _, summary := range summaries {
			charMeta[summary.ID] = summary
		}
	}
	factionMeta, err := s.loadFactionMeta(ctx, idsByType[model.SearchEntityFaction])
	if err != nil {
		return nil, err
	}
	locationMeta, err := s.loadLocationMeta(ctx, idsByType[model.SearchEntityLocation])
	if err != nil {
		return nil, err
	}
	rankingMeta, err := s.loadRankingMeta(ctx, idsByType[model.SearchEntityRanking])
	if err != nil {
		return nil, err
	}
	nodeMeta, err := s.loadNodeMeta(ctx, idsByType[model.SearchEntitySystemNode])
	if err != nil {
		return nil, err
	}

	for _, doc := range docs {
		matched, snippet := search.BuildMatchedFields(doc, tokens)
		item := model.SearchItem{
			EntityID:      doc.EntityID,
			EntityType:    doc.EntityType,
			Title:         doc.NameText,
			MatchedFields: matched,
			Snippet:       snippet,
			Score:         search.Score(doc, tokens),
			UpdatedAt:     doc.UpdatedAt,
			ReviewStatus:  apiReviewStatus(doc.ReviewStatus),
		}
		switch doc.EntityType {
		case model.SearchEntityCharacter:
			summary, ok := charMeta[doc.EntityID]
			if ok {
				item.ImageThumbURL = summary.ImageThumbURL
			}
			item.Subtitle = characterSubtitle(summary)
			item.URL = "/characters?id=" + url.QueryEscape(doc.EntityID)
		case model.SearchEntityFaction:
			meta := factionMeta[doc.EntityID]
			parts := []string{"势力"}
			if meta.ParentName != nil && *meta.ParentName != "" {
				parts = append(parts, "上级："+*meta.ParentName)
			}
			if meta.LeaderName != nil && *meta.LeaderName != "" {
				parts = append(parts, "首领："+*meta.LeaderName)
			}
			item.Subtitle = strings.Join(parts, " · ")
			item.URL = "/factions?id=" + url.QueryEscape(doc.EntityID)
		case model.SearchEntityLocation:
			meta := locationMeta[doc.EntityID]
			parts := []string{"地点"}
			if meta.TypeText != nil && *meta.TypeText != "" {
				parts = append(parts, *meta.TypeText)
			}
			if meta.ParentName != nil && *meta.ParentName != "" {
				parts = append(parts, "上级："+*meta.ParentName)
			}
			item.Subtitle = strings.Join(parts, " · ")
			item.URL = "/locations?id=" + url.QueryEscape(doc.EntityID)
		case model.SearchEntityRanking:
			meta := rankingMeta[doc.EntityID]
			item.Subtitle = fmt.Sprintf("榜单 · %d 席 · %d 人在榜", meta.SeatCount, meta.Occupied)
			item.URL = "/rankings?id=" + url.QueryEscape(doc.EntityID)
		case model.SearchEntitySystemNode:
			meta := nodeMeta[doc.EntityID]
			parts := []string{"体系节点"}
			if meta.CategoryName != "" {
				parts = append(parts, meta.CategoryName)
			}
			if meta.ParentName != nil && *meta.ParentName != "" {
				parts = append(parts, "上级："+*meta.ParentName)
			}
			item.Subtitle = strings.Join(parts, " · ")
			item.URL = "/system?category=" + url.QueryEscape(meta.CategoryID) + "&node=" + url.QueryEscape(doc.EntityID)
		}
		result[doc.EntityType] = append(result[doc.EntityType], item)
	}
	return result, nil
}

func characterSubtitle(summary model.CharacterSummary) string {
	parts := make([]string, 0, 3)
	if summary.Faction != nil && summary.Faction.Name != "" {
		parts = append(parts, summary.Faction.Name)
	}
	if summary.SystemSummary != nil && *summary.SystemSummary != "" {
		parts = append(parts, *summary.SystemSummary)
	}
	if len(summary.Rankings) > 0 {
		parts = append(parts, summary.Rankings[0].Name)
	}
	if len(parts) == 0 {
		return "人物"
	}
	return strings.Join(parts, " · ")
}

func (s *Search) loadFactionMeta(ctx context.Context, ids []string) (map[string]factionSearchMeta, error) {
	out := make(map[string]factionSearchMeta, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	statement := `SELECT f.id, p.name AS parent_name, c.name AS leader_name
		FROM factions f
		LEFT JOIN factions p ON p.id = f.parent_id AND p.deleted_at IS NULL
		LEFT JOIN characters c ON c.id = f.leader_character_id AND c.deleted_at IS NULL
		WHERE f.id IN (` + placeholders(len(ids)) + `)`
	var rows []factionSearchMeta
	if err := s.db.WithContext(ctx).Raw(statement, stringArgs(ids)...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func (s *Search) loadLocationMeta(ctx context.Context, ids []string) (map[string]locationSearchMeta, error) {
	out := make(map[string]locationSearchMeta, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	statement := `SELECT l.id, l.type_text, p.name AS parent_name
		FROM locations l
		LEFT JOIN locations p ON p.id = l.parent_id AND p.deleted_at IS NULL
		WHERE l.id IN (` + placeholders(len(ids)) + `)`
	var rows []locationSearchMeta
	if err := s.db.WithContext(ctx).Raw(statement, stringArgs(ids)...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func (s *Search) loadRankingMeta(ctx context.Context, ids []string) (map[string]rankingSearchMeta, error) {
	out := make(map[string]rankingSearchMeta, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	statement := `SELECT r.id, r.seat_count, count(rs.character_id) AS occupied
		FROM rankings r
		LEFT JOIN ranking_seats rs ON rs.ranking_id = r.id AND rs.character_id IS NOT NULL
		WHERE r.id IN (` + placeholders(len(ids)) + `)
		GROUP BY r.id, r.seat_count`
	var rows []rankingSearchMeta
	if err := s.db.WithContext(ctx).Raw(statement, stringArgs(ids)...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func (s *Search) loadNodeMeta(ctx context.Context, ids []string) (map[string]nodeSearchMeta, error) {
	out := make(map[string]nodeSearchMeta, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	statement := `SELECT n.id, n.category_id, sc.name AS category_name, p.name AS parent_name
		FROM system_nodes n
		JOIN system_categories sc ON sc.id = n.category_id
		LEFT JOIN system_nodes p ON p.id = n.parent_id AND p.deleted_at IS NULL
		WHERE n.id IN (` + placeholders(len(ids)) + `)`
	var rows []nodeSearchMeta
	if err := s.db.WithContext(ctx).Raw(statement, stringArgs(ids)...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func placeholders(count int) string {
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func stringArgs(values []string) []any {
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return args
}

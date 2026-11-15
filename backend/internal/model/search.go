package model

import "time"

// 检索文档实体类型；顺序即结果分组顺序。
const (
	SearchEntityCharacter  = "character"
	SearchEntityFaction    = "faction"
	SearchEntityLocation   = "location"
	SearchEntityRanking    = "ranking"
	SearchEntitySystemNode = "system_node"

	SearchPageSize = 20

	SearchFactionScopeDirect  = "direct"
	SearchFactionScopeSubtree = "subtree"
)

// SearchEntityTypes 返回默认分组顺序的实体类型。
func SearchEntityTypes() []string {
	return []string{
		SearchEntityCharacter,
		SearchEntityFaction,
		SearchEntityLocation,
		SearchEntityRanking,
		SearchEntitySystemNode,
	}
}

// SearchDocument 是 search_documents 行；仅后端使用，不直接暴露给 API。
type SearchDocument struct {
	ID           string    `gorm:"column:id;primaryKey"`
	WorkID       string    `gorm:"column:work_id"`
	EntityType   string    `gorm:"column:entity_type"`
	EntityID     string    `gorm:"column:entity_id"`
	NameText     string    `gorm:"column:name_text"`
	NameKey      string    `gorm:"column:name_key"`
	AliasText    *string   `gorm:"column:alias_text"`
	TitleText    *string   `gorm:"column:title_text"`
	DescText     *string   `gorm:"column:desc_text"`
	RelationText *string   `gorm:"column:relation_text"`
	ReviewStatus string    `gorm:"column:review_status"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
	IsDeleted    bool      `gorm:"column:is_deleted"`
}

func (SearchDocument) TableName() string { return "search_documents" }

// SearchMatchedField 是命中字段与命中原文窗口。
type SearchMatchedField struct {
	Field       string `json:"field"`
	MatchedText string `json:"matched_text"`
}

// SearchItem 是结果行；人物额外带缩略图。
type SearchItem struct {
	EntityID      string               `json:"entity_id"`
	EntityType    string               `json:"entity_type"`
	Title         string               `json:"title"`
	Subtitle      string               `json:"subtitle"`
	MatchedFields []SearchMatchedField `json:"matched_fields"`
	Snippet       string               `json:"snippet"`
	Score         int                  `json:"score"`
	UpdatedAt     time.Time            `json:"updated_at"`
	URL           string               `json:"url"`
	ReviewStatus  string               `json:"review_status"`
	ImageThumbURL *string              `json:"image_thumb_url,omitempty"`
}

// SearchGroup 是类型分组；Total 为组总命中，Items 为当前页切片。
type SearchGroup struct {
	EntityType string       `json:"entity_type"`
	Total      int64        `json:"total"`
	Items      []SearchItem `json:"items"`
}

// SearchFilters 回显本次生效的筛选条件。
type SearchFilters struct {
	EntityTypes      []string `json:"entity_types"`
	FactionID        *string  `json:"faction_id"`
	FactionScope     string   `json:"faction_scope"`
	SystemCategoryID *string  `json:"system_category_id"`
	SystemNodeID     *string  `json:"system_node_id"`
	LocationID       *string  `json:"location_id"`
	RankingID        *string  `json:"ranking_id"`
	ReviewStatus     *string  `json:"review_status"`
}

// SearchQueryView 回显关键词与分词。
type SearchQueryView struct {
	Q       string        `json:"q"`
	Tokens  []string      `json:"tokens"`
	Filters SearchFilters `json:"filters"`
}

// SearchResult 是 GET /search 的响应。
type SearchResult struct {
	Groups   []SearchGroup   `json:"groups"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Query    SearchQueryView `json:"query"`
}

// SearchConsistency 是一致性检查结果。
type SearchConsistency struct {
	WorkID        string           `json:"work_id"`
	TotalDocs     int64            `json:"total_docs"`
	TotalEntities int64            `json:"total_entities"`
	Drift         int64            `json:"drift"`
	ByType        []SearchTypeStat `json:"by_type"`
}

// SearchTypeStat 是按实体类型的计数。
type SearchTypeStat struct {
	EntityType string `json:"entity_type"`
	Docs       int64  `json:"docs"`
	Entities   int64  `json:"entities"`
}

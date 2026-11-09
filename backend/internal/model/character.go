package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	CharacterNameKindAlias = "alias"
	CharacterNameKindTitle = "title"

	MaxCharacterNames     = 100
	MaxCharacterNameRunes = 100
)

// Character 是人物根记录；删除采用软删除三件套，回收站只展示 deleted_root_id = id 的根。
type Character struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	WorkID          string     `gorm:"column:work_id" json:"work_id"`
	Name            string     `gorm:"column:name" json:"name"`
	NameKey         string     `gorm:"column:name_key" json:"-"`
	Description     *string    `gorm:"column:description" json:"description"`
	FactionID       *string    `gorm:"column:faction_id" json:"-"`
	ImagePath       *string    `gorm:"column:image_path" json:"-"`
	ReviewStatus    string     `gorm:"column:review_status" json:"review_status"`
	ReviewedAt      *time.Time `gorm:"column:reviewed_at" json:"reviewed_at"`
	ReviewedBy      *string    `gorm:"column:reviewed_by" json:"reviewed_by"`
	Version         int        `gorm:"column:version" json:"version"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	DeletedRootID   *string    `gorm:"column:deleted_root_id" json:"-"`
	DeletedByAction *string    `gorm:"column:deleted_by_action" json:"-"`
}

func (Character) TableName() string { return "characters" }

// CharacterName 是别名或称号；随人物可见性联动，无独立软删除列。
type CharacterName struct {
	ID          string    `gorm:"column:id;primaryKey" json:"-"`
	CharacterID string    `gorm:"column:character_id" json:"-"`
	WorkID      string    `gorm:"column:work_id" json:"-"`
	Kind        string    `gorm:"column:kind" json:"kind"`
	Value       string    `gorm:"column:value" json:"value"`
	ValueKey    string    `gorm:"column:value_key" json:"-"`
	SortOrder   int       `gorm:"column:sort_order" json:"sort_order"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"-"`
}

func (CharacterName) TableName() string { return "character_names" }

// CharacterSystemRef 是人物在某类别下选中的节点；每类别至多一条。
type CharacterSystemRef struct {
	ID          string    `gorm:"column:id;primaryKey" json:"-"`
	WorkID      string    `gorm:"column:work_id" json:"-"`
	CharacterID string    `gorm:"column:character_id" json:"-"`
	CategoryID  string    `gorm:"column:category_id" json:"category_id"`
	NodeID      string    `gorm:"column:node_id" json:"node_id"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"-"`
}

func (CharacterSystemRef) TableName() string { return "character_system_refs" }

// CharacterNameView 是 API 输出的别名/称号项。
type CharacterNameView struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// FactionBrief 是势力摘要占位；CHG-006 前恒为 null。
type FactionBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CharacterNodeParentView 是选中节点的父层级摘要。
type CharacterNodeParentView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CharacterNodeView 是体系引用中的节点摘要。
type CharacterNodeView struct {
	ID     string                   `json:"id"`
	Name   string                   `json:"name"`
	Parent *CharacterNodeParentView `json:"parent"`
}

// SystemRefView 是详情页按类别分组的体系引用。
type SystemRefView struct {
	CategoryID   string            `json:"category_id"`
	CategoryName string            `json:"category_name"`
	CategoryType string            `json:"category_type"`
	Node         CharacterNodeView `json:"node"`
}

// LocationLinkView 是地点关联占位；CHG-007 前恒为空数组。
type LocationLinkView struct {
	LocationID string  `json:"location_id"`
	Name       string  `json:"name"`
	Note       *string `json:"note"`
}

// CharacterImageView 是主图占位；CHG-005 前恒为 null。
type CharacterImageView struct {
	AssetID string `json:"asset_id"`
	URL     string `json:"url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// CharacterSummary 是列表行；相对契约增加 names[] 与 version（设计 D8）。
type CharacterSummary struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Names         []CharacterNameView `json:"names"`
	Faction       *FactionBrief       `json:"faction"`
	SystemSummary *string             `json:"system_summary"`
	ImageThumbURL *string             `json:"image_thumb_url"`
	ReviewStatus  string              `json:"review_status"`
	ReviewedAt    *time.Time          `json:"reviewed_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Version       int                 `json:"version"`
}

// CharacterDetail 是详情 DTO；占位字段按设计固定为 null 或空数组。
type CharacterDetail struct {
	ID            string              `json:"id"`
	WorkID        string              `json:"work_id"`
	Name          string              `json:"name"`
	Names         []CharacterNameView `json:"names"`
	Description   *string             `json:"description"`
	Faction       *FactionBrief       `json:"faction"`
	SystemRefs    []SystemRefView     `json:"system_refs"`
	LocationLinks []LocationLinkView  `json:"location_links"`
	Image         *CharacterImageView `json:"image"`
	ReviewStatus  string              `json:"review_status"`
	ReviewedAt    *time.Time          `json:"reviewed_at"`
	ReviewedBy    *string             `json:"reviewed_by"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Version       int                 `json:"version"`
	DeletedAt     *time.Time          `json:"deleted_at"`
}

// NormalizeCharacterName 生成人物主名称与唯一键；人物名称仅做 lower(trim)（设计 D6）。
func NormalizeCharacterName(raw string) (name, nameKey string, err error) {
	return normalizeCharacterName(raw, "name")
}

// NormalizeCharacterNameValue 规范化别名/称号的展示值与唯一键。
func NormalizeCharacterNameValue(raw, field string) (value, valueKey string, err error) {
	value = strings.TrimSpace(raw)
	if value == "" {
		return "", "", &ValidationError{Field: field, Message: "别名或称号不能为空"}
	}
	if utf8.RuneCountInString(value) > MaxCharacterNameRunes {
		return "", "", &ValidationError{Field: field, Message: "别名或称号不能超过 100 个字符"}
	}
	return value, strings.ToLower(value), nil
}

func normalizeCharacterName(raw, field string) (name, nameKey string, err error) {
	name = strings.TrimSpace(raw)
	if name == "" {
		return "", "", &ValidationError{Field: field, Message: "人物名称不能为空"}
	}
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return "", "", &ValidationError{Field: field, Message: "人物名称不能超过 200 个字符"}
	}
	return name, strings.ToLower(name), nil
}

package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxLocationName / MaxLocationType / MaxLocationDescription 与契约 /meta 限制一致。
	MaxLocationName        = 200
	MaxLocationType        = 200
	MaxLocationDescription = 20000
	// MaxRelationNote 是人物-地点与地点-势力关系备注上限（/meta limits.relation_note）。
	MaxRelationNote = 500
	// LocationActionDelete 是地点软删除的动作标记。
	LocationActionDelete = "location_delete"
	// ReferenceListLimit 是删除阻断清单单次返回的最大条目数。
	ReferenceListLimit = 50
)

// Location 是地点树节点；删除采用软删除三件套，回收站只展示 deleted_root_id = id 的根。
type Location struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	WorkID          string     `gorm:"column:work_id" json:"work_id"`
	Name            string     `gorm:"column:name" json:"name"`
	NameKey         string     `gorm:"column:name_key" json:"-"`
	ParentID        *string    `gorm:"column:parent_id" json:"-"`
	TypeText        *string    `gorm:"column:type_text" json:"type_text"`
	Description     *string    `gorm:"column:description" json:"description"`
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

func (Location) TableName() string { return "locations" }

// CharacterLocation 是人物-地点关系；随人物/地点可见性联动，无独立软删除列。
type CharacterLocation struct {
	ID          string    `gorm:"column:id;primaryKey" json:"-"`
	WorkID      string    `gorm:"column:work_id" json:"-"`
	CharacterID string    `gorm:"column:character_id" json:"-"`
	LocationID  string    `gorm:"column:location_id" json:"location_id"`
	Note        *string   `gorm:"column:note" json:"note"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"-"`
}

func (CharacterLocation) TableName() string { return "character_locations" }

// LocationFaction 是地点-势力关系；随地点/势力可见性联动，无独立软删除列。
type LocationFaction struct {
	ID         string    `gorm:"column:id;primaryKey" json:"-"`
	WorkID     string    `gorm:"column:work_id" json:"-"`
	LocationID string    `gorm:"column:location_id" json:"-"`
	FactionID  string    `gorm:"column:faction_id" json:"faction_id"`
	Note       *string   `gorm:"column:note" json:"note"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"-"`
}

func (LocationFaction) TableName() string { return "location_factions" }

// LocationBrief 是父级/引用摘要。
type LocationBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LocationSummary 是列表/树行；父级与计数由服务批量装配。
type LocationSummary struct {
	ID              string         `json:"id"`
	WorkID          string         `json:"work_id"`
	Name            string         `json:"name"`
	Parent          *LocationBrief `json:"parent"`
	TypeText        *string        `json:"type_text"`
	Description     *string        `json:"description"`
	ReviewStatus    string         `json:"review_status"`
	ReviewedAt      *time.Time     `json:"reviewed_at"`
	ChildrenCount   int64          `json:"children_count"`
	CharactersCount int64          `json:"characters_count"`
	FactionsCount   int64          `json:"factions_count"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Version         int            `json:"version"`
	DeletedAt       *time.Time     `json:"deleted_at"`
}

// LocationChildView 是详情页下级地点行。
type LocationChildView struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	TypeText        *string `json:"type_text"`
	CharactersCount int64   `json:"characters_count"`
	FactionsCount   int64   `json:"factions_count"`
}

// LocationCharacterView 是地点详情页关联人物行。
type LocationCharacterView struct {
	CharacterID   string        `json:"character_id"`
	Name          string        `json:"name"`
	Faction       *FactionBrief `json:"faction"`
	SystemSummary *string       `json:"system_summary"`
	ImageThumbURL *string       `json:"image_thumb_url"`
	Note          *string       `json:"note"`
	InTrash       bool          `json:"in_trash"`
}

// LocationFactionView 是地点详情页关联势力行。
type LocationFactionView struct {
	FactionID string  `json:"faction_id"`
	Name      string  `json:"name"`
	Note      *string `json:"note"`
	InTrash   bool    `json:"in_trash"`
}

// LocationRelations 是详情页关系区；parent 位于 LocationSummary。
type LocationRelations struct {
	Children   []LocationChildView     `json:"children"`
	Characters []LocationCharacterView `json:"characters"`
	Factions   []LocationFactionView   `json:"factions"`
}

// LocationDetail 是详情 DTO。
type LocationDetail struct {
	LocationSummary
	Relations LocationRelations `json:"relations"`
}

// NormalizeLocationName 生成展示名与唯一键 name_key = lower(trim(name))。
func NormalizeLocationName(raw string) (name, nameKey string, err error) {
	name = strings.TrimSpace(raw)
	if name == "" {
		return "", "", &ValidationError{Field: "name", Message: "地点名称不能为空"}
	}
	if utf8.RuneCountInString(name) > MaxLocationName {
		return "", "", &ValidationError{Field: "name", Message: "地点名称不能超过 200 个字符"}
	}
	return name, strings.ToLower(name), nil
}

// NormalizeLocationType 空串视为未填写（nil），并校验 200 字符上限。
func NormalizeLocationType(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxLocationType {
		return nil, &ValidationError{Field: "type_text", Message: "地点类型不能超过 200 个字符"}
	}
	return &value, nil
}

// NormalizeLocationDescription 空串视为未填写（nil），并校验 20,000 字符上限。
func NormalizeLocationDescription(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxLocationDescription {
		return nil, &ValidationError{Field: "description", Message: "地点简介不能超过 20000 个字符"}
	}
	return &value, nil
}

// NormalizeRelationNote 空串视为未填写（nil），并校验 500 字符上限。
func NormalizeRelationNote(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxRelationNote {
		return nil, &ValidationError{Field: "note", Message: "关系备注不能超过 500 个字符"}
	}
	return &value, nil
}

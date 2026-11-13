package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxFactionName / MaxFactionDescription 与契约 /meta 限制一致。
	MaxFactionName        = 200
	MaxFactionDescription = 20000
	// FactionActionDelete 是势力软删除的动作标记，用于撤销/恢复区分随根删除。
	FactionActionDelete = "faction_delete"
)

// Faction 是势力树节点；删除采用软删除三件套，回收站只展示 deleted_root_id = id 的根。
type Faction struct {
	ID                string     `gorm:"column:id;primaryKey" json:"id"`
	WorkID            string     `gorm:"column:work_id" json:"work_id"`
	Name              string     `gorm:"column:name" json:"name"`
	NameKey           string     `gorm:"column:name_key" json:"-"`
	ParentID          *string    `gorm:"column:parent_id" json:"-"`
	LeaderCharacterID *string    `gorm:"column:leader_character_id" json:"-"`
	Description       *string    `gorm:"column:description" json:"description"`
	ReviewStatus      string     `gorm:"column:review_status" json:"review_status"`
	ReviewedAt        *time.Time `gorm:"column:reviewed_at" json:"reviewed_at"`
	ReviewedBy        *string    `gorm:"column:reviewed_by" json:"reviewed_by"`
	Version           int        `gorm:"column:version" json:"version"`
	CreatedAt         time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt         *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	DeletedRootID     *string    `gorm:"column:deleted_root_id" json:"-"`
	DeletedByAction   *string    `gorm:"column:deleted_by_action" json:"-"`
}

func (Faction) TableName() string { return "factions" }

// FactionSummary 是列表/树行；父子与首领用摘要引用，计数由服务批量装配。
type FactionSummary struct {
	ID            string        `json:"id"`
	WorkID        string        `json:"work_id"`
	Name          string        `json:"name"`
	Parent        *FactionBrief `json:"parent"`
	Leader        *FactionBrief `json:"leader"`
	Description   *string       `json:"description"`
	ReviewStatus  string        `json:"review_status"`
	ReviewedAt    *time.Time    `json:"reviewed_at"`
	ChildrenCount int64         `json:"children_count"`
	MemberCount   int64         `json:"member_count"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Version       int           `json:"version"`
	DeletedAt     *time.Time    `json:"deleted_at"`
}

// LocationBrief 是势力关联地点占位；CHG-007 前恒为空数组。
type LocationBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FactionRelations 是详情侧栏关系；地点关系由 CHG-007 接入，本轮恒为空数组。
type FactionRelations struct {
	Children           []FactionBrief  `json:"children"`
	Locations          []LocationBrief `json:"locations"`
	ChronicleCount     int64           `json:"chronicle_count"`
	SubtreeMemberCount int64           `json:"subtree_member_count"`
}

// FactionDetail 是详情 DTO。
type FactionDetail struct {
	FactionSummary
	Relations FactionRelations `json:"relations"`
}

// NormalizeFactionName 生成展示名与唯一键 name_key = lower(trim(name))（设计 D6）。
func NormalizeFactionName(raw string) (name, nameKey string, err error) {
	name = strings.TrimSpace(raw)
	if name == "" {
		return "", "", &ValidationError{Field: "name", Message: "势力名称不能为空"}
	}
	if utf8.RuneCountInString(name) > MaxFactionName {
		return "", "", &ValidationError{Field: "name", Message: "势力名称不能超过 200 个字符"}
	}
	return name, strings.ToLower(name), nil
}

// NormalizeFactionDescription 空字符串视为未填写（nil），并校验 20,000 字符上限。
func NormalizeFactionDescription(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxFactionDescription {
		return nil, &ValidationError{Field: "description", Message: "势力简介不能超过 20000 个字符"}
	}
	return &value, nil
}

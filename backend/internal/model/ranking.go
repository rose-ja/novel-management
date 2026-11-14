package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxRankingName 是榜单名称上限（/meta limits.ranking_name）。
	MaxRankingName = 200
	// MaxRankingSeats 是榜单席位数上限（含扩容）。
	MaxRankingSeats = 200
	// DefaultRankingSeats 是新建榜单默认席位数。
	DefaultRankingSeats = 100
	// MaxSeatNote 是席位备注上限。
	MaxSeatNote = 500
	// RankingActionDelete 是榜单软删除的动作标记。
	RankingActionDelete = "ranking_delete"

	SeatModeEmpty = "empty"
	SeatModeSwap  = "swap"
)

// Ranking 是榜单根记录；删除采用软删除三件套，席位随榜单可见性联动。
type Ranking struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	WorkID          string     `gorm:"column:work_id" json:"work_id"`
	Name            string     `gorm:"column:name" json:"name"`
	NameKey         string     `gorm:"column:name_key" json:"-"`
	SeatCount       int        `gorm:"column:seat_count" json:"seat_count"`
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

func (Ranking) TableName() string { return "rankings" }

// RankingSeat 是固定席位行；空席 character_id 为 NULL，随榜单可见性联动。
type RankingSeat struct {
	ID          string    `gorm:"column:id;primaryKey" json:"-"`
	WorkID      string    `gorm:"column:work_id" json:"-"`
	RankingID   string    `gorm:"column:ranking_id" json:"-"`
	SeatNo      int       `gorm:"column:seat_no" json:"seat_no"`
	CharacterID *string   `gorm:"column:character_id" json:"character_id"`
	Note        *string   `gorm:"column:note" json:"note"`
	Version     int       `gorm:"column:version" json:"version"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"-"`
}

func (RankingSeat) TableName() string { return "ranking_seats" }

// RankingTopCharacter 是前三席人物摘要。
type RankingTopCharacter struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RankingTopSeat 是列表页前三席展示项；空席 character 为 null。
type RankingTopSeat struct {
	SeatNo    int                  `json:"seat_no"`
	Character *RankingTopCharacter `json:"character"`
}

// RankingSummary 是列表行；占用量与前三席由服务装配。
type RankingSummary struct {
	ID            string           `json:"id"`
	WorkID        string           `json:"work_id"`
	Name          string           `json:"name"`
	SeatCount     int              `json:"seat_count"`
	OccupiedCount int              `json:"occupied_count"`
	EmptyCount    int              `json:"empty_count"`
	TopSeats      []RankingTopSeat `json:"top_seats"`
	ReviewStatus  string           `json:"review_status"`
	ReviewedAt    *time.Time       `json:"reviewed_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Version       int              `json:"version"`
}

// SeatView 是席位行；空席 Character 为 null。
type SeatView struct {
	SeatNo    int               `json:"seat_no"`
	Character *CharacterSummary `json:"character"`
	Note      *string           `json:"note"`
	Version   int               `json:"version"`
}

// SeatPage 是席位分页结果。
type SeatPage struct {
	Items    []SeatView `json:"items"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
}

// RankingDetail 是详情 DTO。
type RankingDetail struct {
	RankingSummary
	Seats SeatPage `json:"seats"`
}

// RankingRefView 是人物侧的名次回显；Note 为空席时为 null。
type RankingRefView struct {
	RankingID string  `json:"ranking_id"`
	Name      string  `json:"name"`
	SeatNo    int     `json:"seat_no"`
	Note      *string `json:"note"`
}

// NormalizeRankingName 生成展示名与唯一键 name_key = lower(trim(name))。
func NormalizeRankingName(raw string) (name, nameKey string, err error) {
	name = strings.TrimSpace(raw)
	if name == "" {
		return "", "", &ValidationError{Field: "name", Message: "榜单名称不能为空"}
	}
	if utf8.RuneCountInString(name) > MaxRankingName {
		return "", "", &ValidationError{Field: "name", Message: "榜单名称不能超过 200 个字符"}
	}
	return name, strings.ToLower(name), nil
}

// NormalizeSeatCount 校验席位数范围 1..200。
func NormalizeSeatCount(count int) error {
	if count < 1 || count > MaxRankingSeats {
		return &ValidationError{Field: "seat_count", Message: "席位数需在 1 到 200 之间"}
	}
	return nil
}

// NormalizeSeatNote 空串视为未填写（nil），并校验 500 字符上限。
func NormalizeSeatNote(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxSeatNote {
		return nil, &ValidationError{Field: "note", Message: "席位备注不能超过 500 个字符"}
	}
	return &value, nil
}

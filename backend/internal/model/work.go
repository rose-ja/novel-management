package model

import "time"

// Work 是作品根记录；删除采用软删除三件套，回收站只展示 deleted_root_id = id 的根。
type Work struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	Name            string     `gorm:"column:name" json:"name"`
	NameKey         string     `gorm:"column:name_key" json:"-"`
	Description     *string    `gorm:"column:description" json:"description"`
	Version         int        `gorm:"column:version" json:"version"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	DeletedRootID   *string    `gorm:"column:deleted_root_id" json:"-"`
	DeletedByAction *string    `gorm:"column:deleted_by_action" json:"-"`
}

func (Work) TableName() string { return "works" }

// SystemCategory 是作品下的内置/自定义类别；本变更只使用内置「境界」。
type SystemCategory struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	WorkID          string     `gorm:"column:work_id" json:"work_id"`
	Name            string     `gorm:"column:name" json:"name"`
	NameKey         string     `gorm:"column:name_key" json:"-"`
	CategoryType    string     `gorm:"column:category_type" json:"category_type"`
	IsBuiltin       bool       `gorm:"column:is_builtin" json:"is_builtin"`
	IsEnabled       bool       `gorm:"column:is_enabled" json:"is_enabled"`
	Version         int        `gorm:"column:version" json:"version"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	DeletedRootID   *string    `gorm:"column:deleted_root_id" json:"-"`
	DeletedByAction *string    `gorm:"column:deleted_by_action" json:"-"`
}

func (SystemCategory) TableName() string { return "system_categories" }

// BuiltinRealmCategory 是创建作品时同事务写入的内置类别名称。
const BuiltinRealmCategory = "境界"

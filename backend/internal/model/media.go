package model

import "time"

const (
	// MaxImageBytes 是单张人物主图的上限（10MB）。
	MaxImageBytes = 10 << 20
	// MaxImagePixels 限制解码后的总像素，避免解压炸弹。
	MaxImagePixels = 40_000_000
	// MaxImageDimension 限制重编码后的最长边。
	MaxImageDimension = 4096
	// ThumbnailLongEdge 是缩略图最长边。
	ThumbnailLongEdge = 320

	MediaEntityCharacter = "character"

	MediaActionReplace = "image_replace"
	MediaActionDelete  = "image_delete"

	// ActionCharacterDelete / ActionWorkDelete 是随根删除的动作标记，用于恢复时区分用户主动删除。
	ActionCharacterDelete = "character_delete"
	ActionWorkDelete      = "work_delete"

	ChronicleOwnerCharacter = "character"
	ChronicleOwnerFaction   = "faction"
	MaxChronicleRunes       = 2000
	MaxChronicleTimeLabel   = 200
	ChronicleActionDelete   = "chronicle_delete"
)

// MediaAsset 是人物主图资产；文件路径相对 DATA_DIR。
type MediaAsset struct {
	ID              string     `gorm:"column:id;primaryKey" json:"-"`
	WorkID          string     `gorm:"column:work_id" json:"-"`
	EntityType      string     `gorm:"column:entity_type" json:"-"`
	EntityID        string     `gorm:"column:entity_id" json:"-"`
	OriginalPath    string     `gorm:"column:original_path" json:"-"`
	ThumbPath       *string    `gorm:"column:thumb_path" json:"-"`
	Mime            string     `gorm:"column:mime" json:"-"`
	ByteSize        int64      `gorm:"column:byte_size" json:"-"`
	Width           *int       `gorm:"column:width" json:"-"`
	Height          *int       `gorm:"column:height" json:"-"`
	SHA256          string     `gorm:"column:sha256" json:"-"`
	Version         int        `gorm:"column:version" json:"-"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt       *time.Time `gorm:"column:deleted_at" json:"-"`
	DeletedRootID   *string    `gorm:"column:deleted_root_id" json:"-"`
	DeletedByAction *string    `gorm:"column:deleted_by_action" json:"-"`
}

func (MediaAsset) TableName() string { return "media_assets" }

// MediaView 是 API 输出的图片信息，URL 为站内相对路径。
type MediaView struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	ByteSize int64  `json:"byte_size"`
}

// Chronicle 是人物大事记；单条删除使用软删除 + 撤销令牌。
type Chronicle struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	WorkID          string     `gorm:"column:work_id" json:"-"`
	OwnerType       string     `gorm:"column:owner_type" json:"owner_type"`
	OwnerID         string     `gorm:"column:owner_id" json:"owner_id"`
	Text            string     `gorm:"column:text" json:"text"`
	TimeLabel       *string    `gorm:"column:time_label" json:"time_label"`
	SortOrder       int        `gorm:"column:sort_order" json:"sort_order"`
	Version         int        `gorm:"column:version" json:"version"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at" json:"-"`
	DeletedRootID   *string    `gorm:"column:deleted_root_id" json:"-"`
	DeletedByAction *string    `gorm:"column:deleted_by_action" json:"-"`
}

func (Chronicle) TableName() string { return "chronicles" }

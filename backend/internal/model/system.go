package model

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	CategoryTypeNormal = "normal"
	CategoryTypeEra    = "era"

	MaxSystemCategories = 10
	MaxNodeLevels       = 2
	MaxEraTextRunes     = 100

	ReviewStatusPending      = "pending"
	ReviewStatusReviewed     = "reviewed"
	ReviewStatusAPIPending   = "pending"
	ReviewStatusAPIConfirmed = "confirmed"
)

// SystemNode 是体系类别下的两级节点；删除采用软删除三件套。
type SystemNode struct {
	ID              string       `gorm:"column:id;primaryKey" json:"id"`
	WorkID          string       `gorm:"column:work_id" json:"work_id"`
	CategoryID      string       `gorm:"column:category_id" json:"category_id"`
	ParentID        *string      `gorm:"column:parent_id" json:"parent_id"`
	Name            string       `gorm:"column:name" json:"name"`
	NameKey         string       `gorm:"column:name_key" json:"-"`
	SortOrder       int          `gorm:"column:sort_order" json:"sort_order"`
	EraStartText    *string      `gorm:"column:era_start_text" json:"era_start_text"`
	EraEndText      *string      `gorm:"column:era_end_text" json:"era_end_text"`
	ReviewStatus    string       `gorm:"column:review_status" json:"-"`
	ReviewedAt      *time.Time   `gorm:"column:reviewed_at" json:"reviewed_at"`
	ReviewedBy      *string      `gorm:"column:reviewed_by" json:"reviewed_by"`
	Version         int          `gorm:"column:version" json:"version"`
	CreatedAt       time.Time    `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time    `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt       *time.Time   `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	DeletedRootID   *string      `gorm:"column:deleted_root_id" json:"-"`
	DeletedByAction *string      `gorm:"column:deleted_by_action" json:"-"`
	ReferenceCount  int64        `gorm:"-" json:"reference_count"`
	Children        []SystemNode `gorm:"-" json:"children,omitempty"`
}

func (SystemNode) TableName() string { return "system_nodes" }

// APIReviewStatus 把数据库值 reviewed 映射为 API 值 confirmed。
func APIReviewStatus(db string) string {
	if db == ReviewStatusReviewed {
		return ReviewStatusAPIConfirmed
	}
	return ReviewStatusAPIPending
}

// DBReviewStatus 把 API 筛选值转换为数据库值；空值表示不过滤。
func DBReviewStatus(api string) (string, error) {
	switch api {
	case "":
		return "", nil
	case ReviewStatusAPIPending:
		return ReviewStatusPending, nil
	case ReviewStatusAPIConfirmed:
		return ReviewStatusReviewed, nil
	default:
		return "", &ValidationError{Field: "review_status", Message: "review_status 仅支持 pending 或 confirmed"}
	}
}

// NormalizeKey 生成体系名称的展示值与唯一键（NFKC → 小写 → trim → 连续空白折叠）。
func NormalizeKey(raw, field string) (display, key string, err error) {
	display = strings.TrimSpace(raw)
	if display == "" {
		return "", "", &ValidationError{Field: field, Message: "名称不能为空"}
	}
	if utf8.RuneCountInString(display) > MaxNameRunes {
		return "", "", &ValidationError{Field: field, Message: "名称不能超过 200 个字符"}
	}
	normalized := norm.NFKC.String(display)
	key = strings.ToLower(strings.Join(strings.Fields(normalized), " "))
	return display, key, nil
}

// NormalizeEraText 将空字符串视为未填写，并校验 100 rune 上限。
func NormalizeEraText(raw *string, field string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxEraTextRunes {
		return nil, &ValidationError{Field: field, Message: "纪元文本不能超过 100 个字符"}
	}
	return &value, nil
}

// AssembleNodeTree 把平铺节点组装为两级树；根与子层均按 (sort_order, id) 排序。
func AssembleNodeTree(nodes []SystemNode) ([]SystemNode, error) {
	byID := make(map[string]SystemNode, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}

	children := make(map[string][]SystemNode)
	var roots []SystemNode
	for _, node := range nodes {
		if node.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		if _, ok := byID[*node.ParentID]; !ok {
			return nil, fmt.Errorf("节点 %s 的上级节点缺失", node.ID)
		}
		children[*node.ParentID] = append(children[*node.ParentID], node)
	}

	sortNodes := func(items []SystemNode) {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].SortOrder != items[j].SortOrder {
				return items[i].SortOrder < items[j].SortOrder
			}
			return items[i].ID < items[j].ID
		})
	}

	sortNodes(roots)
	for i := range roots {
		rootChildren := children[roots[i].ID]
		sortNodes(rootChildren)
		for _, child := range rootChildren {
			if len(children[child.ID]) > 0 {
				return nil, fmt.Errorf("节点 %s 超过两级", child.ID)
			}
			child.Children = nil
		}
		roots[i].Children = rootChildren
	}
	return roots, nil
}

// ValidateSiblingOrder 校验排序请求与当前同级集合完全一致且无重复。
func ValidateSiblingOrder(siblings, ordered []string) error {
	if len(siblings) != len(ordered) {
		return &ValidationError{Field: "ordered_ids", Message: "排序集合必须覆盖全部同级节点"}
	}
	expected := make(map[string]struct{}, len(siblings))
	for _, id := range siblings {
		expected[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(ordered))
	for _, id := range ordered {
		if _, ok := expected[id]; !ok {
			return &ValidationError{Field: "ordered_ids", Message: "排序集合包含非同级节点"}
		}
		if _, dup := seen[id]; dup {
			return &ValidationError{Field: "ordered_ids", Message: "排序集合存在重复节点"}
		}
		seen[id] = struct{}{}
	}
	return nil
}

// SortSystemNodes 按 (sort_order, id) 排序平铺节点，供响应稳定输出。
func SortSystemNodes(nodes []SystemNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].SortOrder != nodes[j].SortOrder {
			return nodes[i].SortOrder < nodes[j].SortOrder
		}
		return nodes[i].ID < nodes[j].ID
	})
}

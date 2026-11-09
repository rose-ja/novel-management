package model

import "errors"

// 领域错误：handler 依据这些哨兵值映射 HTTP 状态码与稳定错误码。
var (
	ErrNotFound        = errors.New("资源不存在")
	ErrNameConflict    = errors.New("名称已存在")
	ErrVersionConflict = errors.New("版本冲突")
	ErrSnapshotFailed  = errors.New("快照失败")
	ErrUndoExpired     = errors.New("撤销窗口已关闭")
	ErrAlreadyDeleted  = errors.New("已不在回收站")
	ErrInvalidRootType = errors.New("不支持的回收站类型")
	ErrForbidden       = errors.New("操作不被允许")

	ErrCategoryLimit            = errors.New("类别数量已达上限")
	ErrCategoryBuiltinImmutable = errors.New("内置类别不可修改")
	ErrCategoryDisabled         = errors.New("类别已停用")
	ErrNodeLevelLimit           = errors.New("体系节点最多两级")
	ErrNodeParentMismatch       = errors.New("上级节点与类别不匹配")
	ErrHasChildren              = errors.New("仍有下级节点")
	ErrParentDeleted            = errors.New("上级或类别仍在回收站")
	ErrPreconditionRequired     = errors.New("缺少 If-Match 请求头")
	ErrCharacterWorkUnavailable = errors.New("作品不存在或已删除")
)

// NameConflictError 表示同名冲突，details.items 列出冲突记录。
type NameConflictError struct {
	Items []ConflictItem `json:"items"`
}

func (e *NameConflictError) Error() string { return "名称已存在" }

// CategoryDisabledError 表示引用了未启用类别，details 携带类别名。
type CategoryDisabledError struct {
	CategoryName string `json:"category_name"`
}

func (e *CategoryDisabledError) Error() string { return "该类别已停用" }

// ValidationError 携带字段名，handler 转成 details.field。
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string { return e.Message }

// RestoreConflictError 表示恢复时存在同名活跃记录，details.items 列出冲突项。
type RestoreConflictError struct {
	Items []ConflictItem `json:"items"`
}

func (e *RestoreConflictError) Error() string { return "恢复目标与现有记录重名" }

// HasChildrenError 表示节点仍有子节点（含已软删除），details.items 列出子节点。
type HasChildrenError struct {
	Items []ConflictItem `json:"items"`
}

func (e *HasChildrenError) Error() string { return "仍有下级节点，无法删除" }

// ConflictItem 是恢复冲突的单项描述。
type ConflictItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	EntityType string `json:"entity_type,omitempty"`
}

// ReferenceItem 是一条引用阻断明细；0003 无人物表时 Name 为 null。
type ReferenceItem struct {
	Type     string  `json:"type"`
	EntityID string  `json:"entity_id"`
	Name     *string `json:"name"`
	Field    string  `json:"field"`
	Value    string  `json:"value"`
	InTrash  bool    `json:"in_trash"`
}

// ReferenceConflictError 表示停用/删除被人物引用阻止，details 列出全部引用。
type ReferenceConflictError struct {
	Message string          `json:"message"`
	Items   []ReferenceItem `json:"items"`
}

func (e *ReferenceConflictError) Error() string { return e.Message }

// CrossWorkReferenceError 表示请求引用了其他作品的类别或节点。
type CrossWorkReferenceError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *CrossWorkReferenceError) Error() string { return e.Message }

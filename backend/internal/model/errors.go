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
)

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

// ConflictItem 是恢复冲突的单项描述。
type ConflictItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

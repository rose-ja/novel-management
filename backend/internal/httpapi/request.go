package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/model"
)

// bindJSON 统一 422 响应；失败时已写出响应体。
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求体格式不正确")
		return false
	}
	return true
}

// parseOptionalString 区分字段缺失（nil, false）、显式 null（nil, true）与字符串。
func parseOptionalString(raw json.RawMessage) (*string, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	if string(raw) == "null" {
		return nil, true, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false, err
	}
	return &value, false, nil
}

// abortServiceError 将领域错误映射到稳定的 HTTP 状态码与错误码。
func abortServiceError(c *gin.Context, err error) {
	var validation *model.ValidationError
	if errors.As(err, &validation) {
		abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", validation.Message, gin.H{
			"field": validation.Field,
		})
		return
	}

	var nameConflict *model.NameConflictError
	if errors.As(err, &nameConflict) {
		abortErrorWithDetails(c, http.StatusConflict, "NAME_CONFLICT", "名称已存在", gin.H{
			"items": nameConflict.Items,
		})
		return
	}

	var categoryDisabled *model.CategoryDisabledError
	if errors.As(err, &categoryDisabled) {
		abortErrorWithDetails(c, http.StatusUnprocessableEntity, "CATEGORY_DISABLED", "该类别已停用", gin.H{
			"category_name": categoryDisabled.CategoryName,
		})
		return
	}

	var crossWork *model.CrossWorkReferenceError
	if errors.As(err, &crossWork) {
		abortErrorWithDetails(c, http.StatusUnprocessableEntity, "CROSS_WORK_REFERENCE", crossWork.Message, gin.H{
			"field": crossWork.Field,
		})
		return
	}

	var conflict *model.RestoreConflictError
	if errors.As(err, &conflict) {
		abortErrorWithDetails(c, http.StatusUnprocessableEntity, "RESTORE_CONFLICT", "恢复目标与现有记录重名", gin.H{
			"items": conflict.Items,
		})
		return
	}

	var hasChildren *model.HasChildrenError
	if errors.As(err, &hasChildren) {
		abortErrorWithDetails(c, http.StatusConflict, "HAS_CHILDREN", "仍有下级节点，无法删除", gin.H{
			"items": hasChildren.Items,
		})
		return
	}

	var reference *model.ReferenceConflictError
	if errors.As(err, &reference) {
		abortErrorWithDetails(c, http.StatusConflict, "REFERENCE_CONFLICT", reference.Message, reference.Items)
		return
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		abortError(c, http.StatusNotFound, "NOT_FOUND", "资源不存在")
	case errors.Is(err, model.ErrNameConflict):
		abortError(c, http.StatusConflict, "NAME_CONFLICT", "名称已存在")
	case errors.Is(err, model.ErrVersionConflict):
		abortError(c, http.StatusConflict, "VERSION_CONFLICT", "内容已被修改，请刷新后重试")
	case errors.Is(err, model.ErrSnapshotFailed):
		abortError(c, http.StatusServiceUnavailable, "SNAPSHOT_FAILED", "快照失败，操作已取消")
	case errors.Is(err, model.ErrUndoExpired):
		abortError(c, http.StatusConflict, "UNDO_EXPIRED", "撤销窗口已关闭")
	case errors.Is(err, model.ErrAlreadyDeleted):
		abortError(c, http.StatusConflict, "ALREADY_DELETED", "记录已不在回收站")
	case errors.Is(err, model.ErrInvalidRootType):
		abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "不支持的回收站类型", gin.H{
			"field": "root_type",
		})
	case errors.Is(err, model.ErrCategoryLimit):
		abortError(c, http.StatusConflict, "CATEGORY_LIMIT", "类别数量已达上限")
	case errors.Is(err, model.ErrCategoryBuiltinImmutable):
		abortError(c, http.StatusConflict, "CATEGORY_BUILTIN_IMMUTABLE", "内置类别不可修改")
	case errors.Is(err, model.ErrCategoryDisabled):
		abortError(c, http.StatusConflict, "CATEGORY_DISABLED", "该类别已停用")
	case errors.Is(err, model.ErrNodeLevelLimit):
		abortErrorWithDetails(c, http.StatusUnprocessableEntity, "NODE_LEVEL_LIMIT", "体系节点最多两级", gin.H{
			"field": "parent_id",
		})
	case errors.Is(err, model.ErrNodeParentMismatch):
		abortErrorWithDetails(c, http.StatusUnprocessableEntity, "NODE_PARENT_MISMATCH", "上级节点与类别不匹配", gin.H{
			"field": "parent_id",
		})
	case errors.Is(err, model.ErrHasChildren):
		abortError(c, http.StatusConflict, "HAS_CHILDREN", "仍有下级节点，无法删除")
	case errors.Is(err, model.ErrParentDeleted):
		abortError(c, http.StatusConflict, "PARENT_DELETED", "上级或类别仍在回收站")
	case errors.Is(err, model.ErrPreconditionRequired):
		abortError(c, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "缺少 If-Match 请求头")
	case errors.Is(err, model.ErrCharacterWorkUnavailable):
		abortError(c, http.StatusUnprocessableEntity, "PARENT_DELETED", "作品不存在或已删除")
	default:
		abortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
	}
}

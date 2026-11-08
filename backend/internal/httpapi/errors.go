package httpapi

import (
	"github.com/gin-gonic/gin"
)

// errorBody 是全局统一的错误结构：{ "error": { "code", "message", "details", "request_id" } }。
type errorBody struct {
	Error errorItem `json:"error"`
}

type errorItem struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id"`
}

// abortError 返回不含 details 的错误响应。
func abortError(c *gin.Context, status int, code, message string) {
	abortErrorWithDetails(c, status, code, message, nil)
}

// abortErrorWithDetails 返回带定位信息的错误响应，用于引用阻断与字段校验。
func abortErrorWithDetails(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, errorBody{Error: errorItem{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: c.GetString(requestIDKey),
	}})
}

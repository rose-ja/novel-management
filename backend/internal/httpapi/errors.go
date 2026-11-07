package httpapi

import (
	"github.com/gin-gonic/gin"
)

// errorBody 是全局统一的错误结构：{ "error": { "code", "message" } }。
type errorBody struct {
	Error errorItem `json:"error"`
}

type errorItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func abortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorBody{Error: errorItem{Code: code, Message: message}})
}

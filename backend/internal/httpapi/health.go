package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthDeps 让健康检查与具体基础设施解耦，便于单元测试。
type HealthDeps struct {
	DBPing          func(context.Context) error
	CachePing       func(context.Context) error // nil 表示缓存未启用
	DataDirWritable func() bool
	Version         string
}

func healthHandler(deps HealthDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		status := "ok"
		statusCode := http.StatusOK
		// 仅数据库决定整体健康：不可用时返回 503。
		dbStatus := "ok"
		if deps.DBPing == nil || deps.DBPing(ctx) != nil {
			dbStatus = "unavailable"
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		}

		cacheStatus := "disabled"
		if deps.CachePing != nil {
			cacheStatus = "ok"
			if err := deps.CachePing(ctx); err != nil {
				cacheStatus = "degraded"
			}
		}

		writable := false
		if deps.DataDirWritable != nil {
			writable = deps.DataDirWritable()
		}

		c.JSON(statusCode, gin.H{
			"status":            status,
			"version":           deps.Version,
			"db":                dbStatus,
			"cache":             cacheStatus,
			"data_dir_writable": writable,
		})
	}
}

package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/config"
)

type Options struct {
	Config config.Config
	Log    *slog.Logger
	Health HealthDeps
}

// NewRouter 组装 API 路由与静态页面托管；业务路由随后续变更逐步加入。
func NewRouter(opts Options) *gin.Engine {
	r := gin.New()
	r.Use(requestID(), accessLog(opts.Log), securityHeaders(), recovery(opts.Log), hostGuard(opts.Config.AllowedHosts))

	api := r.Group("/api/v1")
	api.GET("/health", healthHandler(opts.Health))
	api.GET("/meta", metaHandler(opts.Config))
	api.GET("/settings", settingsHandler(opts.Config))

	serveStatic := staticHandler(opts.Config.WebDir)
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			abortError(c, http.StatusNotFound, "NOT_FOUND", "接口不存在")
			return
		}
		serveStatic(c)
	})

	return r
}

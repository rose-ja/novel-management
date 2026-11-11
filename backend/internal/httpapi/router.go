package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/service"
)

type Options struct {
	Config     config.Config
	Log        *slog.Logger
	Health     HealthDeps
	Works      *service.Works
	Trash      *service.Trash
	Snapshots  *service.Snapshot
	Settings   *service.Settings
	System     *service.System
	Characters *service.Characters
	Media      *service.Media
	Chronicles *service.Chronicles
}

// NewRouter 组装 API 路由与静态页面托管；业务服务为 nil 时对应路由不注册（单元测试可用）。
func NewRouter(opts Options) *gin.Engine {
	r := gin.New()
	r.Use(requestID(), accessLog(opts.Log), securityHeaders(), recovery(opts.Log), hostGuard(opts.Config.AllowedHosts), originGuard())

	api := r.Group("/api/v1")
	api.GET("/health", healthHandler(opts.Health))
	api.GET("/meta", metaHandler(opts.Config))
	api.GET("/settings", settingsHandler(opts.Config, opts.Settings))
	if opts.Settings != nil {
		api.PATCH("/settings", updateSettingsHandler(opts.Config, opts.Settings))
	}
	if opts.Works != nil {
		registerWorks(api, opts.Works)
	}
	if opts.Trash != nil {
		registerTrash(api, opts.Trash)
	}
	if opts.Snapshots != nil {
		api.GET("/snapshots", listSnapshotsHandler(opts.Snapshots))
	}
	if opts.System != nil {
		registerSystem(api, opts.System)
	}
	if opts.Characters != nil {
		registerCharacters(api, opts.Characters)
	}
	if opts.Media != nil {
		registerCharacterImage(api, opts.Media)
		registerMedia(api, opts.Media)
	}
	if opts.Chronicles != nil {
		registerChronicles(api, opts.Chronicles)
	}

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

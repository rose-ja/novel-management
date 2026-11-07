package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/config"
)

// settingsHandler 返回只读的系统设置；作品相关字段在 CHG-002 接入数据库后生效。
func settingsHandler(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"current_work_id": nil,
			"data_dir":        cfg.DataDir,
			"snapshot_keep":   cfg.SnapshotKeep,
			"last_export_at":  nil,
			"last_restore_at": nil,
			"app_version":     cfg.Version,
		})
	}
}

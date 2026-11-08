package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

// settingsHandler 返回只读系统设置；服务未注入时退化为静态占位（单元测试用）。
func settingsHandler(cfg config.Config, svc *service.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			c.JSON(http.StatusOK, settingsPayload(cfg, nil))
			return
		}
		row, err := svc.Get(c.Request.Context())
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, settingsPayload(cfg, row))
	}
}

// updateSettingsHandler 切换当前作品；返回与 GET 相同的结构。
func updateSettingsHandler(cfg config.Config, svc *service.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			CurrentWorkID json.RawMessage `json:"current_work_id"`
		}
		if !bindJSON(c, &body) {
			return
		}
		if len(body.CurrentWorkID) == 0 {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "缺少 current_work_id", gin.H{
				"field": "current_work_id",
			})
			return
		}
		workID, _, err := parseOptionalString(body.CurrentWorkID)
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "current_work_id 必须是字符串或 null")
			return
		}
		if err := svc.SetCurrentWork(c.Request.Context(), workID); err != nil {
			abortServiceError(c, err)
			return
		}
		row, err := svc.Get(c.Request.Context())
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, settingsPayload(cfg, row))
	}
}

func settingsPayload(cfg config.Config, row *model.AppSettings) gin.H {
	payload := gin.H{
		"current_work_id": nil,
		"data_dir":        cfg.DataDir,
		"snapshot_keep":   cfg.SnapshotKeep,
		"last_export_at":  nil,
		"last_restore_at": nil,
		"app_version":     cfg.Version,
	}
	if row == nil {
		return payload
	}
	payload["current_work_id"] = row.CurrentWorkID
	payload["last_export_at"] = row.LastExportAt
	payload["last_restore_at"] = row.LastRestoreAt
	if row.SnapshotKeep > 0 {
		payload["snapshot_keep"] = int(row.SnapshotKeep)
	}
	return payload
}

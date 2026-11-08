package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

// listSnapshotsHandler 返回最近 10 条快照记录（只读，无页面内恢复入口）。
func listSnapshotsHandler(svc *service.Snapshot) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := svc.List(c.Request.Context(), 10)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		if items == nil {
			items = []model.Snapshot{}
		}
		c.JSON(http.StatusOK, items)
	}
}

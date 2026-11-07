package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/config"
)

func metaHandler(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"app_version":      cfg.Version,
			"api_version":      config.APIVersion,
			"export_versions":  []int{1},
			"template_version": "1.0",
			"cache_enabled":    cfg.IsCacheEnabled(),
			"cache_ttl_sec":    cfg.CacheTTLSec,
			"limits": gin.H{
				"name":            200,
				"description":     20000,
				"chronicle":       2000,
				"relation_note":   500,
				"era_text":        100,
				"seat_max":        200,
				"category_max":    10,
				"upload_image_mb": 10,
			},
		})
	}
}

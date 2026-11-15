package httpapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/backup"
	"novel-management/backend/internal/config"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

func registerExports(api *gin.RouterGroup, svc *service.Export, cfg config.Config) {
	api.POST("/exports", createExportHandler(svc))
	api.GET("/exports", listExportsHandler(svc))
	api.GET("/exports/:id", getExportHandler(svc))
	api.GET("/exports/:id/download", downloadExportHandler(svc, cfg))
	api.GET("/export/status", exportStatusHandler(svc))
}

type createExportRequest struct {
	IncludeTrash *bool `json:"include_trash"`
}

func createExportHandler(svc *service.Export) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createExportRequest
		if !bindJSON(c, &body) {
			return
		}
		includeTrash := true
		if body.IncludeTrash != nil {
			includeTrash = *body.IncludeTrash
		}

		record, err := svc.Create(c.Request.Context(), includeTrash)
		if err != nil {
			abortServiceError(c, err)
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"export_id": record.ID,
			"status":    record.Status,
		})

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			svc.Run(ctx, record.ID)
		}()
	}
}

func listExportsHandler(svc *service.Export) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := svc.List(c.Request.Context(), 20)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		if items == nil {
			items = []model.ExportRun{}
		}
		c.JSON(http.StatusOK, items)
	}
}

func getExportHandler(svc *service.Export) gin.HandlerFunc {
	return func(c *gin.Context) {
		record, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, record)
	}
}

func downloadExportHandler(svc *service.Export, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		record, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		if record.Status != model.ExportStatusCompleted || record.FilePath == nil {
			abortError(c, http.StatusNotFound, "NOT_FOUND", "导出文件不存在或尚未完成")
			return
		}

		path, err := backup.SafeJoin(cfg.DataDir, *record.FilePath)
		if err != nil {
			abortError(c, http.StatusNotFound, "NOT_FOUND", "导出文件不存在")
			return
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			abortError(c, http.StatusNotFound, "NOT_FOUND", "导出文件不存在")
			return
		}

		name := filepath.Base(path)
		if record.FileName != nil && *record.FileName != "" {
			name = *record.FileName
		}
		c.Header("Content-Type", "application/zip")
		c.FileAttachment(path, name)
	}
}

func exportStatusHandler(svc *service.Export) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := svc.Status(c.Request.Context())
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, status)
	}
}

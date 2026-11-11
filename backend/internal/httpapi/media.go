package httpapi

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

func registerCharacterImage(api *gin.RouterGroup, svc *service.Media) {
	group := api.Group("/characters")
	group.POST("/:id/image", uploadCharacterImageHandler(svc))
	group.DELETE("/:id/image", deleteCharacterImageHandler(svc))
}

func registerMedia(api *gin.RouterGroup, svc *service.Media) {
	api.GET("/media/:asset_id", serveMediaHandler(svc))
}

func uploadCharacterImageHandler(svc *service.Media) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > model.MaxImageBytes+(1<<20) {
			abortError(c, http.StatusUnprocessableEntity, "FILE_TOO_LARGE", "图片不能超过 10MB")
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, model.MaxImageBytes+(1<<20))
		if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "图片不能超过 10MB", gin.H{"field": "file"})
			return
		}
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请选择图片文件", gin.H{"field": "file"})
			return
		}
		defer func() { _ = file.Close() }()
		view, err := svc.UploadCharacterImage(c.Request.Context(), c.Param("id"), file, header.Size)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, view)
	}
}

func deleteCharacterImageHandler(svc *service.Media) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.DeleteCharacterImage(c.Request.Context(), c.Param("id")); err != nil {
			abortServiceError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func serveMediaHandler(svc *service.Media) gin.HandlerFunc {
	return func(c *gin.Context) {
		variant := c.DefaultQuery("variant", "thumb")
		if variant != "thumb" && variant != "original" {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "variant 仅支持 thumb 或 original", gin.H{"field": "variant"})
			return
		}
		asset, path, err := svc.ServeAsset(c.Request.Context(), c.Param("asset_id"), variant)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		file, err := os.Open(path)
		if err != nil {
			abortError(c, http.StatusNotFound, "NOT_FOUND", "资源不存在")
			return
		}
		defer func() { _ = file.Close() }()
		info, err := file.Stat()
		if err != nil {
			abortError(c, http.StatusNotFound, "NOT_FOUND", "资源不存在")
			return
		}
		etag := `"` + asset.SHA256 + "-" + variant + `"`
		c.Header("ETag", etag)
		if match := c.GetHeader("If-None-Match"); match == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Header("Cache-Control", "private, max-age=31536000, immutable")
		c.Header("Content-Disposition", `inline; filename="image"`)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Type", asset.Mime)
		http.ServeContent(c.Writer, c.Request, "", info.ModTime(), file)
	}
}

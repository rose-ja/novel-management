package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/config"
	"novel-management/backend/internal/imports"
	"novel-management/backend/internal/service"
)

const multipartMemory = 32 << 20

func registerImports(api *gin.RouterGroup, svc *service.Import, cfg config.Config, log *slog.Logger) {
	group := api.Group("/imports")
	group.GET("/template", templateHandler)
	group.GET("/template/example", exampleHandler)
	group.POST("", createImportHandler(svc, cfg, log))
	group.POST("/:id/validate", validateImportHandler(svc, log))
	group.GET("/:id", importDetailHandler(svc))
	group.GET("/:id/report", importReportHandler(svc))
	group.POST("/:id/commit", commitImportHandler(svc))
}

func templateHandler(c *gin.Context) {
	c.Header("Content-Disposition", `attachment; filename="novel-import-template-v1.0.md"`)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", imports.Markdown())
}

func exampleHandler(c *gin.Context) {
	data, err := imports.ExampleZip()
	if err != nil {
		abortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "示例包生成失败")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="novel-import-example.zip"`)
	c.Data(http.StatusOK, "application/zip", data)
}

func createImportHandler(svc *service.Import, cfg config.Config, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		maxBody := int64(cfg.MaxUploadMB)<<20 + 32<<20
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
		if err := c.Request.ParseMultipartForm(multipartMemory); err != nil {
			abortError(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "上传文件过大或格式不正确")
			return
		}
		defer func() {
			if c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()

		markdownHeader, err := singleFile(c, "markdown")
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
			return
		}
		markdown, err := markdownHeader.Open()
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "无法读取 Markdown")
			return
		}
		defer func() { _ = markdown.Close() }()

		var archive io.Reader
		var archiveName string
		var archiveSize int64
		if headers := c.Request.MultipartForm.File["archive"]; len(headers) > 0 {
			archiveHeader, err := singleFile(c, "archive")
			if err != nil {
				abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
				return
			}
			file, err := archiveHeader.Open()
			if err != nil {
				abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "无法读取图片 zip")
				return
			}
			defer func() { _ = file.Close() }()
			archive = file
			archiveName = archiveHeader.Filename
			archiveSize = archiveHeader.Size
		}

		run, err := svc.CreateUpload(c.Request.Context(), markdownHeader.Filename, markdown, markdownHeader.Size, archiveName, archive, archiveSize)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		go func() {
			if err := svc.RunValidate(context.Background(), run.ID); err != nil {
				log.Warn("导入校验失败", "import_id", run.ID, "error", err)
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"import_id": run.ID, "status": run.Status})
	}
}

func singleFile(c *gin.Context, field string) (*multipart.FileHeader, error) {
	headers := c.Request.MultipartForm.File[field]
	if len(headers) == 0 {
		if field == "markdown" {
			return nil, errors.New("请上传固定模板 Markdown（.md）")
		}
		return nil, errors.New("每类文件只能上传一个")
	}
	if len(headers) > 1 {
		return nil, errors.New("每类文件只能上传一个")
	}
	return headers[0], nil
}

func validateImportHandler(svc *service.Import, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if _, err := svc.Get(c.Request.Context(), id); err != nil {
			abortServiceError(c, err)
			return
		}
		go func() {
			if err := svc.RunValidate(context.Background(), id); err != nil {
				log.Warn("导入重新校验失败", "import_id", id, "error", err)
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"import_id": id, "status": "validating"})
	}
}

func importDetailHandler(svc *service.Import) gin.HandlerFunc {
	return func(c *gin.Context) {
		run, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, service.Detail(run))
	}
}

func importReportHandler(svc *service.Import) gin.HandlerFunc {
	return func(c *gin.Context) {
		report, err := svc.Report(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, report)
	}
}

func commitImportHandler(svc *service.Import) gin.HandlerFunc {
	type request struct {
		WorkName string `json:"work_name"`
	}
	return func(c *gin.Context) {
		var body request
		if c.Request.ContentLength > 0 {
			if !bindJSON(c, &body) {
				return
			}
		}
		report, err := svc.Commit(c.Request.Context(), c.Param("id"), body.WorkName)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":     "completed",
			"import_id":  report.ImportID,
			"work_id":    report.WorkID,
			"report":     report,
			"review_url": "/import?step=review&import_id=" + report.ImportID,
		})
	}
}

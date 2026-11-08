package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerSystem(api *gin.RouterGroup, svc *service.System) {
	group := api.Group("/system-categories")
	group.GET("", listSystemCategoriesHandler(svc))
	group.POST("", createSystemCategoryHandler(svc))
	group.PATCH("/:id", updateSystemCategoryHandler(svc))
	group.DELETE("/:id", deleteSystemCategoryHandler(svc))
}

func listSystemCategoriesHandler(svc *service.System) gin.HandlerFunc {
	return func(c *gin.Context) {
		includeDisabled := true
		if raw := strings.TrimSpace(c.Query("include_disabled")); raw != "" {
			parsed, err := strconv.ParseBool(raw)
			if err != nil {
				abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "include_disabled 必须是布尔值", gin.H{
					"field": "include_disabled",
				})
				return
			}
			includeDisabled = parsed
		}

		items, err := svc.ListCategories(c.Request.Context(), c.Query("work_id"), includeDisabled)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func createSystemCategoryHandler(svc *service.System) gin.HandlerFunc {
	type request struct {
		WorkID       string  `json:"work_id"`
		Name         string  `json:"name"`
		CategoryType string  `json:"category_type"`
		IsEnabled    *bool   `json:"is_enabled"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		item, err := svc.CreateCategory(c.Request.Context(), service.CreateCategoryInput{
			WorkID:       body.WorkID,
			Name:         body.Name,
			CategoryType: body.CategoryType,
			IsEnabled:    body.IsEnabled,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, item)
	}
}

func updateSystemCategoryHandler(svc *service.System) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name         json.RawMessage `json:"name"`
			IsEnabled    json.RawMessage `json:"is_enabled"`
			CategoryType json.RawMessage `json:"category_type"`
		}
		if !bindJSON(c, &body) {
			return
		}
		if len(body.CategoryType) > 0 {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "category_type 不可修改", gin.H{
				"field": "category_type",
			})
			return
		}

		var name *string
		if len(body.Name) > 0 {
			parsed, clear, err := parseOptionalString(body.Name)
			if err != nil || clear || parsed == nil {
				abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name 必须是非空字符串", gin.H{
					"field": "name",
				})
				return
			}
			name = parsed
		}

		var enabled *bool
		if len(body.IsEnabled) > 0 {
			var value *bool
			if err := json.Unmarshal(body.IsEnabled, &value); err != nil || value == nil {
				abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "is_enabled 必须是布尔值", gin.H{
					"field": "is_enabled",
				})
				return
			}
			enabled = value
		}

		item, err := svc.UpdateCategory(c.Request.Context(), c.Param("id"), service.UpdateCategoryInput{
			Name:      name,
			IsEnabled: enabled,
			IfMatch:   c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func deleteSystemCategoryHandler(svc *service.System) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.DeleteCategory(c.Request.Context(), c.Param("id"), service.DeleteCategoryInput{
			IfMatch: c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

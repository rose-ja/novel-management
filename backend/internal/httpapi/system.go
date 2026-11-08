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
	group.GET("/:id/nodes", listSystemNodesHandler(svc))
	group.PATCH("/:id", updateSystemCategoryHandler(svc))
	group.DELETE("/:id", deleteSystemCategoryHandler(svc))

	nodes := api.Group("/system-nodes")
	nodes.POST("", createSystemNodeHandler(svc))
	nodes.POST("/reorder", reorderSystemNodesHandler(svc))
	nodes.PATCH("/:id", updateSystemNodeHandler(svc))
	nodes.DELETE("/:id", deleteSystemNodeHandler(svc))
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

func abortField(c *gin.Context, field, message string) {
	abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, gin.H{"field": field})
}

func listSystemNodesHandler(svc *service.System) gin.HandlerFunc {
	return func(c *gin.Context) {
		flat := strings.EqualFold(strings.TrimSpace(c.Query("flat")), "true")
		items, err := svc.ListNodes(c.Request.Context(), c.Param("id"), flat, c.Query("review_status"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func createSystemNodeHandler(svc *service.System) gin.HandlerFunc {
	type request struct {
		CategoryID   string  `json:"category_id"`
		ParentID     *string `json:"parent_id"`
		Name         string  `json:"name"`
		EraStartText *string `json:"era_start_text"`
		EraEndText   *string `json:"era_end_text"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		node, err := svc.CreateNode(c.Request.Context(), service.CreateNodeInput{
			CategoryID:   body.CategoryID,
			ParentID:     body.ParentID,
			Name:         body.Name,
			EraStartText: body.EraStartText,
			EraEndText:   body.EraEndText,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, node)
	}
}

func updateSystemNodeHandler(svc *service.System) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name         json.RawMessage `json:"name"`
			SortOrder    json.RawMessage `json:"sort_order"`
			EraStartText json.RawMessage `json:"era_start_text"`
			EraEndText   json.RawMessage `json:"era_end_text"`
			ParentID     json.RawMessage `json:"parent_id"`
			CategoryType json.RawMessage `json:"category_type"`
		}
		if !bindJSON(c, &body) {
			return
		}
		if len(body.ParentID) > 0 {
			abortField(c, "parent_id", "parent_id 不可修改")
			return
		}
		if len(body.CategoryType) > 0 {
			abortField(c, "category_type", "category_type 不可修改")
			return
		}

		var name *string
		if len(body.Name) > 0 {
			parsed, clear, err := parseOptionalString(body.Name)
			if err != nil || clear || parsed == nil {
				abortField(c, "name", "name 必须是非空字符串")
				return
			}
			name = parsed
		}

		var sortOrder *int
		if len(body.SortOrder) > 0 {
			var value *int
			if err := json.Unmarshal(body.SortOrder, &value); err != nil || value == nil {
				abortField(c, "sort_order", "sort_order 必须是整数")
				return
			}
			sortOrder = value
		}

		var eraStart *string
		clearEraStart := false
		if len(body.EraStartText) > 0 {
			parsed, clear, err := parseOptionalString(body.EraStartText)
			if err != nil {
				abortField(c, "era_start_text", "era_start_text 必须是字符串或 null")
				return
			}
			eraStart, clearEraStart = parsed, clear
		}

		var eraEnd *string
		clearEraEnd := false
		if len(body.EraEndText) > 0 {
			parsed, clear, err := parseOptionalString(body.EraEndText)
			if err != nil {
				abortField(c, "era_end_text", "era_end_text 必须是字符串或 null")
				return
			}
			eraEnd, clearEraEnd = parsed, clear
		}

		node, err := svc.UpdateNode(c.Request.Context(), c.Param("id"), service.UpdateNodeInput{
			Name:          name,
			SortOrder:     sortOrder,
			EraStartText:  eraStart,
			ClearEraStart: clearEraStart,
			EraEndText:    eraEnd,
			ClearEraEnd:   clearEraEnd,
			IfMatch:       c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, node)
	}
}

func deleteSystemNodeHandler(svc *service.System) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.DeleteNode(c.Request.Context(), c.Param("id"), service.DeleteNodeInput{
			IfMatch: c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func reorderSystemNodesHandler(svc *service.System) gin.HandlerFunc {
	type request struct {
		CategoryID string   `json:"category_id"`
		ParentID   *string  `json:"parent_id"`
		OrderedIDs []string `json:"ordered_ids"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		if strings.TrimSpace(body.CategoryID) == "" {
			abortField(c, "category_id", "category_id 不能为空")
			return
		}
		if body.OrderedIDs == nil {
			abortField(c, "ordered_ids", "ordered_ids 不能为空")
			return
		}
		if err := svc.ReorderNodes(c.Request.Context(), service.ReorderNodesInput{
			CategoryID: body.CategoryID,
			ParentID:   body.ParentID,
			OrderedIDs: body.OrderedIDs,
		}); err != nil {
			abortServiceError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

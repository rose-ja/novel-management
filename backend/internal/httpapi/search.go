package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

func registerSearch(api *gin.RouterGroup, svc *service.Search) {
	api.GET("/search", searchHandler(svc))
}

func searchHandler(svc *service.Search) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 0
		if raw := strings.TrimSpace(c.Query("page")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "page 必须为正整数", gin.H{"field": "page"})
				return
			}
			page = parsed
		}
		result, err := svc.Search(c.Request.Context(), service.SearchInput{
			WorkID:           c.Query("work_id"),
			Query:            c.Query("q"),
			EntityTypes:      c.Query("entity_types"),
			FactionID:        c.Query("faction_id"),
			FactionScope:     c.Query("faction_scope"),
			SystemCategoryID: c.Query("system_category_id"),
			SystemNodeID:     c.Query("system_node_id"),
			LocationID:       c.Query("location_id"),
			RankingID:        c.Query("ranking_id"),
			ReviewStatus:     c.Query("review_status"),
			Page:             page,
		})
		if err != nil {
			if errors.Is(err, model.ErrSearchQueryRequired) {
				abortError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}
			abortServiceError(c, err)
			return
		}
		c.Header("X-Cache", "BYPASS")
		c.JSON(http.StatusOK, result)
	}
}

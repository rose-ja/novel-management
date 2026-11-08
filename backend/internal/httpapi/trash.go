package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerTrash(api *gin.RouterGroup, svc *service.Trash) {
	group := api.Group("/trash")
	group.GET("", listTrashHandler(svc))
	group.POST("/undo", undoHandler(svc))
	group.DELETE("", purgeAllHandler(svc))
	group.POST("/:type/:id/restore", restoreHandler(svc))
	group.DELETE("/:type/:id", purgeHandler(svc))
}

func listTrashHandler(svc *service.Trash) gin.HandlerFunc {
	return func(c *gin.Context) {
		rootType := c.Query("root_type")
		if rootType == "" {
			rootType = c.Query("type")
		}
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))

		result, err := svc.List(c.Request.Context(), rootType, page, pageSize)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func restoreHandler(svc *service.Trash) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body service.RestoreInput
		if !bindJSON(c, &body) {
			return
		}
		result, err := svc.Restore(c.Request.Context(), c.Param("type"), c.Param("id"), body)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func purgeHandler(svc *service.Trash) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Confirm bool `json:"confirm"`
		}
		if !bindJSON(c, &body) {
			return
		}
		if !body.Confirm {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "永久删除需要 confirm=true", gin.H{
				"field": "confirm",
			})
			return
		}
		if err := svc.Purge(c.Request.Context(), c.Param("type"), c.Param("id")); err != nil {
			abortServiceError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func purgeAllHandler(svc *service.Trash) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Confirm string  `json:"confirm"`
			WorkID  *string `json:"work_id"`
		}
		if !bindJSON(c, &body) {
			return
		}
		result, err := svc.PurgeAll(c.Request.Context(), body.Confirm)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func undoHandler(svc *service.Trash) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			UndoToken string `json:"undo_token"`
		}
		if !bindJSON(c, &body) {
			return
		}
		result, err := svc.Undo(c.Request.Context(), body.UndoToken)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

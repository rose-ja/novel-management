package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerWorks(api *gin.RouterGroup, svc *service.Works) {
	group := api.Group("/works")
	group.GET("", listWorksHandler(svc))
	group.POST("", createWorkHandler(svc))
	group.GET("/:id", getWorkHandler(svc))
	group.PATCH("/:id", updateWorkHandler(svc))
	group.DELETE("/:id", deleteWorkHandler(svc))
}

func listWorksHandler(svc *service.Works) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))

		result, err := svc.List(c.Request.Context(), c.Query("q"), page, pageSize)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func createWorkHandler(svc *service.Works) gin.HandlerFunc {
	type request struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		work, err := svc.Create(c.Request.Context(), service.CreateWorkInput{
			Name:        body.Name,
			Description: body.Description,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, work)
	}
}

func getWorkHandler(svc *service.Works) gin.HandlerFunc {
	return func(c *gin.Context) {
		work, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, work)
	}
}

func updateWorkHandler(svc *service.Works) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name        *string         `json:"name"`
			Description json.RawMessage `json:"description"`
		}
		if !bindJSON(c, &body) {
			return
		}
		description, clear, err := parseOptionalString(body.Description)
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "description 必须是字符串或 null")
			return
		}

		work, err := svc.Update(c.Request.Context(), c.Param("id"), service.UpdateWorkInput{
			Name:             body.Name,
			Description:      description,
			ClearDescription: clear,
			IfMatch:          c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, work)
	}
}

func deleteWorkHandler(svc *service.Works) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			ConfirmName string `json:"confirm_name"`
		}
		if !bindJSON(c, &body) {
			return
		}
		result, err := svc.Delete(c.Request.Context(), c.Param("id"), service.DeleteWorkInput{
			ConfirmName: body.ConfirmName,
			IfMatch:     c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

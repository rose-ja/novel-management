package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerChronicles(api *gin.RouterGroup, svc *service.Chronicles) {
	group := api.Group("/chronicles")
	group.GET("", listChroniclesHandler(svc))
	group.POST("", createChronicleHandler(svc))
	group.PATCH("/:id", updateChronicleHandler(svc))
	group.DELETE("/:id", deleteChronicleHandler(svc))
}

func listChroniclesHandler(svc *service.Chronicles) gin.HandlerFunc {
	return func(c *gin.Context) {
		ownerType := strings.TrimSpace(c.Query("owner_type"))
		ownerID := strings.TrimSpace(c.Query("owner_id"))
		if ownerType == "" || ownerID == "" {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "owner_type 与 owner_id 必填", gin.H{"field": "owner_id"})
			return
		}
		items, err := svc.List(c.Request.Context(), ownerType, ownerID)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func createChronicleHandler(svc *service.Chronicles) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			OwnerType string  `json:"owner_type"`
			OwnerID   string  `json:"owner_id"`
			Text      string  `json:"text"`
			TimeLabel *string `json:"time_label"`
		}
		if !bindJSON(c, &body) {
			return
		}
		item, err := svc.Create(c.Request.Context(), service.ChronicleInput{
			OwnerType: strings.TrimSpace(body.OwnerType),
			OwnerID:   strings.TrimSpace(body.OwnerID),
			Text:      body.Text,
			TimeLabel: body.TimeLabel,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.Header("Location", "/api/v1/chronicles/"+item.ID)
		c.JSON(http.StatusCreated, item)
	}
}

func updateChronicleHandler(svc *service.Chronicles) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Text      *string         `json:"text"`
			TimeLabel json.RawMessage `json:"time_label"`
			SortOrder *int            `json:"sort_order"`
		}
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求体格式不正确", gin.H{"field": "body"})
			return
		}
		in := service.UpdateChronicleInput{
			Text:      body.Text,
			SortOrder: body.SortOrder,
			IfMatch:   c.GetHeader("If-Match"),
		}
		if len(body.TimeLabel) > 0 {
			if bytes.Equal(bytes.TrimSpace(body.TimeLabel), []byte("null")) {
				in.ClearTimeLabel = true
			} else {
				var label string
				if err := json.Unmarshal(body.TimeLabel, &label); err != nil {
					abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "time_label 必须是字符串或 null", gin.H{"field": "time_label"})
					return
				}
				in.TimeLabel = &label
			}
		}
		item, err := svc.Update(c.Request.Context(), c.Param("id"), in)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func deleteChronicleHandler(svc *service.Chronicles) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.Delete(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

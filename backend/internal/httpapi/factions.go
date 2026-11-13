package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerFactions(api *gin.RouterGroup, svc *service.Factions) {
	group := api.Group("/factions")
	group.GET("", listFactionsHandler(svc))
	group.POST("", createFactionHandler(svc))
	group.GET("/:id", getFactionHandler(svc))
	group.PATCH("/:id", updateFactionHandler(svc))
	group.DELETE("/:id", deleteFactionHandler(svc))
	group.GET("/:id/members", listFactionMembersHandler(svc))
}

func listFactionsHandler(svc *service.Factions) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.List(c.Request.Context(), service.ListFactionsInput{
			WorkID:       c.Query("work_id"),
			Query:        c.Query("q"),
			ParentID:     c.Query("parent_id"),
			ReviewStatus: c.Query("review_status"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func createFactionHandler(svc *service.Factions) gin.HandlerFunc {
	type request struct {
		WorkID            string          `json:"work_id"`
		Name              string          `json:"name"`
		ParentID          json.RawMessage `json:"parent_id"`
		LeaderCharacterID json.RawMessage `json:"leader_character_id"`
		Description       *string         `json:"description"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		parentID, _, err := parseOptionalString(body.ParentID)
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "parent_id 必须是字符串或 null", gin.H{"field": "parent_id"})
			return
		}
		leaderID, _, err := parseOptionalString(body.LeaderCharacterID)
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "leader_character_id 必须是字符串或 null", gin.H{"field": "leader_character_id"})
			return
		}
		detail, err := svc.Create(c.Request.Context(), service.CreateFactionInput{
			WorkID:            body.WorkID,
			Name:              body.Name,
			ParentID:          parentID,
			LeaderCharacterID: leaderID,
			Description:       body.Description,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.Header("Location", "/api/v1/factions/"+detail.ID)
		c.JSON(http.StatusCreated, detail)
	}
}

func getFactionHandler(svc *service.Factions) gin.HandlerFunc {
	return func(c *gin.Context) {
		detail, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func updateFactionHandler(svc *service.Factions) gin.HandlerFunc {
	type request struct {
		Name              *string         `json:"name"`
		ParentID          json.RawMessage `json:"parent_id"`
		LeaderCharacterID json.RawMessage `json:"leader_character_id"`
		Description       json.RawMessage `json:"description"`
	}
	return func(c *gin.Context) {
		var body request
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求体格式不正确")
			return
		}
		parentID, clearParent, err := parseOptionalString(body.ParentID)
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "parent_id 必须是字符串或 null", gin.H{"field": "parent_id"})
			return
		}
		leaderID, clearLeader, err := parseOptionalString(body.LeaderCharacterID)
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "leader_character_id 必须是字符串或 null", gin.H{"field": "leader_character_id"})
			return
		}
		description, clearDescription, err := parseOptionalString(body.Description)
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "description 必须是字符串或 null")
			return
		}
		detail, err := svc.Update(c.Request.Context(), c.Param("id"), service.UpdateFactionInput{
			Name:              body.Name,
			ParentID:          parentID,
			ClearParent:       clearParent,
			LeaderCharacterID: leaderID,
			ClearLeader:       clearLeader,
			Description:       description,
			ClearDescription:  clearDescription,
			IfMatch:           c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func deleteFactionHandler(svc *service.Factions) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.Delete(c.Request.Context(), c.Param("id"), c.GetHeader("If-Match"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func listFactionMembersHandler(svc *service.Factions) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))
		result, err := svc.Members(c.Request.Context(), c.Param("id"), c.Query("scope"), page, pageSize)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

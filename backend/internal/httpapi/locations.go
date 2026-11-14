package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerLocations(api *gin.RouterGroup, svc *service.Locations) {
	group := api.Group("/locations")
	group.GET("", listLocationsHandler(svc))
	group.POST("", createLocationHandler(svc))
	group.GET("/:id", getLocationHandler(svc))
	group.PATCH("/:id", updateLocationHandler(svc))
	group.DELETE("/:id", deleteLocationHandler(svc))
	group.GET("/:id/characters", listLocationCharactersHandler(svc))
	group.GET("/:id/factions", listLocationFactionsHandler(svc))
	group.POST("/:id/factions", addLocationFactionHandler(svc))
	group.PATCH("/:id/factions/:faction_id", updateLocationFactionNoteHandler(svc))
	group.DELETE("/:id/factions/:faction_id", removeLocationFactionHandler(svc))
}

func listLocationsHandler(svc *service.Locations) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.List(c.Request.Context(), service.ListLocationsInput{
			WorkID:       c.Query("work_id"),
			Query:        c.Query("q"),
			ParentID:     c.Query("parent_id"),
			TypeText:     c.Query("type_text"),
			ReviewStatus: c.Query("review_status"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func createLocationHandler(svc *service.Locations) gin.HandlerFunc {
	type request struct {
		WorkID      string          `json:"work_id"`
		Name        string          `json:"name"`
		ParentID    json.RawMessage `json:"parent_id"`
		TypeText    *string         `json:"type_text"`
		Description *string         `json:"description"`
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
		detail, err := svc.Create(c.Request.Context(), service.CreateLocationInput{
			WorkID:      body.WorkID,
			Name:        body.Name,
			ParentID:    parentID,
			TypeText:    body.TypeText,
			Description: body.Description,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.Header("Location", "/api/v1/locations/"+detail.ID)
		c.JSON(http.StatusCreated, detail)
	}
}

func getLocationHandler(svc *service.Locations) gin.HandlerFunc {
	return func(c *gin.Context) {
		detail, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func updateLocationHandler(svc *service.Locations) gin.HandlerFunc {
	type request struct {
		Name        *string         `json:"name"`
		ParentID    json.RawMessage `json:"parent_id"`
		TypeText    json.RawMessage `json:"type_text"`
		Description json.RawMessage `json:"description"`
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
		typeText, clearType, err := parseOptionalString(body.TypeText)
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "type_text 必须是字符串或 null", gin.H{"field": "type_text"})
			return
		}
		description, clearDescription, err := parseOptionalString(body.Description)
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "description 必须是字符串或 null")
			return
		}
		detail, err := svc.Update(c.Request.Context(), c.Param("id"), service.UpdateLocationInput{
			Name:             body.Name,
			ParentID:         parentID,
			ClearParent:      clearParent,
			TypeText:         typeText,
			ClearType:        clearType,
			Description:      description,
			ClearDescription: clearDescription,
			IfMatch:          c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func deleteLocationHandler(svc *service.Locations) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.Delete(c.Request.Context(), c.Param("id"), c.GetHeader("If-Match"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func listLocationCharactersHandler(svc *service.Locations) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := svc.ListCharacters(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func listLocationFactionsHandler(svc *service.Locations) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := svc.ListFactions(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func addLocationFactionHandler(svc *service.Locations) gin.HandlerFunc {
	type request struct {
		FactionID string  `json:"faction_id"`
		Note      *string `json:"note"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		rel, err := svc.AddFaction(c.Request.Context(), c.Param("id"), service.AddLocationFactionInput{
			FactionID: body.FactionID,
			Note:      body.Note,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, rel)
	}
}

func updateLocationFactionNoteHandler(svc *service.Locations) gin.HandlerFunc {
	type request struct {
		Note *string `json:"note"`
	}
	return func(c *gin.Context) {
		var body request
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求体格式不正确")
			return
		}
		rel, err := svc.UpdateFactionNote(c.Request.Context(), c.Param("id"), c.Param("faction_id"), body.Note)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, rel)
	}
}

func removeLocationFactionHandler(svc *service.Locations) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.RemoveFaction(c.Request.Context(), c.Param("id"), c.Param("faction_id")); err != nil {
			abortServiceError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

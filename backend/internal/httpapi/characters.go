package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerCharacters(api *gin.RouterGroup, svc *service.Characters) {
	group := api.Group("/characters")
	group.GET("", listCharactersHandler(svc))
	group.POST("", createCharacterHandler(svc))
	group.GET("/:id", getCharacterHandler(svc))
	group.PATCH("/:id", updateCharacterHandler(svc))
	group.DELETE("/:id", deleteCharacterHandler(svc))
}

func listCharactersHandler(svc *service.Characters) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))

		result, err := svc.List(c.Request.Context(), service.ListCharactersInput{
			WorkID:       c.Query("work_id"),
			Query:        c.Query("q"),
			SystemNodeID: c.Query("system_node_id"),
			FactionID:    c.Query("faction_id"),
			LocationID:   c.Query("location_id"),
			ReviewStatus: c.Query("review_status"),
			Sort:         c.Query("sort"),
			Page:         page,
			PageSize:     pageSize,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func createCharacterHandler(svc *service.Characters) gin.HandlerFunc {
	type request struct {
		WorkID        string                               `json:"work_id"`
		Name          string                               `json:"name"`
		Names         []service.CharacterNameInput         `json:"names"`
		Description   *string                              `json:"description"`
		FactionID     *string                              `json:"faction_id"`
		SystemRefs    []service.CharacterSystemRefInput    `json:"system_refs"`
		LocationLinks []service.CharacterLocationLinkInput `json:"location_links"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		detail, err := svc.Create(c.Request.Context(), service.CreateCharacterInput{
			WorkID:        body.WorkID,
			Name:          body.Name,
			Names:         body.Names,
			Description:   body.Description,
			FactionID:     body.FactionID,
			SystemRefs:    body.SystemRefs,
			LocationLinks: body.LocationLinks,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.Header("Location", "/api/v1/characters/"+detail.ID)
		c.JSON(http.StatusCreated, detail)
	}
}

func getCharacterHandler(svc *service.Characters) gin.HandlerFunc {
	return func(c *gin.Context) {
		detail, err := svc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func deleteCharacterHandler(svc *service.Characters) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.Delete(c.Request.Context(), c.Param("id"), c.GetHeader("If-Match"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func updateCharacterHandler(svc *service.Characters) gin.HandlerFunc {
	type request struct {
		Name          *string                               `json:"name"`
		Names         *[]service.CharacterNameInput         `json:"names"`
		Description   json.RawMessage                       `json:"description"`
		FactionID     json.RawMessage                       `json:"faction_id"`
		SystemRefs    *[]service.CharacterSystemRefInput    `json:"system_refs"`
		LocationLinks *[]service.CharacterLocationLinkInput `json:"location_links"`
	}
	return func(c *gin.Context) {
		var body request
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求体格式不正确")
			return
		}
		description, clearDescription, err := parseOptionalString(body.Description)
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "description 必须是字符串或 null")
			return
		}
		factionID, clearFaction, err := parseOptionalString(body.FactionID)
		if err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "faction_id 必须是字符串或 null")
			return
		}

		detail, err := svc.Update(c.Request.Context(), c.Param("id"), service.UpdateCharacterInput{
			Name:             body.Name,
			Names:            body.Names,
			Description:      description,
			ClearDescription: clearDescription,
			FactionID:        factionID,
			ClearFaction:     clearFaction,
			SystemRefs:       body.SystemRefs,
			LocationLinks:    body.LocationLinks,
			IfMatch:          c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

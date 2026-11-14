package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/service"
)

func registerRankings(api *gin.RouterGroup, svc *service.Rankings) {
	group := api.Group("/rankings")
	group.GET("", listRankingsHandler(svc))
	group.POST("", createRankingHandler(svc))
	group.GET("/:id", getRankingHandler(svc))
	group.PATCH("/:id", updateRankingHandler(svc))
	group.DELETE("/:id", deleteRankingHandler(svc))
	group.GET("/:id/seats", listRankingSeatsHandler(svc))
	group.PUT("/:id/seats/:seat_no", putRankingSeatHandler(svc))
	group.POST("/:id/seats/:seat_no/move", moveRankingSeatHandler(svc))
}

func listRankingsHandler(svc *service.Rankings) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))
		result, err := svc.List(c.Request.Context(), service.ListRankingsInput{
			WorkID:       c.Query("work_id"),
			Query:        c.Query("q"),
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

func createRankingHandler(svc *service.Rankings) gin.HandlerFunc {
	type request struct {
		WorkID    string `json:"work_id"`
		Name      string `json:"name"`
		SeatCount int    `json:"seat_count"`
	}
	return func(c *gin.Context) {
		var body request
		if !bindJSON(c, &body) {
			return
		}
		detail, err := svc.Create(c.Request.Context(), service.CreateRankingInput{
			WorkID:    body.WorkID,
			Name:      body.Name,
			SeatCount: body.SeatCount,
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.Header("Location", "/api/v1/rankings/"+detail.ID)
		c.JSON(http.StatusCreated, detail)
	}
}

func getRankingHandler(svc *service.Rankings) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))
		detail, err := svc.Get(c.Request.Context(), c.Param("id"), page, pageSize)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func updateRankingHandler(svc *service.Rankings) gin.HandlerFunc {
	type request struct {
		Name      *string `json:"name"`
		SeatCount *int    `json:"seat_count"`
	}
	return func(c *gin.Context) {
		var body request
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求体格式不正确")
			return
		}
		detail, err := svc.Update(c.Request.Context(), c.Param("id"), service.UpdateRankingInput{
			Name:      body.Name,
			SeatCount: body.SeatCount,
			IfMatch:   c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, detail)
	}
}

func deleteRankingHandler(svc *service.Rankings) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.Delete(c.Request.Context(), c.Param("id"), c.GetHeader("If-Match"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func listRankingSeatsHandler(svc *service.Rankings) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.Query("page"))
		pageSize, _ := strconv.Atoi(c.Query("page_size"))
		occupied, ok := parseOccupied(c.Query("occupied"))
		if !ok {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "occupied 仅支持 true 或 false", gin.H{"field": "occupied"})
			return
		}
		result, err := svc.ListSeats(c.Request.Context(), c.Param("id"), page, pageSize, occupied)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func putRankingSeatHandler(svc *service.Rankings) gin.HandlerFunc {
	type request struct {
		CharacterID json.RawMessage `json:"character_id"`
		Note        json.RawMessage `json:"note"`
	}
	return func(c *gin.Context) {
		seatNo, ok := parseSeatNo(c)
		if !ok {
			return
		}
		var body request
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			abortError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求体格式不正确")
			return
		}
		characterID, clearCharacter, err := parseOptionalString(body.CharacterID)
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "character_id 必须是字符串或 null", gin.H{"field": "character_id"})
			return
		}
		note, clearNote, err := parseOptionalString(body.Note)
		if err != nil {
			abortErrorWithDetails(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "note 必须是字符串或 null", gin.H{"field": "note"})
			return
		}
		seat, err := svc.PutSeat(c.Request.Context(), c.Param("id"), seatNo, service.PutSeatInput{
			CharacterID:  characterID,
			CharacterSet: clearCharacter || characterID != nil,
			Note:         note,
			NoteSet:      clearNote || note != nil,
			IfMatch:      c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, seat)
	}
}

func moveRankingSeatHandler(svc *service.Rankings) gin.HandlerFunc {
	type request struct {
		TargetSeatNo int    `json:"target_seat_no"`
		Mode         string `json:"mode"`
	}
	return func(c *gin.Context) {
		seatNo, ok := parseSeatNo(c)
		if !ok {
			return
		}
		var body request
		if !bindJSON(c, &body) {
			return
		}
		result, err := svc.MoveSeat(c.Request.Context(), c.Param("id"), seatNo, service.MoveSeatInput{
			TargetSeatNo: body.TargetSeatNo,
			Mode:         body.Mode,
			IfMatch:      c.GetHeader("If-Match"),
		})
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// parseSeatNo 解析路径中的席位号；失败按越界处理。
func parseSeatNo(c *gin.Context) (int, bool) {
	seatNo, err := strconv.Atoi(c.Param("seat_no"))
	if err != nil || seatNo < 1 {
		abortErrorWithDetails(c, http.StatusBadRequest, "SEAT_OUT_OF_RANGE", "席位号超出榜单范围", gin.H{
			"seat_no": seatNo,
		})
		return 0, false
	}
	return seatNo, true
}

func parseOccupied(raw string) (*bool, bool) {
	switch raw {
	case "":
		return nil, true
	case "true":
		value := true
		return &value, true
	case "false":
		value := false
		return &value, true
	default:
		return nil, false
	}
}

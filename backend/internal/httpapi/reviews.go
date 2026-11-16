package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/service"
)

func registerReviews(api *gin.RouterGroup, svc *service.Review) {
	api.POST("/reviews/batch", batchReviewHandler(svc))
	api.POST("/:resource/:id/review", reviewHandler(svc))
}

func reviewHandler(svc *service.Review) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.Confirm(c.Request.Context(), c.Param("resource"), c.Param("id"))
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func batchReviewHandler(svc *service.Review) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.ReviewBatchInput
		if !bindJSON(c, &input) {
			return
		}
		result, err := svc.Batch(c.Request.Context(), input)
		if err != nil {
			abortServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

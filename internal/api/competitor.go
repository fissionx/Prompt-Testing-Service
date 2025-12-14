package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
)

// suggestCompetitors handles GET /api/v1/competitors/suggest
func (s *Server) suggestCompetitors(c *gin.Context) {
	var req models.SuggestCompetitorsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand parameter is required")
		return
	}

	ctx := c.Request.Context()

	response, err := s.competitorService.SuggestCompetitors(
		ctx,
		req.Brand,
		req.Website,
		req.Description,
		req.Category,
		req.ForceRefresh,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to suggest competitors: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: response.Message,
	})
}

// saveCompetitors handles POST /api/v1/competitors
func (s *Server) saveCompetitors(c *gin.Context) {
	var req models.SaveCompetitorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	if len(req.Competitors) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one competitor is required")
		return
	}

	ctx := c.Request.Context()

	response, err := s.competitorService.SaveCompetitors(
		ctx,
		req.Brand,
		req.Competitors,
		req.Source,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to save competitors: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: response.Message,
	})
}

// getCompetitors handles GET /api/v1/competitors
func (s *Server) getCompetitors(c *gin.Context) {
	brand := c.Query("brand")
	if brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand parameter is required")
		return
	}

	ctx := c.Request.Context()

	response, err := s.competitorService.GetCompetitors(ctx, brand)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get competitors: "+err.Error())
		return
	}

	message := "Competitors retrieved successfully"
	if len(response.Competitors) == 0 {
		message = "No competitors saved for this brand"
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: message,
	})
}

// deleteCompetitors handles DELETE /api/v1/competitors
func (s *Server) deleteCompetitors(c *gin.Context) {
	brand := c.Query("brand")
	if brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand parameter is required")
		return
	}

	ctx := c.Request.Context()

	err := s.competitorService.DeleteCompetitors(ctx, brand)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to delete competitors: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Competitors deleted successfully",
	})
}

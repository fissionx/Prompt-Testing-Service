package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
)

// listCompetitors handles GET /api/v1/geo/competitors
// Returns the list of competitors for a brand with their logos and websites
func (s *Server) listCompetitors(c *gin.Context) {
	brand := c.Query("brand")
	if brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	includeCustom := c.Query("includeCustom") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	ctx := c.Request.Context()

	competitorService := services.NewCompetitorService(s.db)
	result, err := competitorService.ListCompetitors(ctx, brand, includeCustom, limit)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list competitors: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
		Message: "Competitors retrieved successfully",
	})
}

// discoverCompetitors handles POST /api/v1/geo/competitors/discover
// Discovers competitors from response data
func (s *Server) discoverCompetitors(c *gin.Context) {
	var req models.DiscoverCompetitorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	ctx := c.Request.Context()

	competitorService := services.NewCompetitorService(s.db)
	result, err := competitorService.DiscoverCompetitors(
		ctx,
		req.Brand,
		req.StartTime,
		req.EndTime,
		req.Limit,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to discover competitors: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
		Message: "Competitors discovered successfully",
	})
}

// suggestCompetitors handles POST /api/v1/geo/competitors/suggest
// Uses LLM to automatically suggest competitors based on brand name/website
func (s *Server) suggestCompetitors(c *gin.Context) {
	var req models.SuggestCompetitorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	ctx := c.Request.Context()

	// Use the competitor service with LLM support
	competitorService := services.NewCompetitorServiceWithLLM(s.db, s.llmRegistry)
	result, err := competitorService.SuggestCompetitors(ctx, &req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to suggest competitors: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
		Message: "Competitors suggested successfully",
	})
}

// addCustomCompetitors handles POST /api/v1/geo/competitors/custom
// Allows users to add their own competitors to the list
func (s *Server) addCustomCompetitors(c *gin.Context) {
	var req models.AddCustomCompetitorsRequest
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

	competitorService := services.NewCompetitorService(s.db)
	result, err := competitorService.AddCustomCompetitors(ctx, req.Brand, req.Competitors)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to add custom competitors: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
		Message: "Custom competitors added successfully",
	})
}

// getCompetitorInsights handles POST /api/v1/geo/competitors/insights
// Returns comprehensive competitor insights and analysis
func (s *Server) getCompetitorInsights(c *gin.Context) {
	var req models.CompetitorInsightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	ctx := c.Request.Context()

	competitorService := services.NewCompetitorService(s.db)
	result, err := competitorService.GetCompetitorInsights(ctx, &req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get competitor insights: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
		Message: "Competitor insights retrieved successfully",
	})
}


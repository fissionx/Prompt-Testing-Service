package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AI2HU/gego/internal/models"
	"github.com/AI2HU/gego/internal/services"
)

// generatePrompts handles POST /api/v1/geo/prompts/generate
func (s *Server) generatePrompts(c *gin.Context) {
	var req models.GeneratePromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	if req.Count == 0 {
		req.Count = 20
	}

	// Create prompt generation service
	promptGenService := services.NewPromptGenerationService(s.db, s.llmRegistry)

	// Generate prompts
	prompts, existingCount, generatedCount, err := promptGenService.GeneratePromptsForBrand(
		c.Request.Context(),
		req.Brand,
		req.Category,
		req.Domain,
		req.Description,
		req.Count,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to generate prompts: "+err.Error())
		return
	}

	// Build response
	var promptPreviews []models.PromptPreview
	for _, prompt := range prompts {
		promptPreviews = append(promptPreviews, models.PromptPreview{
			ID:       prompt.ID,
			Template: prompt.Template,
			Category: prompt.Category,
			Reused:   !prompt.Generated || prompt.Brand != req.Brand,
		})
	}

	response := models.GeneratePromptsResponse{
		Brand:     req.Brand,
		Category:  req.Category,
		Domain:    req.Domain,
		Prompts:   promptPreviews,
		Existing:  existingCount,
		Generated: generatedCount,
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "Prompts generated successfully",
	})
}

// bulkExecute handles POST /api/v1/geo/execute/bulk
func (s *Server) bulkExecute(c *gin.Context) {
	var req models.BulkExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	// Validate temperature
	if req.Temperature < 0 || req.Temperature > 2 {
		s.errorResponse(c, http.StatusBadRequest, "Temperature must be between 0 and 2")
		return
	}

	// Create bulk execution service
	bulkService := services.NewBulkExecutionService(s.db, s.llmRegistry)

	// Start campaign execution
	campaign, err := bulkService.ExecuteCampaign(
		c.Request.Context(),
		req.CampaignName,
		req.Brand,
		req.PromptIDs,
		req.LLMIDs,
		req.Temperature,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to start campaign: "+err.Error())
		return
	}

	response := models.BulkExecuteResponse{
		CampaignID:   campaign.ID,
		CampaignName: campaign.Name,
		Brand:        campaign.Brand,
		TotalRuns:    campaign.TotalRuns,
		Status:       campaign.Status,
		StartedAt:    campaign.CreatedAt,
		Message:      "Campaign started successfully. Execution running in background.",
	}

	c.JSON(http.StatusAccepted, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "Campaign execution started",
	})
}

// getGEOInsights handles POST /api/v1/geo/insights
func (s *Server) getGEOInsights(c *gin.Context) {
	var req models.GEOInsightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	// Create analytics service
	analyticsService := services.NewGEOAnalyticsService(s.db)

	// Get insights
	insights, err := analyticsService.GetGEOInsights(
		c.Request.Context(),
		req.Brand,
		req.StartTime,
		req.EndTime,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get insights: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    insights,
		Message: "GEO insights retrieved successfully",
	})
}

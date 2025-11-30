package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
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

	// Generate prompts with optional website scraping
	prompts, existingCount, generatedCount, err := promptGenService.GeneratePromptsForBrand(
		c.Request.Context(),
		req.Brand,
		req.Website,
		req.Category,
		req.Domain,
		req.Description,
		req.Count,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to generate prompts: "+err.Error())
		return
	}

	// Build response with grouping by type
	var promptPreviews []models.PromptPreview
	promptsByType := make(map[string][]models.PromptPreview)
	typeCounts := make(map[string]int)

	for _, prompt := range prompts {
		preview := models.PromptPreview{
			ID:         prompt.ID,
			Template:   prompt.Template,
			PromptType: prompt.PromptType,
			Category:   prompt.Category,
			Reused:     !prompt.Generated || prompt.Brand != req.Brand,
		}
		promptPreviews = append(promptPreviews, preview)

		// Group by type
		typeKey := string(prompt.PromptType)
		if typeKey == "" {
			typeKey = "unknown"
		}
		promptsByType[typeKey] = append(promptsByType[typeKey], preview)
		typeCounts[typeKey]++
	}

	response := models.GeneratePromptsResponse{
		Brand:         req.Brand,
		Category:      req.Category,
		Domain:        req.Domain,
		Prompts:       promptPreviews,
		PromptsByType: promptsByType,
		Existing:      existingCount,
		Generated:     generatedCount,
		TypeCounts:    typeCounts,
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

// listPromptLibraries handles GET /api/v1/geo/libraries
func (s *Server) listPromptLibraries(c *gin.Context) {
	libraries, err := s.db.ListPromptLibraries(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list libraries: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    libraries,
		Message: "Prompt libraries retrieved successfully",
	})
}

// listBrandProfiles handles GET /api/v1/geo/profiles
func (s *Server) listBrandProfiles(c *gin.Context) {
	profiles, err := s.db.ListBrandProfiles(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list profiles: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    profiles,
		Message: "Brand profiles retrieved successfully",
	})
}

// getBrandProfile handles GET /api/v1/geo/profiles/:brand
func (s *Server) getBrandProfile(c *gin.Context) {
	brandName := c.Param("brand")
	if brandName == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand name is required")
		return
	}

	profile, err := s.db.GetBrandProfile(c.Request.Context(), brandName)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get profile: "+err.Error())
		return
	}

	if profile == nil {
		s.errorResponse(c, http.StatusNotFound, "Brand profile not found")
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    profile,
		Message: "Brand profile retrieved successfully",
	})
}

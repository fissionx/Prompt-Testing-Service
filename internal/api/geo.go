package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

	// Get LLM config if LLMID is provided
	var llmConfig *models.LLMConfig
	if req.LLMID != "" {
		config, err := s.llmService.GetLLM(c.Request.Context(), req.LLMID)
		if err != nil {
			s.errorResponse(c, http.StatusNotFound, "LLM not found: "+err.Error())
			return
		}
		if !config.Enabled {
			s.errorResponse(c, http.StatusBadRequest, "LLM is disabled")
			return
		}
		llmConfig = config
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
		llmConfig,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to generate prompts: "+err.Error())
		return
	}

	// Build response with grouping by type
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

	// Validate that at least one prompt ID is provided
	if len(req.PromptIDs) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one prompt ID must be provided")
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

	ctx := c.Request.Context()

	// Set default total runs if not specified
	totalRuns := req.TotalRuns
	if totalRuns == 0 {
		totalRuns = 1
	}

	// If scheduleCron is provided, create a scheduled campaign
	if req.ScheduleCron != "" {
		scheduledCampaign, err := s.scheduledCampaignManager.CreateScheduledCampaign(
			ctx,
			req.CampaignName,
			req.Brand,
			req.PromptIDs,
			req.LLMIDs,
			req.Temperature,
			req.ScheduleCron,
			totalRuns,
		)
		if err != nil {
			s.errorResponse(c, http.StatusInternalServerError, "Failed to create scheduled campaign: "+err.Error())
			return
		}

		response := models.BulkExecuteResponse{
			CampaignID:   scheduledCampaign.ID,
			CampaignName: scheduledCampaign.CampaignName,
			Brand:        scheduledCampaign.Brand,
			TotalRuns:    scheduledCampaign.TotalRuns,
			Status:       scheduledCampaign.Status,
			StartedAt:    scheduledCampaign.CreatedAt,
			NextRunAt:    scheduledCampaign.NextRunAt,
			ScheduleCron: scheduledCampaign.ScheduleCron,
			Message:      "Scheduled campaign created successfully. First execution started in background. Next run at " + scheduledCampaign.NextRunAt.Format("2006-01-02 15:04:05 UTC"),
		}

		c.JSON(http.StatusAccepted, models.APIResponse{
			Success: true,
			Data:    response,
			Message: "Scheduled campaign created and first execution started",
		})
		return
	}

	// Create bulk execution service for one-time execution
	bulkService := services.NewBulkExecutionService(s.db, s.llmRegistry)

	// Start campaign execution
	campaign, err := bulkService.ExecuteCampaign(
		ctx,
		req.CampaignName,
		req.Brand,
		req.PromptIDs,
		req.LLMIDs,
		req.Temperature,
		totalRuns,
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

	ctx := c.Request.Context()

	// Try to get cached data first (unless force refresh is requested)
	var cachedData *models.CachedGEOInsights
	if !req.ForceRefresh {
		query := models.AnalyticsCacheQuery{
			CampaignID: req.CampaignID,
			Brand:      req.Brand,
			StartTime:  req.StartTime,
			EndTime:    req.EndTime,
		}

		var err error
		cachedData, err = s.db.GetCachedGEOInsights(ctx, query)
		if err == nil && cachedData != nil {
			// Return cached data
			response := &models.GEOInsightsResponse{
				Brand:                 cachedData.Brand,
				LogoURL:               cachedData.LogoURL,
				FallbackLogoURL:       cachedData.FallbackLogoURL,
				AverageVisibility:     cachedData.AverageVisibility,
				MentionRate:           cachedData.MentionRate,
				GroundingRate:         cachedData.GroundingRate,
				SentimentBreakdown:    cachedData.SentimentBreakdown,
				TopCompetitors:        cachedData.TopCompetitors,
				PerformanceByLLM:      cachedData.PerformanceByLLM,
				PerformanceByCategory: cachedData.PerformanceByCategory,
				Trends:                cachedData.Trends,
				TotalResponses:        cachedData.TotalResponses,
			}

			c.JSON(http.StatusOK, models.APIResponse{
				Success: true,
				Data:    response,
				Message: "GEO insights retrieved from cache",
			})
			return
		}
	}

	// Create analytics service
	analyticsService := services.NewGEOAnalyticsService(s.db)

	// Compute insights if not cached
	insights, err := analyticsService.GetGEOInsights(
		ctx,
		req.Brand,
		req.StartTime,
		req.EndTime,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get insights: "+err.Error())
		return
	}

	// Cache the computed data for future requests
	go func() {
		startTime := time.Now().Add(-30 * 24 * time.Hour) // Default to last 30 days
		endTime := time.Now()
		if req.StartTime != nil {
			startTime = *req.StartTime
		}
		if req.EndTime != nil {
			endTime = *req.EndTime
		}

		cachedInsights := &models.CachedGEOInsights{
			ID:                    uuid.New().String(),
			CampaignID:            req.CampaignID,
			Brand:                 req.Brand,
			StartTime:             startTime,
			EndTime:               endTime,
			LogoURL:               insights.LogoURL,
			FallbackLogoURL:       insights.FallbackLogoURL,
			AverageVisibility:     insights.AverageVisibility,
			MentionRate:           insights.MentionRate,
			GroundingRate:         insights.GroundingRate,
			SentimentBreakdown:    insights.SentimentBreakdown,
			TopCompetitors:        insights.TopCompetitors,
			PerformanceByLLM:      insights.PerformanceByLLM,
			PerformanceByCategory: insights.PerformanceByCategory,
			Trends:                insights.Trends,
			TotalResponses:        insights.TotalResponses,
		}
		s.db.SaveCachedGEOInsights(context.Background(), cachedInsights)
	}()

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

// listScheduledCampaigns handles GET /api/v1/geo/campaigns
// Returns all scheduled campaigns with full prompt and LLM details
func (s *Server) listScheduledCampaigns(c *gin.Context) {
	brand := c.Query("brand")
	status := c.Query("status")

	ctx := c.Request.Context()

	// Get all scheduled campaigns
	campaigns, err := s.db.ListScheduledCampaigns(ctx, status)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list campaigns: "+err.Error())
		return
	}

	// Filter by brand if specified
	var filteredCampaigns []*models.ScheduledCampaign
	for _, campaign := range campaigns {
		if brand == "" || campaign.Brand == brand {
			filteredCampaigns = append(filteredCampaigns, campaign)
		}
	}

	// Build detailed response with prompt and LLM details
	campaignDetails := make([]models.ScheduledCampaignDetail, 0, len(filteredCampaigns))
	activeCount, pausedCount, completedCount := 0, 0, 0

	for _, campaign := range filteredCampaigns {
		// Get prompt details
		prompts := make([]models.PromptDetail, 0, len(campaign.PromptIDs))
		for _, promptID := range campaign.PromptIDs {
			prompt, err := s.db.GetPrompt(ctx, promptID)
			if err == nil && prompt != nil {
				prompts = append(prompts, models.PromptDetail{
					ID:         prompt.ID,
					Template:   prompt.Template,
					PromptType: prompt.PromptType,
					Category:   prompt.Category,
				})
			}
		}

		// Get LLM details
		llms := make([]models.LLMDetail, 0, len(campaign.LLMIDs))
		for _, llmID := range campaign.LLMIDs {
			llm, err := s.db.GetLLM(ctx, llmID)
			if err == nil && llm != nil {
				llms = append(llms, models.LLMDetail{
					ID:       llm.ID,
					Name:     llm.Name,
					Provider: llm.Provider,
					Model:    llm.Model,
				})
			}
		}

		// Generate human-readable schedule description
		scheduleDesc := cronToDescription(campaign.ScheduleCron)

		detail := models.ScheduledCampaignDetail{
			ID:           campaign.ID,
			CampaignName: campaign.CampaignName,
			Brand:        campaign.Brand,
			Prompts:      prompts,
			LLMs:         llms,
			Temperature:  campaign.Temperature,
			ScheduleCron: campaign.ScheduleCron,
			ScheduleDesc: scheduleDesc,
			Status:       campaign.Status,
			TotalRuns:    campaign.TotalRuns,
			RunCount:     campaign.RunCount,
			LastRunAt:    campaign.LastRunAt,
			NextRunAt:    campaign.NextRunAt,
			CreatedAt:    campaign.CreatedAt,
			UpdatedAt:    campaign.UpdatedAt,
		}

		campaignDetails = append(campaignDetails, detail)

		// Count by status
		switch campaign.Status {
		case "active":
			activeCount++
		case "paused":
			pausedCount++
		case "completed":
			completedCount++
		}
	}

	response := models.ListScheduledCampaignsResponse{
		Brand:     brand,
		Campaigns: campaignDetails,
		Total:     len(campaignDetails),
		Active:    activeCount,
		Paused:    pausedCount,
		Completed: completedCount,
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "Scheduled campaigns retrieved successfully",
	})
}

// saveCustomPrompts handles POST /api/v1/geo/prompts/save
// Stores custom prompts list along with promptIds from suggested prompts
func (s *Server) saveCustomPrompts(c *gin.Context) {
	var req models.SaveCustomPromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	// Validate that at least promptIds or customPrompts are provided
	if len(req.PromptIDs) == 0 && len(req.CustomPrompts) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one prompt ID or custom prompt must be provided")
		return
	}

	ctx := c.Request.Context()
	savedPromptIDs := make([]string, 0, len(req.PromptIDs)+len(req.CustomPrompts))
	createdCount := 0
	existingCount := 0

	// First, disable all existing prompts for this brand (they will be replaced by the saved ones)
	// This ensures only prompts saved via this endpoint are active
	allBrandPrompts, _ := s.db.ListPrompts(ctx, nil)
	for _, prompt := range allBrandPrompts {
		if prompt.Brand != "" && strings.EqualFold(prompt.Brand, req.Brand) {
			prompt.Enabled = false
			_ = s.db.UpdatePrompt(ctx, prompt)
		}
	}

	// Add existing prompt IDs (validate they exist and mark them as saved/finalized)
	for _, promptID := range req.PromptIDs {
		prompt, err := s.db.GetPrompt(ctx, promptID)
		if err != nil || prompt == nil {
			// Skip invalid prompt IDs but continue with others
			continue
		}

		// Mark prompt as saved/finalized: ensure it's enabled and has the brand set
		prompt.Enabled = true
		prompt.Brand = req.Brand
		prompt.UpdatedAt = time.Now()
		if err := s.db.UpdatePrompt(ctx, prompt); err != nil {
			// Log error but continue
			continue
		}

		savedPromptIDs = append(savedPromptIDs, promptID)
		existingCount++
	}

	// Create new custom prompts
	if len(req.CustomPrompts) > 0 {
		for _, customPrompt := range req.CustomPrompts {
			// Create prompt in database
			promptType := models.PromptType(customPrompt.PromptType)
			if promptType == "" {
				promptType = models.PromptTypeCustom
			}

			prompt := &models.Prompt{
				ID:         uuid.New().String(),
				Template:   customPrompt.Template,
				PromptType: promptType,
				Category:   customPrompt.Category,
				Tags:       customPrompt.Tags,
				Brand:      req.Brand,
				Generated:  false,
				Enabled:    true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			// Save prompt to database
			if err := s.db.CreatePrompt(ctx, prompt); err != nil {
				s.errorResponse(c, http.StatusInternalServerError, "Failed to save custom prompt: "+err.Error())
				return
			}

			savedPromptIDs = append(savedPromptIDs, prompt.ID)
			createdCount++
		}
	}

	response := models.SaveCustomPromptsResponse{
		Brand:          req.Brand,
		SavedPromptIDs: savedPromptIDs,
		CreatedCount:   createdCount,
		ExistingCount:  existingCount,
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "Custom prompts saved successfully",
	})
}

// deletePromptsByIDs handles DELETE /api/v1/geo/prompts
// Deletes one or more prompts by IDs from the active collection
func (s *Server) deletePromptsByIDs(c *gin.Context) {
	var req models.DeletePromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if len(req.PromptIDs) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one prompt ID is required")
		return
	}

	ctx := c.Request.Context()

	deletedCount := 0
	failedCount := 0
	deletedIDs := make([]string, 0)
	failedIDs := make([]string, 0)

	for _, id := range req.PromptIDs {
		if err := s.db.DeletePrompt(ctx, id); err != nil {
			failedCount++
			failedIDs = append(failedIDs, id)
			continue
		}
		deletedCount++
		deletedIDs = append(deletedIDs, id)
	}

	response := models.DeletePromptsResponse{
		DeletedCount: deletedCount,
		FailedCount:  failedCount,
	}

	// Only include IDs if there are results
	if len(deletedIDs) > 0 {
		response.DeletedIDs = deletedIDs
	}
	if len(failedIDs) > 0 {
		response.FailedIDs = failedIDs
	}

	// If all deletions failed, return error status
	if deletedCount == 0 {
		s.errorResponse(c, http.StatusNotFound, "No prompts were deleted. All prompt IDs were invalid or not found")
		return
	}

	// If some deletions failed, return partial success with warning
	if failedCount > 0 {
		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Data:    response,
			Message: fmt.Sprintf("Partially successful: %d deleted, %d failed", deletedCount, failedCount),
		})
		return
	}

	// All deletions succeeded
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: fmt.Sprintf("Successfully deleted %d prompt(s)", deletedCount),
	})
}

// cronToDescription converts a cron expression to human-readable text
func cronToDescription(cronExpr string) string {
	// Common cron patterns
	cronDescriptions := map[string]string{
		"0 * * * *":    "Every hour",
		"0 */2 * * *":  "Every 2 hours",
		"0 */3 * * *":  "Every 3 hours",
		"0 */4 * * *":  "Every 4 hours",
		"0 */6 * * *":  "Every 6 hours",
		"0 */8 * * *":  "Every 8 hours",
		"0 */12 * * *": "Every 12 hours",
		"0 0 * * *":    "Daily at midnight",
		"0 9 * * *":    "Daily at 9 AM",
		"0 0 * * 0":    "Weekly on Sunday",
		"0 0 * * 1":    "Weekly on Monday",
		"0 0 1 * *":    "Monthly on the 1st",
		"0 0 15 * *":   "Monthly on the 15th",
		"*/5 * * * *":  "Every 5 minutes",
		"*/10 * * * *": "Every 10 minutes",
		"*/15 * * * *": "Every 15 minutes",
		"*/30 * * * *": "Every 30 minutes",
	}

	if desc, ok := cronDescriptions[cronExpr]; ok {
		return desc
	}

	return "Custom schedule: " + cronExpr
}

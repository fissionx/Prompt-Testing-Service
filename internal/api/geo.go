package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
	"github.com/fissionx/gego/internal/shared"
)

// generatePrompts handles POST /api/v1/geo/prompts/generate
func (s *Server) generatePrompts(c *gin.Context) {
	var req models.GeneratePromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.BrandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	if req.Count == 0 {
		req.Count = 20
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	// Normalize domain to www. format
	normalizedDomain := shared.NormalizeDomainToURL(brandInfo.Domain)
	brandName := brandInfo.Name

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

	// Use brand info domain if not provided in request, otherwise use normalized domain
	domain := normalizedDomain
	if req.Domain != "" {
		domain = shared.NormalizeDomainToURL(req.Domain)
	}

	// Generate prompts with optional website scraping
	prompts, existingCount, generatedCount, err := promptGenService.GeneratePromptsForBrand(
		c.Request.Context(),
		req.BrandID,
		brandInfo.OrgID,
		brandName,
		req.Website,
		req.Category,
		domain,
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
			Reused:     !prompt.Generated || prompt.Brand != brandName,
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
		Brand:         brandName,
		Category:      req.Category,
		Domain:        domain,
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
// Executes selected prompts on selected LLM configurations
func (s *Server) bulkExecute(c *gin.Context) {
	var req models.BulkExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate request
	if err := s.validateBulkExecuteRequest(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get brand information
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	// Apply defaults
	s.applyBulkExecuteDefaults(&req)

	// Invalidate analytics cache for this brand (async)
	s.invalidateBrandAnalyticsCache(brandInfo.Name)

	ctx := c.Request.Context()

	// Execute campaign (scheduled or one-time)
	response, err := s.executeBulkCampaign(ctx, &req, brandInfo)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusAccepted, models.APIResponse{
		Success: true,
		Data:    response,
		Message: response.Message,
	})
}

// validateBulkExecuteRequest validates the bulk execute request
func (s *Server) validateBulkExecuteRequest(req *models.BulkExecuteRequest) error {
	if req.BrandID == "" {
		return fmt.Errorf("brandId is required")
	}
	if len(req.PromptIDs) == 0 {
		return fmt.Errorf("at least one prompt ID must be provided")
	}
	if len(req.LLMIDs) == 0 {
		return fmt.Errorf("at least one LLM ID must be provided")
	}
	if req.Temperature < 0 || req.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	return nil
}

// applyBulkExecuteDefaults applies default values to the request
func (s *Server) applyBulkExecuteDefaults(req *models.BulkExecuteRequest) {
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.TotalRuns == 0 {
		req.TotalRuns = 1
	}
}

// invalidateBrandAnalyticsCache invalidates all analytics cache for a brand
func (s *Server) invalidateBrandAnalyticsCache(brandName string) {
	go func() {
		bgCtx := context.Background()
		_ = s.db.DeleteCachedGEOInsightsByBrand(bgCtx, brandName)
		_ = s.db.DeleteCachedSourceAnalyticsByBrand(bgCtx, brandName)
		_ = s.db.DeleteCachedCompetitiveBenchmarkByBrand(bgCtx, brandName)
		_ = s.db.DeleteCachedPromptPerformanceByBrand(bgCtx, brandName)
		_ = s.db.DeleteCachedPromptTimeSeriesByBrand(bgCtx, brandName)
	}()
}

// executeBulkCampaign executes a bulk campaign (scheduled or one-time)
func (s *Server) executeBulkCampaign(ctx context.Context, req *models.BulkExecuteRequest, brandInfo *services.BrandInfo) (*models.BulkExecuteResponse, error) {
	// Handle scheduled campaign
	if req.ScheduleCron != "" {
		return s.createScheduledCampaign(ctx, req, brandInfo)
	}

	// Handle one-time execution
	return s.createOneTimeCampaign(ctx, req, brandInfo)
}

// createScheduledCampaign creates and starts a scheduled campaign
func (s *Server) createScheduledCampaign(ctx context.Context, req *models.BulkExecuteRequest, brandInfo *services.BrandInfo) (*models.BulkExecuteResponse, error) {
	scheduledCampaign, err := s.scheduledCampaignManager.CreateScheduledCampaign(
		ctx,
		req.CampaignName,
		brandInfo.Name,
		req.PromptIDs,
		req.LLMIDs,
		req.Temperature,
		req.ScheduleCron,
		req.TotalRuns,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduled campaign: %w", err)
	}

	message := "Scheduled campaign created successfully. First execution started in background."
	if scheduledCampaign.NextRunAt != nil {
		message += " Next run at " + scheduledCampaign.NextRunAt.Format("2006-01-02 15:04:05 UTC")
	}

	return &models.BulkExecuteResponse{
		CampaignID:   scheduledCampaign.ID,
		CampaignName: scheduledCampaign.CampaignName,
		Brand:        scheduledCampaign.Brand,
		TotalRuns:    scheduledCampaign.TotalRuns,
		Status:       scheduledCampaign.Status,
		StartedAt:    scheduledCampaign.CreatedAt,
		NextRunAt:    scheduledCampaign.NextRunAt,
		ScheduleCron: scheduledCampaign.ScheduleCron,
		Message:      message,
	}, nil
}

// createOneTimeCampaign creates and starts a one-time campaign
func (s *Server) createOneTimeCampaign(ctx context.Context, req *models.BulkExecuteRequest, brandInfo *services.BrandInfo) (*models.BulkExecuteResponse, error) {
	bulkService := services.NewBulkExecutionService(s.db, s.llmRegistry)

	campaign, err := bulkService.ExecuteCampaign(
		ctx,
		req.CampaignName,
		req.BrandID,
		brandInfo.OrgID,
		brandInfo.Name,
		req.PromptIDs,
		req.LLMIDs,
		req.Temperature,
		req.TotalRuns,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start campaign: %w", err)
	}

	return &models.BulkExecuteResponse{
		CampaignID:   campaign.ID,
		CampaignName: campaign.Name,
		Brand:        campaign.Brand,
		TotalRuns:    campaign.TotalRuns,
		Status:       campaign.Status,
		StartedAt:    campaign.CreatedAt,
		Message:      "Campaign started successfully. Execution running in background.",
	}, nil
}

// getGEOInsights handles POST /api/v1/geo/insights
func (s *Server) getGEOInsights(c *gin.Context) {
	var req models.GEOInsightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.BrandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	ctx := c.Request.Context()

	// Try to get cached data first (unless force refresh is requested)
	var cachedData *models.CachedGEOInsights
	if !req.ForceRefresh {
		query := models.AnalyticsCacheQuery{
			CampaignID: req.CampaignID,
			Brand:      brandName,
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
		req.BrandID,
		brandInfo.OrgID,
		brandName,
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
			Brand:                 brandName,
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

// saveCustomPrompts handles POST /api/v1/geo/brand/:brandId/prompts/save
// Accepts both promptIds from suggested prompts and custom prompts
// Moves prompts from suggested list to active list
// Returns the same response format as GET /api/v1/geo/brand/:brandId/prompts
func (s *Server) saveCustomPrompts(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	var req models.SaveCustomPromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name
	normalizedDomain := shared.NormalizeDomainToURL(brandInfo.Domain)

	// Use domain from brand info as website (convert to https URL)
	website := ""
	if normalizedDomain != "" {
		website = "https://" + normalizedDomain
	}

	// Use category from brand info
	category := brandInfo.Category
	domain := normalizedDomain

	// Validate that at least promptIds or customPrompts are provided
	if len(req.PromptIDs) == 0 && len(req.CustomPrompts) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one prompt ID or custom prompt must be provided")
		return
	}

	ctx := c.Request.Context()

	// Use the brand prompt service to save prompts
	// This will move prompts from suggested to active
	source := "custom"
	if len(req.PromptIDs) > 0 && len(req.CustomPrompts) > 0 {
		source = "mixed"
	} else if len(req.PromptIDs) > 0 {
		source = "suggested"
	}

	_, err = s.brandPromptService.SavePrompts(
		ctx,
		brandName,
		brandID,
		brandInfo.OrgID,
		req.PromptIDs,
		req.CustomPrompts,
		source,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to save prompts: "+err.Error())
		return
	}

	// Return the updated prompts list in the same format as GET endpoint
	s.getBrandPromptsResponse(c, brandID, brandInfo.OrgID, brandName, website, category, domain, "", 20, false)
}

// deletePromptsByIDs handles DELETE /api/v1/geo/prompts
// deletePromptsByIDs handles DELETE /api/v1/geo/brand/:brandId/prompts
// Deletes one or more prompts by IDs from the active list and moves them back to suggested list
// Supports:
//   - Query parameters: ?id=Y (single prompt)
//   - JSON body: { "promptIds": ["id1", "id2"] } (multiple prompts)
//   - No query params or body (delete all prompts for brand)
//
// Returns the same response format as GET /api/v1/geo/brand/:brandId/prompts
func (s *Server) deletePromptsByIDs(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId parameter is required")
		return
	}

	promptID := c.Query("id") // Individual prompt ID to delete

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name
	normalizedDomain := shared.NormalizeDomainToURL(brandInfo.Domain)

	// Use domain from brand info as website (convert to https URL)
	website := ""
	if normalizedDomain != "" {
		website = "https://" + normalizedDomain
	}

	// Use category from brand info
	category := brandInfo.Category
	domain := normalizedDomain

	ctx := c.Request.Context()

	// If brandId is provided without prompt ID, check if we should delete all
	if promptID == "" {
		// Try to parse JSON body (optional - won't fail if empty)
		var req models.DeletePromptsRequest
		if err := c.ShouldBindJSON(&req); err == nil && len(req.PromptIDs) > 0 {
			// Multiple prompt IDs from body
			deletedCount := 0
			for _, id := range req.PromptIDs {
				if err := s.brandPromptService.DeletePrompt(ctx, brandID, id); err != nil {
					// Continue with other IDs even if one fails
					continue
				}
				deletedCount++
			}

			if deletedCount == 0 {
				s.errorResponse(c, http.StatusNotFound, "No prompts were deleted. All prompt IDs were invalid or not found")
				return
			}

			// Return the updated prompts list in the same format as GET endpoint
			s.getBrandPromptsResponse(c, brandID, brandInfo.OrgID, brandName, website, category, domain, "", 20, false)
			return
		}

		// No body or empty body - delete all prompts for brand
		if err := s.brandPromptService.DeleteAllPrompts(ctx, brandID); err != nil {
			s.errorResponse(c, http.StatusInternalServerError, "Failed to delete prompts: "+err.Error())
			return
		}

		// Return the updated prompts list in the same format as GET endpoint
		s.getBrandPromptsResponse(c, brandID, brandInfo.OrgID, brandName, website, category, domain, "", 20, false)
		return
	}

	// Delete single prompt by ID from query parameter
	if err := s.brandPromptService.DeletePrompt(ctx, brandID, promptID); err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to delete prompt: "+err.Error())
		return
	}

	// Return the updated prompts list in the same format as GET endpoint
	s.getBrandPromptsResponse(c, brandID, brandInfo.OrgID, brandName, website, category, domain, "", 20, false)
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

package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
	"github.com/fissionx/gego/internal/shared"
)

// getSourceAnalytics handles GET /api/v1/geo/brand/:brandId/analytics/sources
// Path parameters:
//   - brandId (required): Brand UUID from external API
//
// Query parameters:
//   - startTime (optional): Start time in ISO 8601 format (e.g., "2024-01-01T00:00:00Z")
//   - endTime (optional): End time in ISO 8601 format (e.g., "2024-01-31T23:59:59Z")
//   - topN (optional): Number of top sources to return (default: 20)
//   - forceRefresh (optional): Force re-computation even if cached (default: false)
func (s *Server) getSourceAnalytics(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	var req models.SourceAnalyticsRequest
	req.BrandID = brandID

	// Parse time parameters if provided
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &t
		}
	}

	// Parse TopN from query parameter
	if topNStr := c.Query("topN"); topNStr != "" {
		if parsed, err := strconv.Atoi(topNStr); err == nil && parsed > 0 {
			req.TopN = parsed
		}
	}

	// Parse ForceRefresh from query parameter
	if forceRefreshStr := c.Query("forceRefresh"); forceRefreshStr != "" {
		req.ForceRefresh = forceRefreshStr == "true"
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	if req.TopN == 0 {
		req.TopN = 20
	}

	ctx := c.Request.Context()

	// Try to get cached data first (unless force refresh is requested)
	var cachedData *models.CachedSourceAnalytics
	if !req.ForceRefresh {
		query := models.AnalyticsCacheQuery{
			Brand:     brandName,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		}

		var err error
		cachedData, err = s.db.GetCachedSourceAnalytics(ctx, query)
		if err == nil && cachedData != nil {
			// Normalize domains in cached data to ensure www. prefix
			normalizedSources := make([]models.SourceInsight, len(cachedData.TopSources))
			for i, source := range cachedData.TopSources {
				normalizedSources[i] = source
				normalizedSources[i].Domain = shared.NormalizeDomainToURL(source.Domain)
			}

			// Return cached data with normalized domains
			response := &models.SourceAnalyticsResponse{
				Brand:           cachedData.Brand,
				LogoURL:         cachedData.LogoURL,
				FallbackLogoURL: cachedData.FallbackLogoURL,
				Period:          cachedData.Period,
				TopSources:      normalizedSources,
				Recommendations: cachedData.Recommendations,
				TotalSources:    cachedData.TotalSources,
				TotalCitations:  cachedData.TotalCitations,
			}

			c.JSON(http.StatusOK, models.APIResponse{
				Success: true,
				Data:    response,
				Message: "Source analytics retrieved from cache",
			})
			return
		}
	}

	// Compute analytics if not cached
	analytics, err := s.sourceAnalyticsService.GetSourceAnalytics(
		ctx,
		req.BrandID,
		brandInfo.OrgID,
		brandName,
		req.StartTime,
		req.EndTime,
		req.TopN,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get source analytics: "+err.Error())
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

		cachedAnalytics := &models.CachedSourceAnalytics{
			ID:              uuid.New().String(),
			Brand:           brandName,
			StartTime:       startTime,
			EndTime:         endTime,
			LogoURL:         analytics.LogoURL,
			FallbackLogoURL: analytics.FallbackLogoURL,
			Period:          analytics.Period,
			TopSources:      analytics.TopSources,
			Recommendations: analytics.Recommendations,
			TotalSources:    analytics.TotalSources,
			TotalCitations:  analytics.TotalCitations,
		}
		s.db.SaveCachedSourceAnalytics(context.Background(), cachedAnalytics)
	}()

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    analytics,
		Message: "Source analytics retrieved successfully",
	})
}

// getCompetitiveBenchmark handles GET /api/v1/geo/brand/:brandId/analytics/competitive
func (s *Server) getCompetitiveBenchmark(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	var req models.CompetitiveBenchmarkRequest
	req.BrandID = brandID

	// Parse time parameters if provided
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &t
		}
	}

	// Parse array parameters (promptIds, llmIds, competitors)
	if promptIDsStr := c.Query("promptIds"); promptIDsStr != "" {
		req.PromptIDs = parseStringArray(promptIDsStr)
	}
	if llmIDsStr := c.Query("llmIds"); llmIDsStr != "" {
		req.LLMIDs = parseStringArray(llmIDsStr)
	}
	if competitorsStr := c.Query("competitors"); competitorsStr != "" {
		req.Competitors = parseCompetitorsFromQuery(competitorsStr)
	}

	// Parse forceRefresh
	if forceRefreshStr := c.Query("forceRefresh"); forceRefreshStr != "" {
		req.ForceRefresh = forceRefreshStr == "true"
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	mainBrandName := brandInfo.Name
	mainBrandDomain := shared.NormalizeDomainToURL(brandInfo.Domain)

	ctx := c.Request.Context()

	// Try to get cached data first (only if no specific competitors are requested and not forcing refresh)
	if len(req.Competitors) == 0 && !req.ForceRefresh {
		query := models.AnalyticsCacheQuery{
			Brand:     mainBrandName,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		}

		cachedData, err := s.db.GetCachedCompetitiveBenchmark(ctx, query)
		if err == nil && cachedData != nil {
			// Ensure domains are populated in cached data
			// Main brand domain
			mainBrandPerf := cachedData.MainBrandPerf
			if mainBrandPerf.Domain == "" {
				mainBrandPerf.Domain = deriveDomainFromName(mainBrandPerf.Brand)
			}

			// Competitor domains
			competitors := make([]models.BrandPerformance, len(cachedData.Competitors))
			for i, comp := range cachedData.Competitors {
				competitors[i] = comp
				if competitors[i].Domain == "" {
					competitors[i].Domain = deriveDomainFromName(comp.Brand)
				}
			}

			// Return cached data with domains
			response := &models.CompetitiveBenchmarkResponse{
				MainBrand:       mainBrandPerf,
				Competitors:     competitors,
				OtherBrands:     cachedData.OtherBrands,
				MarketLeader:    cachedData.MarketLeader,
				YourRank:        cachedData.YourRank,
				TotalBrands:     cachedData.TotalBrands,
				PromptBreakdown: cachedData.PromptBreakdown,
				AnalyzedAt:      cachedData.AnalyzedAt,
			}

			c.JSON(http.StatusOK, models.APIResponse{
				Success: true,
				Data:    response,
				Message: "Competitive benchmark retrieved from cache",
			})
			return
		}
	}

	// Convert Competitor objects to strings for internal processing
	competitorNames := make([]string, 0, len(req.Competitors))
	competitorMap := make(map[string]string) // name -> domain mapping
	for _, comp := range req.Competitors {
		competitorNames = append(competitorNames, comp.Name)
		if comp.Domain != "" {
			// Normalize competitor domains to www. format
			competitorMap[comp.Name] = shared.NormalizeDomainToURL(comp.Domain)
		}
	}

	// Ensure main brand domain is in the map
	if mainBrandDomain != "" {
		competitorMap[mainBrandName] = mainBrandDomain
	}

	// Compute benchmark if not cached or specific competitors requested
	benchmark, err := s.competitiveBenchmarkService.GetCompetitiveBenchmark(
		ctx,
		mainBrandName,
		competitorNames,
		req.PromptIDs,
		req.LLMIDs,
		req.StartTime,
		req.EndTime,
		req.Region,
		competitorMap,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get competitive benchmark: "+err.Error())
		return
	}

	// Ensure main brand domain is set in response
	if benchmark.MainBrand.Domain == "" {
		benchmark.MainBrand.Domain = mainBrandDomain
	}

	// Cache the computed data for future requests
	// Cache when: (1) no specific competitors requested (auto-detected), OR (2) forceRefresh is true (update cache)
	if len(req.Competitors) == 0 || req.ForceRefresh {
		go func() {
			// Use the same time ranges that were used for the query
			startTime := time.Now().Add(-30 * 24 * time.Hour)
			endTime := time.Now()
			if req.StartTime != nil {
				startTime = *req.StartTime
			}
			if req.EndTime != nil {
				endTime = *req.EndTime
			}

			// If forceRefresh is true, delete old cache entries for this brand first
			if req.ForceRefresh {
				_ = s.db.DeleteCachedCompetitiveBenchmarkByBrand(context.Background(), mainBrandName)
			}

			cachedBenchmark := &models.CachedCompetitiveBenchmark{
				ID:              uuid.New().String(),
				CampaignID:      "", // Empty for general cache (not campaign-specific)
				MainBrand:       mainBrandName,
				StartTime:       startTime,
				EndTime:         endTime,
				MainBrandPerf:   benchmark.MainBrand,
				Competitors:     benchmark.Competitors,
				OtherBrands:     benchmark.OtherBrands,
				MarketLeader:    benchmark.MarketLeader,
				YourRank:        benchmark.YourRank,
				TotalBrands:     benchmark.TotalBrands,
				PromptBreakdown: benchmark.PromptBreakdown,
				AnalyzedAt:      benchmark.AnalyzedAt,
			}
			s.db.SaveCachedCompetitiveBenchmark(context.Background(), cachedBenchmark)
		}()
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    benchmark,
		Message: "Competitive benchmark retrieved successfully",
	})
}

// getPositionAnalytics handles GET /api/v1/geo/brand/:brandId/analytics/position
func (s *Server) getPositionAnalytics(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	// Parse time parameters
	var startTime, endTime *time.Time
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	// Get position analytics
	analytics, err := getPositionAnalyticsForBrand(
		c.Request.Context(),
		s.db,
		brandID,
		brandInfo.OrgID,
		brandName,
		startTime,
		endTime,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get position analytics: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    analytics,
		Message: "Position analytics retrieved successfully",
	})
}

// getPositionAnalyticsForBrand computes position analytics for a brand
func getPositionAnalyticsForBrand(
	ctx context.Context,
	database db.Database,
	brandID string,
	orgID string,
	brand string,
	startTime, endTime *time.Time,
) (*models.PositionAnalyticsResponse, error) {
	// Get logo service
	logoService := services.NewLogoService(database)
	// Fetch responses
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := database.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter for brand
	var brandResponses []*models.Response
	for _, resp := range allResponses {
		if resp.Brand == brand {
			brandResponses = append(brandResponses, resp)
		}
	}

	if len(brandResponses) == 0 {
		return &models.PositionAnalyticsResponse{
			Brand:         brand,
			TotalMentions: 0,
		}, nil
	}

	// Calculate metrics
	totalPosition := 0.0
	positionCount := 0
	topPositionCount := 0
	positionBreakdown := make(map[string]int)
	byPromptType := make(map[string][]float64)
	byLLM := make(map[string][]float64)

	for _, resp := range brandResponses {
		if resp.BrandPosition > 0 {
			totalPosition += float64(resp.BrandPosition)
			positionCount++

			if resp.BrandPosition <= 3 {
				topPositionCount++
			}

			// Position breakdown
			posKey := fmt.Sprintf("position_%d", resp.BrandPosition)
			positionBreakdown[posKey]++

			// Get prompt to determine type
			prompt, err := database.GetPrompt(ctx, resp.PromptID)
			if err == nil && prompt != nil {
				promptType := string(prompt.PromptType)
				if promptType == "" {
					promptType = "unknown"
				}
				byPromptType[promptType] = append(byPromptType[promptType], float64(resp.BrandPosition))
			}

			// By LLM
			byLLM[resp.LLMName] = append(byLLM[resp.LLMName], float64(resp.BrandPosition))
		}
	}

	// Get brand logo
	brandLogo := logoService.GetBrandLogo(ctx, brandID, orgID, brand, "")

	response := &models.PositionAnalyticsResponse{
		Brand:             brand,
		LogoURL:           brandLogo.LogoURL,
		FallbackLogoURL:   brandLogo.FallbackLogoURL,
		TotalMentions:     len(brandResponses),
		PositionBreakdown: positionBreakdown,
		ByPromptType:      make(map[string]float64),
		ByLLM:             make(map[string]float64),
	}

	if positionCount > 0 {
		response.AveragePosition = totalPosition / float64(positionCount)
		response.TopPositionRate = float64(topPositionCount) / float64(positionCount) * 100

		// Calculate averages by prompt type
		for promptType, positions := range byPromptType {
			sum := 0.0
			for _, pos := range positions {
				sum += pos
			}
			response.ByPromptType[promptType] = sum / float64(len(positions))
		}

		// Calculate averages by LLM
		for llmName, positions := range byLLM {
			sum := 0.0
			for _, pos := range positions {
				sum += pos
			}
			response.ByLLM[llmName] = sum / float64(len(positions))
		}
	}

	return response, nil
}

// getPromptPerformance handles GET /api/v1/geo/brand/:brandId/analytics/prompt-performance
func (s *Server) getPromptPerformance(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	var req models.PromptPerformanceRequest
	req.BrandID = brandID

	// Parse time parameters if provided
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &t
		}
	}

	// Parse minResponses
	if minResponsesStr := c.Query("minResponses"); minResponsesStr != "" {
		if parsed, err := strconv.Atoi(minResponsesStr); err == nil && parsed > 0 {
			req.MinResponses = parsed
		}
	}

	// Parse forceRefresh
	if forceRefreshStr := c.Query("forceRefresh"); forceRefreshStr != "" {
		req.ForceRefresh = forceRefreshStr == "true"
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	if req.MinResponses == 0 {
		req.MinResponses = 3
	}

	ctx := c.Request.Context()

	// Try to get cached data first (unless force refresh is requested)
	var cachedData *models.CachedPromptPerformance
	if !req.ForceRefresh {
		query := models.AnalyticsCacheQuery{
			Brand:     brandName,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		}

		var err error
		cachedData, err = s.db.GetCachedPromptPerformance(ctx, query)
		if err == nil && cachedData != nil {
			// Return cached data
			response := &models.PromptPerformanceResponse{
				Brand:                cachedData.Brand,
				LogoURL:              cachedData.LogoURL,
				FallbackLogoURL:      cachedData.FallbackLogoURL,
				Period:               cachedData.Period,
				Prompts:              cachedData.Prompts,
				TopPerformers:        cachedData.TopPerformers,
				LowPerformers:        cachedData.LowPerformers,
				AvgEffectiveness:     cachedData.AvgEffectiveness,
				TotalPromptsAnalyzed: cachedData.TotalPromptsAnalyzed,
			}

			c.JSON(http.StatusOK, models.APIResponse{
				Success: true,
				Data:    response,
				Message: "Prompt performance retrieved from cache",
			})
			return
		}
	}

	// Compute prompt performance analytics if not cached
	performance, err := s.promptPerformanceService.GetPromptPerformance(
		ctx,
		req.BrandID,
		brandInfo.OrgID,
		brandName,
		req.StartTime,
		req.EndTime,
		req.MinResponses,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get prompt performance: "+err.Error())
		return
	}

	// Cache the computed data for future requests
	go func() {
		startTime := time.Now().Add(-30 * 24 * time.Hour)
		endTime := time.Now()
		if req.StartTime != nil {
			startTime = *req.StartTime
		}
		if req.EndTime != nil {
			endTime = *req.EndTime
		}

		cachedPerformance := &models.CachedPromptPerformance{
			ID:                   uuid.New().String(),
			Brand:                brandName,
			StartTime:            startTime,
			EndTime:              endTime,
			LogoURL:              performance.LogoURL,
			FallbackLogoURL:      performance.FallbackLogoURL,
			Period:               performance.Period,
			Prompts:              performance.Prompts,
			TopPerformers:        performance.TopPerformers,
			LowPerformers:        performance.LowPerformers,
			AvgEffectiveness:     performance.AvgEffectiveness,
			TotalPromptsAnalyzed: performance.TotalPromptsAnalyzed,
		}
		s.db.SaveCachedPromptPerformance(context.Background(), cachedPerformance)
	}()

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    performance,
		Message: "Prompt performance retrieved successfully",
	})
}

// getPromptTimeSeries handles GET /api/v1/geo/brand/:brandId/analytics/prompt-timeseries
func (s *Server) getPromptTimeSeries(c *gin.Context) {
	// Get brandId from path parameter
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	// Get promptId from query parameter
	promptID := c.Query("promptId")
	if promptID == "" {
		s.errorResponse(c, http.StatusBadRequest, "promptId is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brand := brandInfo.Name

	// Parse time parameters
	var startTime, endTime *time.Time
	if startStr := c.Query("startTime"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("endTime"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = &t
		}
	}

	ctx := c.Request.Context()

	// Try to get cached data first
	query := models.AnalyticsCacheQuery{
		PromptID:  promptID,
		Brand:     brand,
		StartTime: startTime,
		EndTime:   endTime,
	}

	cachedData, err := s.db.GetCachedPromptTimeSeries(ctx, query)
	if err == nil && cachedData != nil {
		// Return cached data
		response := &models.PromptTimeSeriesResponse{
			PromptID:   cachedData.PromptID,
			PromptText: cachedData.PromptText,
			PromptType: cachedData.PromptType,
			Category:   cachedData.Category,
			Brand:      cachedData.Brand,
			Period:     cachedData.StartTime.Format("2006-01-02") + " to " + cachedData.EndTime.Format("2006-01-02"),
			Overview:   convertCachedOverview(cachedData.Overview),
			TimeSeries: convertCachedTimeSeries(cachedData.TimeSeries),
		}

		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Data:    response,
			Message: "Prompt time series analytics retrieved from cache",
		})
		return
	}

	// Compute prompt time series analytics if not cached
	promptTimeSeriesService := services.NewPromptTimeSeriesService(s.db)
	timeSeries, err := promptTimeSeriesService.GetPromptTimeSeries(ctx, promptID, brand, startTime, endTime)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get prompt time series: "+err.Error())
		return
	}

	// Cache the computed data for future requests
	go func() {
		cacheStartTime := time.Now().Add(-30 * 24 * time.Hour)
		cacheEndTime := time.Now()
		if startTime != nil {
			cacheStartTime = *startTime
		}
		if endTime != nil {
			cacheEndTime = *endTime
		}

		cachedTimeSeries := &models.CachedPromptTimeSeries{
			ID:         uuid.New().String(),
			PromptID:   promptID,
			Brand:      brand,
			StartTime:  cacheStartTime,
			EndTime:    cacheEndTime,
			PromptText: timeSeries.PromptText,
			PromptType: timeSeries.PromptType,
			Category:   timeSeries.Category,
			Overview:   convertOverviewToCache(timeSeries.Overview),
			TimeSeries: convertTimeSeriesToCache(timeSeries.TimeSeries),
		}
		s.db.SaveCachedPromptTimeSeries(context.Background(), cachedTimeSeries)
	}()

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    timeSeries,
		Message: "Prompt time series analytics retrieved successfully",
	})
}

// Helper functions to convert between API and cache models
func convertCachedOverview(cached models.PromptTimeSeriesOverview) models.PromptTimeSeriesOverview {
	llmBreakdown := make(map[string]models.PromptLLMStats)
	for k, v := range cached.LLMBreakdown {
		llmBreakdown[k] = models.PromptLLMStats{
			LLMName:       v.LLMName,
			LLMProvider:   v.LLMProvider,
			ResponseCount: v.ResponseCount,
			MentionRate:   v.MentionRate,
			AvgVisibility: v.AvgVisibility,
		}
	}

	return models.PromptTimeSeriesOverview{
		TotalResponses:     cached.TotalResponses,
		TotalMentions:      cached.TotalMentions,
		AvgVisibility:      cached.AvgVisibility,
		AvgPosition:        cached.AvgPosition,
		MentionRate:        cached.MentionRate,
		TopPositionRate:    cached.TopPositionRate,
		GroundingRate:      cached.GroundingRate,
		EffectivenessScore: cached.EffectivenessScore,
		EffectivenessGrade: cached.EffectivenessGrade,
		PositiveSentiment:  cached.PositiveSentiment,
		NeutralSentiment:   cached.NeutralSentiment,
		NegativeSentiment:  cached.NegativeSentiment,
		LLMBreakdown:       llmBreakdown,
		TopCompetitors:     cached.TopCompetitors,
	}
}

func convertCachedTimeSeries(cached []models.PromptTimeSeriesDataPoint) []models.PromptTimeSeriesDataPoint {
	result := make([]models.PromptTimeSeriesDataPoint, len(cached))
	for i, dp := range cached {
		result[i] = models.PromptTimeSeriesDataPoint{
			Date:           dp.Date,
			ResponseCount:  dp.ResponseCount,
			MentionCount:   dp.MentionCount,
			AvgVisibility:  dp.AvgVisibility,
			AvgPosition:    dp.AvgPosition,
			MentionRate:    dp.MentionRate,
			GroundingCount: dp.GroundingCount,
			PositiveCount:  dp.PositiveCount,
			NeutralCount:   dp.NeutralCount,
			NegativeCount:  dp.NegativeCount,
		}
	}
	return result
}

func convertOverviewToCache(overview models.PromptTimeSeriesOverview) models.PromptTimeSeriesOverview {
	llmBreakdown := make(map[string]models.PromptLLMStats)
	for k, v := range overview.LLMBreakdown {
		llmBreakdown[k] = models.PromptLLMStats{
			LLMName:       v.LLMName,
			LLMProvider:   v.LLMProvider,
			ResponseCount: v.ResponseCount,
			MentionRate:   v.MentionRate,
			AvgVisibility: v.AvgVisibility,
		}
	}

	return models.PromptTimeSeriesOverview{
		TotalResponses:     overview.TotalResponses,
		TotalMentions:      overview.TotalMentions,
		AvgVisibility:      overview.AvgVisibility,
		AvgPosition:        overview.AvgPosition,
		MentionRate:        overview.MentionRate,
		TopPositionRate:    overview.TopPositionRate,
		GroundingRate:      overview.GroundingRate,
		EffectivenessScore: overview.EffectivenessScore,
		EffectivenessGrade: overview.EffectivenessGrade,
		PositiveSentiment:  overview.PositiveSentiment,
		NeutralSentiment:   overview.NeutralSentiment,
		NegativeSentiment:  overview.NegativeSentiment,
		LLMBreakdown:       llmBreakdown,
		TopCompetitors:     overview.TopCompetitors,
	}
}

func convertTimeSeriesToCache(timeSeries []models.PromptTimeSeriesDataPoint) []models.PromptTimeSeriesDataPoint {
	result := make([]models.PromptTimeSeriesDataPoint, len(timeSeries))
	for i, dp := range timeSeries {
		result[i] = models.PromptTimeSeriesDataPoint{
			Date:           dp.Date,
			ResponseCount:  dp.ResponseCount,
			MentionCount:   dp.MentionCount,
			AvgVisibility:  dp.AvgVisibility,
			AvgPosition:    dp.AvgPosition,
			MentionRate:    dp.MentionRate,
			GroundingCount: dp.GroundingCount,
			PositiveCount:  dp.PositiveCount,
			NeutralCount:   dp.NeutralCount,
			NegativeCount:  dp.NegativeCount,
		}
	}
	return result
}

// normalizeDomainForSource normalizes a citation source domain to include www. prefix
// Example: "dev.to" -> "www.dev.to", "https://betterstack.com" -> "www.betterstack.com"
func normalizeDomainForSource(domain string) string {
	// Remove protocol if present
	normalized := strings.TrimPrefix(domain, "http://")
	normalized = strings.TrimPrefix(normalized, "https://")

	// Remove path if present (everything after first /)
	if idx := strings.Index(normalized, "/"); idx != -1 {
		normalized = normalized[:idx]
	}

	// Remove port if present (everything after :)
	if idx := strings.Index(normalized, ":"); idx != -1 {
		normalized = normalized[:idx]
	}

	// Add www. prefix if not present
	if !strings.HasPrefix(normalized, "www.") {
		normalized = "www." + normalized
	}

	return normalized
}

// deriveDomainFromName derives a domain from a brand/competitor name
// This is a simplified version for API layer
func deriveDomainFromName(name string) string {
	// If it already looks like a domain, normalize it
	if strings.Contains(name, ".") {
		normalized := strings.ToLower(strings.TrimSpace(name))
		normalized = strings.TrimPrefix(normalized, "http://")
		normalized = strings.TrimPrefix(normalized, "https://")
		if idx := strings.Index(normalized, "/"); idx != -1 {
			normalized = normalized[:idx]
		}
		if !strings.HasPrefix(normalized, "www.") {
			return "www." + normalized
		}
		return normalized
	}

	// Clean the name
	cleaned := strings.ToLower(strings.TrimSpace(name))
	// Remove content in parentheses
	for {
		openIdx := strings.Index(cleaned, "(")
		if openIdx == -1 {
			break
		}
		closeIdx := strings.Index(cleaned[openIdx:], ")")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx
		cleaned = cleaned[:openIdx] + cleaned[closeIdx+1:]
	}

	// Extract first word
	words := strings.Fields(cleaned)
	if len(words) > 0 {
		cleaned = words[0]
	}

	// Remove invalid characters
	var result strings.Builder
	for _, r := range cleaned {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	cleaned = result.String()

	// Special cases
	if strings.Contains(cleaned, "visualstudio") || strings.Contains(cleaned, "vscode") {
		return "www.code.visualstudio.com"
	}

	return "www." + cleaned + ".com"
}

// parseStringArray parses a comma-separated string into a slice of strings
func parseStringArray(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// parseCompetitorsFromQuery parses competitors from query string
// Expected format: "name1|domain1,name2|domain2" or "name1,name2"
func parseCompetitorsFromQuery(s string) []models.Competitor {
	if s == "" {
		return []models.Competitor{}
	}
	parts := strings.Split(s, ",")
	competitors := make([]models.Competitor, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var name, domain string
		if idx := strings.Index(part, "|"); idx != -1 {
			name = strings.TrimSpace(part[:idx])
			domain = strings.TrimSpace(part[idx+1:])
		} else {
			name = part
			domain = deriveDomainFromName(name)
		}

		if name != "" {
			competitors = append(competitors, models.Competitor{
				Name:   name,
				Domain: domain,
			})
		}
	}
	return competitors
}

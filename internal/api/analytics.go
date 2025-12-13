package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
	"github.com/fissionx/gego/internal/shared"
)

// getSourceAnalytics handles POST /api/v1/geo/analytics/sources
func (s *Server) getSourceAnalytics(c *gin.Context) {
	var req models.SourceAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	if req.TopN == 0 {
		req.TopN = 20
	}

	ctx := c.Request.Context()

	// Try to get cached data first
	query := models.AnalyticsCacheQuery{
		Brand:     req.Brand,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	cachedData, err := s.db.GetCachedSourceAnalytics(ctx, query)
	if err == nil && cachedData != nil {
		// Return cached data
		response := &models.SourceAnalyticsResponse{
			Brand:           cachedData.Brand,
			LogoURL:         cachedData.LogoURL,
			FallbackLogoURL: cachedData.FallbackLogoURL,
			Period:          cachedData.Period,
			TopSources:      cachedData.TopSources,
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

	// Compute analytics if not cached
	analytics, err := s.sourceAnalyticsService.GetSourceAnalytics(
		ctx,
		req.Brand,
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
			Brand:           req.Brand,
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

// getCompetitiveBenchmark handles POST /api/v1/geo/analytics/competitive
func (s *Server) getCompetitiveBenchmark(c *gin.Context) {
	var req models.CompetitiveBenchmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.MainBrand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Main brand is required")
		return
	}

	ctx := c.Request.Context()

	// Try to get cached data first (only if no specific competitors are requested)
	if len(req.Competitors) == 0 {
		query := models.AnalyticsCacheQuery{
			Brand:     req.MainBrand,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		}

		cachedData, err := s.db.GetCachedCompetitiveBenchmark(ctx, query)
		if err == nil && cachedData != nil {
			// Return cached data
			response := &models.CompetitiveBenchmarkResponse{
				MainBrand:       cachedData.MainBrandPerf,
				Competitors:     cachedData.Competitors,
				MarketLeader:    cachedData.MarketLeader,
				YourRank:        cachedData.YourRank,
				TotalBrands:     cachedData.TotalBrands,
				PromptBreakdown: cachedData.PromptBreakdown,
				Recommendations: cachedData.Recommendations,
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

	// Compute benchmark if not cached or specific competitors requested
	benchmark, err := s.competitiveBenchmarkService.GetCompetitiveBenchmark(
		ctx,
		req.MainBrand,
		req.Competitors,
		req.PromptIDs,
		req.LLMIDs,
		req.StartTime,
		req.EndTime,
		req.Region,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get competitive benchmark: "+err.Error())
		return
	}

	// Cache the computed data for future requests (only for auto-detected competitors)
	if len(req.Competitors) == 0 {
		go func() {
			startTime := time.Now().Add(-30 * 24 * time.Hour)
			endTime := time.Now()
			if req.StartTime != nil {
				startTime = *req.StartTime
			}
			if req.EndTime != nil {
				endTime = *req.EndTime
			}

			cachedBenchmark := &models.CachedCompetitiveBenchmark{
				ID:              uuid.New().String(),
				MainBrand:       req.MainBrand,
				StartTime:       startTime,
				EndTime:         endTime,
				MainBrandPerf:   benchmark.MainBrand,
				Competitors:     benchmark.Competitors,
				MarketLeader:    benchmark.MarketLeader,
				YourRank:        benchmark.YourRank,
				TotalBrands:     benchmark.TotalBrands,
				PromptBreakdown: benchmark.PromptBreakdown,
				Recommendations: benchmark.Recommendations,
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

// getPositionAnalytics handles POST /api/v1/geo/analytics/position
func (s *Server) getPositionAnalytics(c *gin.Context) {
	var req struct {
		Brand     string     `json:"brand" binding:"required"`
		StartTime *time.Time `json:"start_time,omitempty"`
		EndTime   *time.Time `json:"end_time,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Get position analytics
	analytics, err := getPositionAnalyticsForBrand(
		c.Request.Context(),
		s.db,
		req.Brand,
		req.StartTime,
		req.EndTime,
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
	brandLogo := logoService.GetBrandLogo(ctx, brand, "")

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

// getPromptPerformance handles POST /api/v1/geo/analytics/prompt-performance
func (s *Server) getPromptPerformance(c *gin.Context) {
	var req models.PromptPerformanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	if req.MinResponses == 0 {
		req.MinResponses = 3
	}

	ctx := c.Request.Context()

	// Try to get cached data first
	query := models.AnalyticsCacheQuery{
		Brand:     req.Brand,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	cachedData, err := s.db.GetCachedPromptPerformance(ctx, query)
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

	// Compute prompt performance analytics if not cached
	performance, err := s.promptPerformanceService.GetPromptPerformance(
		ctx,
		req.Brand,
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
			Brand:                req.Brand,
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

// getPromptTimeSeries handles GET /api/v1/geo/analytics/prompt-timeseries
func (s *Server) getPromptTimeSeries(c *gin.Context) {
	// Get query parameters
	promptID := c.Query("promptId")
	if promptID == "" {
		s.errorResponse(c, http.StatusBadRequest, "promptId is required")
		return
	}

	brand := c.Query("brand")

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

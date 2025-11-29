package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AI2HU/gego/internal/db"
	"github.com/AI2HU/gego/internal/models"
	"github.com/AI2HU/gego/internal/shared"
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

	// Get source analytics
	analytics, err := s.sourceAnalyticsService.GetSourceAnalytics(
		c.Request.Context(),
		req.Brand,
		req.StartTime,
		req.EndTime,
		req.TopN,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get source analytics: "+err.Error())
		return
	}

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

	if len(req.Competitors) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one competitor is required")
		return
	}

	// Get competitive benchmark
	benchmark, err := s.competitiveBenchmarkService.GetCompetitiveBenchmark(
		c.Request.Context(),
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

	response := &models.PositionAnalyticsResponse{
		Brand:             brand,
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

	// Get prompt performance analytics
	performance, err := s.promptPerformanceService.GetPromptPerformance(
		c.Request.Context(),
		req.Brand,
		req.StartTime,
		req.EndTime,
		req.MinResponses,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get prompt performance: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    performance,
		Message: "Prompt performance retrieved successfully",
	})
}

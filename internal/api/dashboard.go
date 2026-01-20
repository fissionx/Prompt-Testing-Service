package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// getDashboardOverview handles GET /api/v1/geo/brand/:brandId/dashboard/overview
func (s *Server) getDashboardOverview(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "brandId is required")
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

	overview, err := s.dashboardService.GetDashboardOverview(ctx, brandID, brandInfo.OrgID, brandName, startTime, endTime)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get dashboard overview: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    overview,
		Message: "Dashboard overview retrieved successfully",
	})
}

// getPromptExecutionStatus handles GET /api/v1/geo/brand/:brandId/prompts/execution/status
func (s *Server) getPromptExecutionStatus(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "brandId is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name
	ctx := c.Request.Context()

	status, err := s.dashboardService.GetPromptExecutionStatus(ctx, brandID, brandName)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get prompt execution status: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    status,
		Message: "Prompt execution status retrieved successfully",
	})
}

// getModelAnalytics handles GET /api/v1/geo/brand/:brandId/analytics/models
func (s *Server) getModelAnalytics(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	var req models.ModelAnalyticsRequest
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

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	ctx := c.Request.Context()

	analytics, err := s.dashboardService.GetModelAnalytics(ctx, req.BrandID, brandInfo.OrgID, brandName, req.StartTime, req.EndTime)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get model analytics: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    analytics,
		Message: "Model analytics retrieved successfully",
	})
}

// getCompetitorMatrix handles POST /api/v1/geo/analytics/competitor-matrix
func (s *Server) getCompetitorMatrix(c *gin.Context) {
	var req models.CompetitorMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.MainBrandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "MainBrandId is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.MainBrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	mainBrandName := brandInfo.Name
	mainBrandDomain := shared.NormalizeDomainToURL(brandInfo.Domain)

	ctx := c.Request.Context()

	// Convert Competitor objects to strings for internal processing
	competitorNames := make([]string, 0, len(req.Competitors))
	competitorMap := make(map[string]string) // name -> domain mapping
	for _, comp := range req.Competitors {
		competitorNames = append(competitorNames, comp.Name)
		if comp.Domain != "" {
			competitorMap[comp.Name] = shared.NormalizeDomainToURL(comp.Domain)
		}
	}

	// Ensure main brand domain is in the map
	if mainBrandDomain != "" {
		competitorMap[mainBrandName] = mainBrandDomain
	}

	matrix, err := s.dashboardService.GetCompetitorMatrix(
		ctx,
		mainBrandName,
		competitorNames,
		req.StartTime,
		req.EndTime,
		competitorMap,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get competitor matrix: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    matrix,
		Message: "Competitor matrix retrieved successfully",
	})
}

// getTrendComparison handles GET /api/v1/geo/brand/:brandId/analytics/trend-comparison
func (s *Server) getTrendComparison(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	var req models.TrendComparisonRequest
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

	// Parse array parameters (competitors)
	if competitorsStr := c.Query("competitors"); competitorsStr != "" {
		req.Competitors = parseCompetitorsFromQuery(competitorsStr)
	}

	// Parse metric and granularity
	if metricStr := c.Query("metric"); metricStr != "" {
		req.Metric = metricStr
	}
	if granularityStr := c.Query("granularity"); granularityStr != "" {
		req.Granularity = granularityStr
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

	// Convert Competitor objects to strings for internal processing
	competitorNames := make([]string, 0, len(req.Competitors))
	competitorMap := make(map[string]string) // name -> domain mapping
	for _, comp := range req.Competitors {
		competitorNames = append(competitorNames, comp.Name)
		if comp.Domain != "" {
			competitorMap[comp.Name] = shared.NormalizeDomainToURL(comp.Domain)
		}
	}

	// Ensure main brand domain is in the map
	if mainBrandDomain != "" {
		competitorMap[mainBrandName] = mainBrandDomain
	}

	trends, err := s.dashboardService.GetTrendComparison(
		ctx,
		mainBrandName,
		competitorNames,
		req.Metric,
		req.StartTime,
		req.EndTime,
		req.Granularity,
		competitorMap,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get trend comparison: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    trends,
		Message: "Trend comparison retrieved successfully",
	})
}

// exportData handles POST /api/v1/geo/export
func (s *Server) exportData(c *gin.Context) {
	var req models.ExportRequest
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

	// Set brand name in request for service
	req.Brand = brandInfo.Name

	if req.ExportType == "" {
		s.errorResponse(c, http.StatusBadRequest, "Export type is required")
		return
	}

	ctx := c.Request.Context()

	exportData, err := s.exportService.Export(ctx, req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to export data: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    exportData,
		Message: "Data exported successfully",
	})
}

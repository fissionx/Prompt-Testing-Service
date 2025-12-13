package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
)

// getDashboardOverview handles GET /api/v1/geo/dashboard/overview
func (s *Server) getDashboardOverview(c *gin.Context) {
	brand := c.Query("brand")
	if brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "brand is required")
		return
	}

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

	overview, err := s.dashboardService.GetDashboardOverview(ctx, brand, startTime, endTime)
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

// getModelAnalytics handles POST /api/v1/geo/analytics/models
func (s *Server) getModelAnalytics(c *gin.Context) {
	var req models.ModelAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

	ctx := c.Request.Context()

	analytics, err := s.dashboardService.GetModelAnalytics(ctx, req.Brand, req.StartTime, req.EndTime)
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

	if req.MainBrand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Main brand is required")
		return
	}

	ctx := c.Request.Context()

	matrix, err := s.dashboardService.GetCompetitorMatrix(
		ctx,
		req.MainBrand,
		req.Competitors,
		req.StartTime,
		req.EndTime,
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

// getTrendComparison handles POST /api/v1/geo/analytics/trend-comparison
func (s *Server) getTrendComparison(c *gin.Context) {
	var req models.TrendComparisonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.MainBrand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Main brand is required")
		return
	}

	ctx := c.Request.Context()

	trends, err := s.dashboardService.GetTrendComparison(
		ctx,
		req.MainBrand,
		req.Competitors,
		req.Metric,
		req.StartTime,
		req.EndTime,
		req.Granularity,
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

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

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

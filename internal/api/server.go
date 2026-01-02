package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
)

// Server represents the API server
type Server struct {
	db                          db.Database
	llmService                  *services.LLMService
	promptService               *services.PromptManagementService
	scheduleService             *services.ScheduleService
	statsService                *services.StatsService
	searchService               *services.SearchService
	sourceAnalyticsService      *services.SourceAnalyticsService
	competitiveBenchmarkService *services.CompetitiveBenchmarkService
	promptPerformanceService    *services.PromptPerformanceService
	scheduledCampaignManager    *services.ScheduledCampaignManager
	dashboardService            *services.DashboardService
	exportService               *services.ExportService
	competitorService           *services.CompetitorService
	brandPromptService          *services.BrandPromptService
	brandService                *services.BrandService
	llmRegistry                 *llm.Registry
	router                      *gin.Engine
	corsOrigin                  string
}

// NewServer creates a new API server
func NewServer(database db.Database, llmRegistry *llm.Registry, corsOrigin string) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	allowedOrigins := parseAllowedOrigins(corsOrigin)

	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigin := getAllowedOrigin(origin, allowedOrigins, corsOrigin)

		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH, HEAD")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	scheduledCampaignManager := services.NewScheduledCampaignManager(database, llmRegistry)
	brandService := services.NewBrandService("") // Uses default URL

	server := &Server{
		db:                          database,
		llmService:                  services.NewLLMService(database),
		promptService:               services.NewPromptManagementService(database),
		scheduleService:             services.NewScheduleService(database),
		statsService:                services.NewStatsService(database),
		searchService:               services.NewSearchService(database),
		sourceAnalyticsService:      services.NewSourceAnalyticsService(database),
		competitiveBenchmarkService: services.NewCompetitiveBenchmarkService(database),
		promptPerformanceService:    services.NewPromptPerformanceService(database),
		scheduledCampaignManager:    scheduledCampaignManager,
		dashboardService:            services.NewDashboardService(database),
		exportService:               services.NewExportService(database),
		competitorService:           services.NewCompetitorService(database, llmRegistry),
		brandPromptService:          services.NewBrandPromptService(database, llmRegistry),
		brandService:                brandService,
		llmRegistry:                 llmRegistry,
		router:                      router,
		corsOrigin:                  corsOrigin,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")

	api.GET("/llms", s.listLLMs)
	api.GET("/llms/:id", s.getLLM)
	api.POST("/llms", s.createLLM)
	api.PUT("/llms/:id", s.updateLLM)
	api.DELETE("/llms/:id", s.deleteLLM)
	api.DELETE("/llms", s.deleteAllLLMs)

	api.GET("/prompts", s.listPrompts)
	// More specific routes must come before generic :id routes
	api.GET("/prompts/:id/responses", s.getPromptResponses)
	api.GET("/prompts/:id", s.getPrompt)
	api.POST("/prompts", s.createPrompt)
	api.PUT("/prompts/:id", s.updatePrompt)
	api.DELETE("/prompts/:id", s.deletePrompt)

	api.GET("/schedules", s.listSchedules)
	api.GET("/schedules/:id", s.getSchedule)
	api.POST("/schedules", s.createSchedule)
	api.PUT("/schedules/:id", s.updateSchedule)
	api.DELETE("/schedules/:id", s.deleteSchedule)

	api.GET("/stats", s.getStats)

	api.POST("/search", s.search)

	api.GET("/responses", s.listResponses)
	api.GET("/brand/:brand/prompts/responses", s.getBrandPromptsWithLatestResponses)

	api.POST("/execute", s.execute)

	// GEO (Generative Engine Optimization) endpoints
	geo := api.Group("/geo")
	{
		// Prompt Generation & Library
		geo.POST("/prompts/generate", s.generatePrompts)
		geo.GET("/libraries", s.listPromptLibraries)

		// Brand Profiles
		geo.GET("/profiles", s.listBrandProfiles)
		geo.GET("/profiles/:brand", s.getBrandProfile)

		// Bulk Execution
		geo.POST("/execute/bulk", s.bulkExecute)

		// Analytics & Insights
		geo.POST("/insights", s.getGEOInsights)

		// NEW: Advanced Analytics
		geo.GET("/analytics/sources", s.getSourceAnalytics)
		geo.POST("/analytics/competitive", s.getCompetitiveBenchmark)
		geo.POST("/analytics/position", s.getPositionAnalytics)
		geo.POST("/analytics/prompt-performance", s.getPromptPerformance)
		geo.GET("/analytics/prompt-timeseries", s.getPromptTimeSeries)

		// Dashboard & Overview
		geo.GET("/dashboard/overview", s.getDashboardOverview)
		geo.POST("/analytics/models", s.getModelAnalytics)
		geo.POST("/analytics/competitor-matrix", s.getCompetitorMatrix)
		geo.POST("/analytics/trend-comparison", s.getTrendComparison)

		// Export
		geo.POST("/export", s.exportData)

		// Competitor Management
		geo.POST("/competitors", s.saveCompetitors)
		geo.GET("/competitors", s.getCompetitors)
		geo.DELETE("/competitors", s.deleteCompetitors)

		// Prompt Management (prompts per brand)
		geo.GET("/prompts", s.getBrandPrompts)
		geo.POST("/prompts/save", s.saveCustomPrompts)
		geo.POST("/prompts/execute/bulk", s.saveAndExecutePrompts)
		geo.DELETE("/prompts", s.deletePromptsByIDs)         // Deletes prompts by IDs (request body with promptIds array)
		geo.DELETE("/prompts/brand", s.deletePromptsByBrand) // Deletes all prompts by brand (query param ?brand=X)
	}

	api.GET("/health", s.healthCheck)
}

// Run starts the API server
func (s *Server) Run(address string) error {
	// Start the scheduled campaign manager
	if err := s.scheduledCampaignManager.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start scheduled campaign manager: %w", err)
	}

	return s.router.Run(address)
}

// Stop stops the API server components
func (s *Server) Stop() {
	s.scheduledCampaignManager.Stop()
}

// Helper functions
func (s *Server) successResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    data,
	})
}

func (s *Server) errorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, models.APIResponse{
		Success: false,
		Error:   message,
	})
}

func (s *Server) parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	return page, limit
}

func parseAllowedOrigins(corsOrigin string) []string {
	if corsOrigin == "" || corsOrigin == "*" {
		return nil
	}

	origins := strings.Split(corsOrigin, ",")
	allowed := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return allowed
}

func getAllowedOrigin(requestOrigin string, allowedOrigins []string, corsOrigin string) string {
	if corsOrigin == "*" {
		return "*"
	}

	if requestOrigin == "" {
		return ""
	}

	for _, allowed := range allowedOrigins {
		if requestOrigin == allowed {
			return allowed
		}
	}

	return ""
}

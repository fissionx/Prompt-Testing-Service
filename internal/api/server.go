package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/config"
	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	applogger "github.com/fissionx/gego/internal/logger"
	"github.com/fissionx/gego/internal/middleware"
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
	schedulerService            *services.SchedulerService
	dashboardService            *services.DashboardService
	exportService               *services.ExportService
	competitorService           *services.CompetitorService
	brandPromptService          *services.BrandPromptService
	brandService                *services.BrandService
	llmRegistry                 *llm.Registry
	router                      *gin.Engine
	corsOrigin                  string
	clerkConfig                 config.ClerkConfig
	logger                      config.Logger
}

// NewServer creates a new API server
func NewServer(database db.Database, llmRegistry *llm.Registry, corsOrigin string, clerkConfig config.ClerkConfig, logger config.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// CORS configuration
	// For production: set specific allowed origins
	// For development: allow localhost and common dev ports
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept", "Accept-Encoding", "Authorization", "X-Requested-With", "X-Org-Id"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,  // Required for cookies
		MaxAge:           86400, // 24 hours
	}

	// Get allowed origins from environment or use defaults
	if allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); allowedOrigins != "" {
		corsConfig.AllowOrigins = strings.Split(allowedOrigins, ",")
	} else {
		// Default: allow common development origins + Vercel deployments
		corsConfig.AllowOrigins = []string{
			// Production/Staging
			"https://app.geo.fissionx.ai",
			"https://fissionx-geo.vercel.app",
			// Local development
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:3002",
			"http://localhost:4000",
			"http://localhost:5173",
			"http://localhost:5174",
			"http://localhost:8000",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		}
	}

	router.Use(cors.New(corsConfig))

	// Request latency tracking middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Initialize database total time tracking in context (for PostgreSQL)
		dbTotalTime := time.Duration(0)
		ctx := context.WithValue(c.Request.Context(), "mongo_total_time", &dbTotalTime)
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Calculate latencies
		apiDuration := time.Since(start)
		dbDuration := dbTotalTime
		status := c.Writer.Status()

		// Log single summary line with API time and database time
		// Use the internal logger package (not config.Logger) for request logging
		applogger.GetLogger().Info(
			"[REQUEST] method=%s path=%s status=%d api_time_ms=%.2f db_time_ms=%.2f",
			method,
			path,
			status,
			float64(apiDuration.Nanoseconds())/1e6,
			float64(dbDuration.Nanoseconds())/1e6,
		)
	})

	scheduledCampaignManager := services.NewScheduledCampaignManager(database, llmRegistry)
	schedulerService := services.NewSchedulerService(database, llmRegistry)
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
		schedulerService:            schedulerService,
		dashboardService:            services.NewDashboardService(database),
		exportService:               services.NewExportService(database),
		competitorService:           services.NewCompetitorService(database, llmRegistry),
		brandPromptService:          services.NewBrandPromptService(database, llmRegistry),
		brandService:                brandService,
		llmRegistry:                 llmRegistry,
		router:                      router,
		corsOrigin:                  corsOrigin,
		clerkConfig:                 clerkConfig,
		logger:                      logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// =========================================
	// PUBLIC ROUTES (No authentication required)
	// =========================================
	public := s.router.Group("/api/v1")
	{
		// Health check route
		public.GET("/health", s.healthCheck)
	}

	// =========================================
	// PROTECTED ROUTES (Authentication required)
	// =========================================
	api := s.router.Group("/api/v1")

	// Check environment variable to determine if auth should be enabled
	// If GEGO_ENV is set to "local", skip authentication middleware
	env := strings.ToLower(os.Getenv("GEGO_ENV"))
	isLocalEnv := env == "local"

	// Apply Clerk authentication middleware
	// Skip auth if:
	// 1. GEGO_ENV is set to "local" (local development mode)
	// 2. CLERK_SECRET_KEY is not configured
	authEnabled := !isLocalEnv && s.clerkConfig.SecretKey != ""
	if authEnabled {
		api.Use(middleware.ClerkAuth(s.clerkConfig, s.logger))
		if s.logger != nil {
			s.logger.Info("Clerk authentication enabled for protected routes")
		}
	} else {
		if s.logger != nil {
			if isLocalEnv {
				s.logger.Info("Clerk authentication DISABLED - GEGO_ENV=local (local development mode)")
			} else {
				s.logger.Warn("Clerk authentication DISABLED - CLERK_SECRET_KEY not set")
			}
		}
	}

	// LLM endpoints
	api.GET("/llms", s.listLLMs)
	api.GET("/llms/:id", s.getLLM)
	api.POST("/llms", s.createLLM)
	api.PUT("/llms/:id", s.updateLLM)
	api.DELETE("/llms/:id", s.deleteLLM)
	api.DELETE("/llms", s.deleteAllLLMs)

	// Prompt endpoints
	api.GET("/prompts", s.listPrompts)
	// More specific routes must come before generic :id routes
	api.GET("/prompts/:id/responses", s.getPromptResponses)
	api.GET("/prompts/:id", s.getPrompt)
	api.POST("/prompts", s.createPrompt)
	api.PUT("/prompts/:id", s.updatePrompt)
	api.DELETE("/prompts/:id", s.deletePrompt)

	// Schedule endpoints
	api.GET("/brand/:brandId/schedules", s.listSchedules)
	api.GET("/brand/:brandId/schedules/:id", s.getSchedule)
	api.POST("/brand/:brandId/schedules", s.createSchedule)
	api.PUT("/brand/:brandId/schedules/:id", s.updateSchedule)
	api.DELETE("/brand/:brandId/schedules/:id", s.deleteSchedule)

	// Stats endpoint
	api.GET("/stats", s.getStats)

	// Search endpoint
	api.POST("/search", s.search)

	// Response endpoints
	api.GET("/responses", s.listResponses)
	api.GET("/brand/:brandId/prompts/responses", s.getBrandPromptsWithLatestResponses)

	// Brand endpoints
	api.GET("/brands/:id", s.getBrand)

	// Execute endpoint
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
		geo.GET("/brand/:brandId/analytics/sources", s.getSourceAnalytics)
		geo.GET("/brand/:brandId/analytics/competitive", s.getCompetitiveBenchmark)
		geo.GET("/brand/:brandId/analytics/position", s.getPositionAnalytics)
		geo.GET("/brand/:brandId/analytics/prompt-performance", s.getPromptPerformance)
		geo.GET("/brand/:brandId/analytics/prompt-timeseries", s.getPromptTimeSeries)

		// Dashboard & Overview
		geo.GET("/brand/:brandId/dashboard/overview", s.getDashboardOverview)
		geo.GET("/brand/:brandId/prompts/execution/status", s.getPromptExecutionStatus)
		geo.GET("/brand/:brandId/analytics/models", s.getModelAnalytics)
		geo.POST("/analytics/competitor-matrix", s.getCompetitorMatrix)
		geo.GET("/brand/:brandId/analytics/trend-comparison", s.getTrendComparison)

		// Export
		geo.POST("/export", s.exportData)

		// Competitor Management
		geo.POST("/brand/:brandId/competitors", s.saveCompetitors)
		geo.GET("/brand/:brandId/competitors", s.getCompetitors)
		geo.DELETE("/brand/:brandId/competitors", s.deleteCompetitors)

		// Prompt Management (prompts per brand)
		geo.GET("/brand/:brandId/prompts", s.getBrandPrompts)
		geo.POST("/brand/:brandId/prompts/save", s.saveCustomPrompts)
		geo.POST("/brand/:brandId/prompts/execute/bulk", s.saveAndExecutePrompts)
		geo.DELETE("/brand/:brandId/prompts", s.deletePromptsByIDs)         // Deletes prompts by IDs (request body with promptIds array)
		geo.DELETE("/brand/:brandId/prompts/brand", s.deletePromptsByBrand) // Deletes all prompts by brand
	}
}

// Run starts the API server
func (s *Server) Run(address string) error {
	// Start the scheduled campaign manager
	if err := s.scheduledCampaignManager.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start scheduled campaign manager: %w", err)
	}

	// Start the scheduler service to handle cron schedules
	if err := s.schedulerService.Start(context.Background()); err != nil {
		// Log error but don't fail startup - scheduler might already be running
		applogger.GetLogger().Info("Failed to start scheduler service (may already be running): %v", err)
	}

	return s.router.Run(address)
}

// Stop stops the API server components
func (s *Server) Stop() {
	s.scheduledCampaignManager.Stop()
	s.schedulerService.Stop()
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

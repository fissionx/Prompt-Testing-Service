package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/fissionx/gego/internal/models"
)

// getStats handles GET /api/v1/stats
func (s *Server) getStats(c *gin.Context) {
	totalResponses, err := s.statsService.GetTotalResponses(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get total responses: "+err.Error())
		return
	}

	totalPrompts, err := s.statsService.GetTotalPrompts(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get total prompts: "+err.Error())
		return
	}

	totalLLMs, err := s.statsService.GetTotalLLMs(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get total LLMs: "+err.Error())
		return
	}

	totalSchedules, err := s.statsService.GetTotalSchedules(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get total schedules: "+err.Error())
		return
	}

	limitStr := c.DefaultQuery("keyword_limit", "10")
	keywordLimit, _ := strconv.Atoi(limitStr)
	if keywordLimit <= 0 || keywordLimit > 100 {
		keywordLimit = 10
	}

	topKeywords, err := s.statsService.GetTopKeywords(c.Request.Context(), keywordLimit, nil, nil)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get top keywords: "+err.Error())
		return
	}

	promptStats, err := s.statsService.GetAllPromptStats(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get prompt stats: "+err.Error())
		return
	}

	llmStats, err := s.statsService.GetAllLLMStats(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get LLM stats: "+err.Error())
		return
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)
	responseTrends, err := s.statsService.GetResponseTrends(c.Request.Context(), startTime, endTime)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get response trends: "+err.Error())
		return
	}

	response := models.StatsResponse{
		TotalResponses: totalResponses,
		TotalPrompts:   totalPrompts,
		TotalLLMs:      totalLLMs,
		TotalSchedules: totalSchedules,
		TopKeywords:    topKeywords,
		PromptStats:    promptStats,
		LLMStats:       llmStats,
		ResponseTrends: responseTrends,
		LastUpdated:    time.Now(),
	}

	s.successResponse(c, response)
}

// healthCheck handles GET /api/v1/health
func (s *Server) healthCheck(c *gin.Context) {
	// Try to ping with a timeout context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	if err := s.db.Ping(ctx); err != nil {
		// Log the error for debugging
		if s.logger != nil {
			s.logger.Error("Health check failed: database ping error", zap.Error(err))
		}
		
		// Determine the specific issue and provide appropriate hints
		errMsg := err.Error()
		var issue, hint string
		
		if strings.Contains(errMsg, "SQL database ping failed") {
			if strings.Contains(errMsg, "database is closed") {
				issue = "sqlite_connection_closed"
				hint = "SQLite database connection was closed. This may indicate filesystem issues on the mounted volume. Check Fly.io volume status."
			} else {
				issue = "sqlite_connection_failed"
				hint = "SQLite database connection failed. Check if the database file is accessible at /app/data/gego.db"
			}
		} else if strings.Contains(errMsg, "NoSQL database ping failed") {
			issue = "postgresql_connection_failed"
			hint = "PostgreSQL connection failed. Check POSTGRESQL_URI secret: flyctl secrets list --app gego"
		} else {
			issue = "database_connection_failed"
			hint = "Database connection failed. Check both SQLite and PostgreSQL configurations."
		}
		
		c.JSON(http.StatusServiceUnavailable, models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Database connection failed: %v", err),
			Data: map[string]interface{}{
				"status":    "unhealthy",
				"timestamp": time.Now(),
				"issue":     issue,
				"hint":      hint,
				"error":     errMsg,
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "1.0.0",
		},
	})
}

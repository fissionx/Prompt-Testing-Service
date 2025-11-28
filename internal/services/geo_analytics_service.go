package services

import (
	"context"
	"fmt"
	"time"

	"github.com/AI2HU/gego/internal/db"
	"github.com/AI2HU/gego/internal/models"
	"github.com/AI2HU/gego/internal/shared"
)

// GEOAnalyticsService provides GEO analytics and insights
type GEOAnalyticsService struct {
	db db.Database
}

// NewGEOAnalyticsService creates a new GEO analytics service
func NewGEOAnalyticsService(database db.Database) *GEOAnalyticsService {
	return &GEOAnalyticsService{
		db: database,
	}
}

// GetGEOInsights computes comprehensive GEO insights for a brand
func (s *GEOAnalyticsService) GetGEOInsights(ctx context.Context, brand string, startTime, endTime *time.Time) (*models.GEOInsightsResponse, error) {
	if brand == "" {
		return nil, fmt.Errorf("brand is required")
	}

	// Fetch all responses for the brand
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000, // Get all responses
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter responses for this brand
	var brandResponses []*models.Response
	for _, resp := range allResponses {
		if resp.Brand == brand {
			brandResponses = append(brandResponses, resp)
		}
	}

	if len(brandResponses) == 0 {
		return &models.GEOInsightsResponse{
			Brand:          brand,
			TotalResponses: 0,
		}, nil
	}

	// Calculate metrics
	insights := &models.GEOInsightsResponse{
		Brand:              brand,
		TotalResponses:     len(brandResponses),
		SentimentBreakdown: make(map[string]int),
	}

	// Aggregate data
	totalVisibility := 0
	mentionedCount := 0
	groundedCount := 0
	competitorCounts := make(map[string]int)
	llmPerformance := make(map[string]*llmStats)
	categoryPerformance := make(map[string]*categoryStats)

	for _, resp := range brandResponses {
		// Visibility
		totalVisibility += resp.VisibilityScore
		if resp.BrandMentioned {
			mentionedCount++
		}
		if resp.InGroundingSources {
			groundedCount++
		}

		// Sentiment
		if resp.Sentiment != "" {
			insights.SentimentBreakdown[resp.Sentiment]++
		}

		// Competitors
		for _, comp := range resp.CompetitorsMention {
			competitorCounts[comp]++
		}

		// LLM performance
		llmKey := fmt.Sprintf("%s-%s", resp.LLMProvider, resp.LLMName)
		if _, exists := llmPerformance[llmKey]; !exists {
			llmPerformance[llmKey] = &llmStats{
				name:     resp.LLMName,
				provider: resp.LLMProvider,
			}
		}
		llmPerformance[llmKey].totalVisibility += resp.VisibilityScore
		llmPerformance[llmKey].totalResponses++
		if resp.BrandMentioned {
			llmPerformance[llmKey].mentionCount++
		}

		// Category performance
		prompt, err := s.db.GetPrompt(ctx, resp.PromptID)
		if err == nil && prompt.Category != "" {
			if _, exists := categoryPerformance[prompt.Category]; !exists {
				categoryPerformance[prompt.Category] = &categoryStats{}
			}
			categoryPerformance[prompt.Category].totalVisibility += resp.VisibilityScore
			categoryPerformance[prompt.Category].totalResponses++
			if resp.BrandMentioned {
				categoryPerformance[prompt.Category].mentionCount++
			}
		}
	}

	// Calculate averages
	insights.AverageVisibility = float64(totalVisibility) / float64(len(brandResponses))
	insights.MentionRate = float64(mentionedCount) / float64(len(brandResponses)) * 100
	insights.GroundingRate = float64(groundedCount) / float64(len(brandResponses)) * 100

	// Top competitors
	for comp, count := range competitorCounts {
		insights.TopCompetitors = append(insights.TopCompetitors, models.CompetitorInsight{
			Name:         comp,
			MentionCount: count,
		})
	}

	// LLM performance
	for _, stats := range llmPerformance {
		insights.PerformanceByLLM = append(insights.PerformanceByLLM, models.LLMPerformance{
			LLMName:       stats.name,
			LLMProvider:   stats.provider,
			Visibility:    float64(stats.totalVisibility) / float64(stats.totalResponses),
			MentionRate:   float64(stats.mentionCount) / float64(stats.totalResponses) * 100,
			ResponseCount: stats.totalResponses,
		})
	}

	// Category performance
	for category, stats := range categoryPerformance {
		insights.PerformanceByCategory = append(insights.PerformanceByCategory, models.CategoryPerformance{
			Category:      category,
			Visibility:    float64(stats.totalVisibility) / float64(stats.totalResponses),
			MentionRate:   float64(stats.mentionCount) / float64(stats.totalResponses) * 100,
			ResponseCount: stats.totalResponses,
		})
	}

	return insights, nil
}

type llmStats struct {
	name            string
	provider        string
	totalVisibility int
	totalResponses  int
	mentionCount    int
}

type categoryStats struct {
	totalVisibility int
	totalResponses  int
	mentionCount    int
}


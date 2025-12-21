package services

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// PromptTimeSeriesService provides prompt time series analytics
type PromptTimeSeriesService struct {
	db db.Database
}

// NewPromptTimeSeriesService creates a new prompt time series service
func NewPromptTimeSeriesService(database db.Database) *PromptTimeSeriesService {
	return &PromptTimeSeriesService{
		db: database,
	}
}

// GetPromptTimeSeries computes time series analytics for a specific prompt
func (s *PromptTimeSeriesService) GetPromptTimeSeries(
	ctx context.Context,
	promptID string,
	brand string,
	startTime, endTime *time.Time,
) (*models.PromptTimeSeriesResponse, error) {
	// Get prompt details
	prompt, err := s.db.GetPrompt(ctx, promptID)
	if err != nil {
		return nil, err
	}

	// Fetch all responses for the prompt
	filter := shared.ResponseFilter{
		PromptID:  promptID,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter by brand if specified
	var responses []*models.Response
	for _, resp := range allResponses {
		if brand == "" || resp.Brand == brand {
			responses = append(responses, resp)
		}
	}

	// Build response
	response := &models.PromptTimeSeriesResponse{
		PromptID:   promptID,
		PromptText: prompt.Template,
		PromptType: string(prompt.PromptType),
		Category:   prompt.Category,
		Brand:      brand,
	}

	// Determine period string
	if startTime != nil && endTime != nil {
		response.Period = startTime.Format("2006-01-02") + " to " + endTime.Format("2006-01-02")
	} else {
		response.Period = "all-time"
	}

	if len(responses) == 0 {
		response.Overview = models.PromptTimeSeriesOverview{
			LLMBreakdown: make(map[string]models.PromptLLMStats),
		}
		response.TimeSeries = []models.PromptTimeSeriesDataPoint{}
		return response, nil
	}

	// Compute overview statistics
	response.Overview = s.computeOverview(responses)

	// Compute time series
	response.TimeSeries = s.computeTimeSeries(responses)

	return response, nil
}

// computeOverview computes aggregated statistics from responses
func (s *PromptTimeSeriesService) computeOverview(responses []*models.Response) models.PromptTimeSeriesOverview {
	overview := models.PromptTimeSeriesOverview{
		LLMBreakdown: make(map[string]models.PromptLLMStats),
	}

	totalVisibility := 0.0
	totalPosition := 0.0
	positionCount := 0
	topPositionCount := 0
	mentionCount := 0
	groundingCount := 0
	competitorCounts := make(map[string]int)

	// LLM aggregation
	llmStats := make(map[string]*llmAggregator)

	for _, resp := range responses {
		overview.TotalResponses++

		// Visibility
		totalVisibility += float64(resp.VisibilityScore)

		// Mentions
		if resp.BrandMentioned {
			mentionCount++
			overview.TotalMentions++
		}

		// Position
		if resp.BrandPosition > 0 {
			totalPosition += float64(resp.BrandPosition)
			positionCount++
			if resp.BrandPosition <= 3 {
				topPositionCount++
			}
		}

		// Grounding
		if resp.InGroundingSources {
			groundingCount++
		}

		// Sentiment
		switch strings.ToLower(resp.Sentiment) {
		case "positive":
			overview.PositiveSentiment++
		case "neutral":
			overview.NeutralSentiment++
		case "negative":
			overview.NegativeSentiment++
		}

		// Competitors
		for _, comp := range resp.CompetitorsMention {
			competitorCounts[comp]++
		}

		// LLM stats
		llmKey := resp.LLMName
		if _, exists := llmStats[llmKey]; !exists {
			llmStats[llmKey] = &llmAggregator{
				name:     resp.LLMName,
				provider: resp.LLMProvider,
			}
		}
		llmStats[llmKey].totalResponses++
		llmStats[llmKey].totalVisibility += float64(resp.VisibilityScore)
		if resp.BrandMentioned {
			llmStats[llmKey].mentionCount++
		}
	}

	// Calculate averages
	if overview.TotalResponses > 0 {
		overview.AvgVisibility = roundToTwo(totalVisibility / float64(overview.TotalResponses))
		overview.MentionRate = roundToTwo(float64(mentionCount) / float64(overview.TotalResponses) * 100)
		overview.GroundingRate = roundToTwo(float64(groundingCount) / float64(overview.TotalResponses) * 100)
	}

	if positionCount > 0 {
		overview.AvgPosition = roundToTwo(totalPosition / float64(positionCount))
		overview.TopPositionRate = roundToTwo(float64(topPositionCount) / float64(positionCount) * 100)
	}

	// Calculate effectiveness score
	overview.EffectivenessScore = calculateEffectivenessScore(
		overview.AvgVisibility,
		overview.MentionRate,
		overview.TopPositionRate,
		overview.AvgPosition,
	)
	overview.EffectivenessGrade = getEffectivenessGrade(overview.EffectivenessScore)

	// Build LLM breakdown
	for llmName, stats := range llmStats {
		overview.LLMBreakdown[llmName] = models.PromptLLMStats{
			LLMName:       stats.name,
			LLMProvider:   stats.provider,
			ResponseCount: stats.totalResponses,
			MentionRate:   roundToTwo(float64(stats.mentionCount) / float64(stats.totalResponses) * 100),
			AvgVisibility: roundToTwo(stats.totalVisibility / float64(stats.totalResponses)),
		}
	}

	// Get top competitors
	type compCount struct {
		name  string
		count int
	}
	var compList []compCount
	for name, count := range competitorCounts {
		compList = append(compList, compCount{name, count})
	}
	sort.Slice(compList, func(i, j int) bool {
		return compList[i].count > compList[j].count
	})

	// Top 5 competitors - convert to TopCompetitor objects with name and domain
	for i := 0; i < len(compList) && i < 5; i++ {
		compName := compList[i].name
		domain := deriveCompetitorDomainFromName(compName)
		overview.TopCompetitors = append(overview.TopCompetitors, models.TopCompetitor{
			Name:   compName,
			Domain: domain,
		})
	}

	return overview
}

// computeTimeSeries computes daily time series data from responses
func (s *PromptTimeSeriesService) computeTimeSeries(responses []*models.Response) []models.PromptTimeSeriesDataPoint {
	// Group responses by date
	dailyData := make(map[string]*dailyAggregator)

	for _, resp := range responses {
		date := resp.CreatedAt.Format("2006-01-02")

		if _, exists := dailyData[date]; !exists {
			dailyData[date] = &dailyAggregator{
				date: date,
			}
		}

		agg := dailyData[date]
		agg.responseCount++
		agg.totalVisibility += float64(resp.VisibilityScore)

		if resp.BrandMentioned {
			agg.mentionCount++
		}

		if resp.BrandPosition > 0 {
			agg.totalPosition += float64(resp.BrandPosition)
			agg.positionCount++
		}

		if resp.InGroundingSources {
			agg.groundingCount++
		}

		switch strings.ToLower(resp.Sentiment) {
		case "positive":
			agg.positiveCount++
		case "neutral":
			agg.neutralCount++
		case "negative":
			agg.negativeCount++
		}
	}

	// Convert to time series data points
	var timeSeries []models.PromptTimeSeriesDataPoint

	for _, agg := range dailyData {
		dataPoint := models.PromptTimeSeriesDataPoint{
			Date:           agg.date,
			ResponseCount:  agg.responseCount,
			MentionCount:   agg.mentionCount,
			GroundingCount: agg.groundingCount,
			PositiveCount:  agg.positiveCount,
			NeutralCount:   agg.neutralCount,
			NegativeCount:  agg.negativeCount,
		}

		if agg.responseCount > 0 {
			dataPoint.AvgVisibility = roundToTwo(agg.totalVisibility / float64(agg.responseCount))
			dataPoint.MentionRate = roundToTwo(float64(agg.mentionCount) / float64(agg.responseCount) * 100)
		}

		if agg.positionCount > 0 {
			dataPoint.AvgPosition = roundToTwo(agg.totalPosition / float64(agg.positionCount))
		}

		timeSeries = append(timeSeries, dataPoint)
	}

	// Sort by date
	sort.Slice(timeSeries, func(i, j int) bool {
		return timeSeries[i].Date < timeSeries[j].Date
	})

	return timeSeries
}

// Helper structs for aggregation
type llmAggregator struct {
	name            string
	provider        string
	totalResponses  int
	totalVisibility float64
	mentionCount    int
}

type dailyAggregator struct {
	date            string
	responseCount   int
	mentionCount    int
	totalVisibility float64
	totalPosition   float64
	positionCount   int
	groundingCount  int
	positiveCount   int
	neutralCount    int
	negativeCount   int
}

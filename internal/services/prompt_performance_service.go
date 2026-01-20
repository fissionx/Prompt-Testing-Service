package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// PromptPerformanceService analyzes prompt effectiveness for GEO
type PromptPerformanceService struct {
	db          db.Database
	logoService *LogoService
}

// NewPromptPerformanceService creates a new prompt performance service
func NewPromptPerformanceService(database db.Database) *PromptPerformanceService {
	return &PromptPerformanceService{
		db:          database,
		logoService: NewLogoService(database),
	}
}

// GetPromptPerformance analyzes prompt performance for a brand
func (s *PromptPerformanceService) GetPromptPerformance(
	ctx context.Context,
	brandID string,
	orgID string,
	brand string,
	startTime, endTime *time.Time,
	minResponses int,
) (*models.PromptPerformanceResponse, error) {
	if minResponses <= 0 {
		minResponses = 3 // Minimum responses to have meaningful data
	}

	// Get active prompts for this brand - optimized: filter at database level
	enabled := true
	brandPrompts, err := s.db.ListPromptsByBrand(ctx, brand, &enabled)
	if err != nil {
		return nil, err
	}

	// Fetch responses for the brand - optimized: filter by brand at database level
	filter := shared.ResponseFilter{
		Brand:     brand, // Filter at database level instead of in-memory
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	responses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Group responses by prompt
	promptData := make(map[string]*promptPerformanceData)
	for _, resp := range responses {
		if _, exists := promptData[resp.PromptID]; !exists {
			promptData[resp.PromptID] = &promptPerformanceData{
				promptID:  resp.PromptID,
				responses: []*models.Response{},
			}
		}
		promptData[resp.PromptID].responses = append(promptData[resp.PromptID].responses, resp)
	}

	// Calculate performance metrics for ALL active prompts (including those with 0 responses)
	var promptPerformances []models.PromptPerformance
	filteredCount := 0
	totalActivePrompts := len(brandPrompts)

	// Process all active prompts for the brand
	for _, prompt := range brandPrompts {
		data, hasResponses := promptData[prompt.ID]
		
		if !hasResponses {
			// Prompt has no responses - create performance entry with zero metrics
			perf := models.PromptPerformance{
				PromptID:            prompt.ID,
				PromptText:          prompt.Template,
				PromptType:          string(prompt.PromptType),
				Category:            prompt.Category,
				AvgVisibility:       0,
				AvgPosition:         0,
				MentionRate:         0,
				TopPositionRate:     0,
				AvgSentiment:        0,
				TotalResponses:      0,
				BrandMentions:       0,
				EffectivenessScore:  0,
				EffectivenessGrade:  "N/A",
				Status:              "no_data",
				Recommendation:      "No responses yet. Run this prompt to collect performance data.",
			}
			promptPerformances = append(promptPerformances, perf)
			continue
		}

		// Prompt has responses - check if it meets minimum threshold
		if len(data.responses) < minResponses {
			// Include prompts with fewer than minResponses but show them with a note
			perf := s.calculatePromptPerformance(prompt, data.responses)
			perf.Status = "insufficient_data"
			perf.Recommendation = fmt.Sprintf("Only %d response(s) available (minimum %d recommended for accurate analysis). Run more executions to get better insights.", len(data.responses), minResponses)
			promptPerformances = append(promptPerformances, perf)
			filteredCount++
			continue
		}

		// Prompt has sufficient data
		perf := s.calculatePromptPerformance(prompt, data.responses)
		promptPerformances = append(promptPerformances, perf)
	}

	// Sort by effectiveness score (highest first), then by total responses
	sort.Slice(promptPerformances, func(i, j int) bool {
		if promptPerformances[i].EffectivenessScore != promptPerformances[j].EffectivenessScore {
			return promptPerformances[i].EffectivenessScore > promptPerformances[j].EffectivenessScore
		}
		return promptPerformances[i].TotalResponses > promptPerformances[j].TotalResponses
	})

	// Calculate summary statistics (only for prompts with data)
	var promptsWithData []models.PromptPerformance
	for _, perf := range promptPerformances {
		if perf.TotalResponses > 0 {
			promptsWithData = append(promptsWithData, perf)
		}
	}

	avgEffectiveness := 0.0
	if len(promptsWithData) > 0 {
		for _, perf := range promptsWithData {
			avgEffectiveness += perf.EffectivenessScore
		}
		avgEffectiveness /= float64(len(promptsWithData))
	}

	// Identify top and low performers (only from prompts with sufficient data)
	var topPerformers, lowPerformers []string
	for _, perf := range promptPerformances {
		if perf.Status == "high_performer" {
			topPerformers = append(topPerformers, perf.PromptID)
		} else if perf.Status == "low_performer" {
			lowPerformers = append(lowPerformers, perf.PromptID)
		}
	}

	// Determine period
	period := "all-time"
	if startTime != nil && endTime != nil {
		period = startTime.Format("2006-01-02") + " to " + endTime.Format("2006-01-02")
	}

	// Get brand logo
	brandLogo := s.logoService.GetBrandLogo(ctx, brandID, orgID, brand, "")

	return &models.PromptPerformanceResponse{
		Brand:                brand,
		LogoURL:              brandLogo.LogoURL,
		FallbackLogoURL:      brandLogo.FallbackLogoURL,
		Period:               period,
		Prompts:              promptPerformances,
		TopPerformers:        topPerformers,
		LowPerformers:        lowPerformers,
		AvgEffectiveness:     avgEffectiveness,
		TotalPromptsAnalyzed: len(promptPerformances),
		TotalPromptsFound:    totalActivePrompts,
		FilteredOutCount:     filteredCount,
		MinResponsesRequired: minResponses,
	}, nil
}

// calculatePromptPerformance computes performance metrics for a single prompt
func (s *PromptPerformanceService) calculatePromptPerformance(
	prompt *models.Prompt,
	responses []*models.Response,
) models.PromptPerformance {
	totalResponses := len(responses)
	
	// Initialize metrics
	totalVisibility := 0.0
	totalPosition := 0.0
	positionCount := 0
	topPositionCount := 0
	brandMentionCount := 0
	sentimentSum := 0.0
	sentimentCount := 0

	// Aggregate metrics
	for _, resp := range responses {
		// Visibility
		totalVisibility += float64(resp.VisibilityScore)

		// Brand mentions
		if resp.BrandMentioned {
			brandMentionCount++
		}

		// Position tracking
		if resp.BrandPosition > 0 {
			totalPosition += float64(resp.BrandPosition)
			positionCount++

			if resp.BrandPosition <= 3 {
				topPositionCount++
			}
		}

		// Sentiment
		if resp.Sentiment != "" {
			sentimentSum += calculateSentimentScore(resp.Sentiment)
			sentimentCount++
		}
	}

	// Calculate averages
	avgVisibility := totalVisibility / float64(totalResponses)
	mentionRate := float64(brandMentionCount) / float64(totalResponses) * 100

	avgPosition := 0.0
	topPositionRate := 0.0
	if positionCount > 0 {
		avgPosition = totalPosition / float64(positionCount)
		topPositionRate = float64(topPositionCount) / float64(positionCount) * 100
	}

	avgSentiment := 0.0
	if sentimentCount > 0 {
		avgSentiment = sentimentSum / float64(sentimentCount)
	}

	// Calculate effectiveness score (0-100)
	effectivenessScore := calculateEffectivenessScore(
		avgVisibility,
		mentionRate,
		topPositionRate,
		avgPosition,
	)

	// Determine grade and status
	grade := getEffectivenessGrade(effectivenessScore)
	status := getPerformanceStatus(effectivenessScore)
	recommendation := getPromptRecommendation(effectivenessScore, avgVisibility, mentionRate, avgPosition)

	return models.PromptPerformance{
		PromptID:            prompt.ID,
		PromptText:          prompt.Template,
		PromptType:          string(prompt.PromptType),
		Category:            prompt.Category,
		AvgVisibility:       roundToTwo(avgVisibility),
		AvgPosition:         roundToTwo(avgPosition),
		MentionRate:         roundToTwo(mentionRate),
		TopPositionRate:     roundToTwo(topPositionRate),
		AvgSentiment:        roundToTwo(avgSentiment),
		TotalResponses:      totalResponses,
		BrandMentions:       brandMentionCount,
		EffectivenessScore:  roundToTwo(effectivenessScore),
		EffectivenessGrade:  grade,
		Status:              status,
		Recommendation:      recommendation,
	}
}

// calculateEffectivenessScore computes a composite score (0-100)
func calculateEffectivenessScore(avgVisibility, mentionRate, topPositionRate, avgPosition float64) float64 {
	// Weighted scoring:
	// - Visibility: 40% (most important - 0-10 scale)
	// - Mention Rate: 30% (important - 0-100 scale)
	// - Top Position Rate: 20% (valuable - 0-100 scale)
	// - Position: 10% (bonus - inverted, 0-10+ scale)

	visibilityScore := (avgVisibility / 10.0) * 40.0
	mentionScore := (mentionRate / 100.0) * 30.0
	topPosScore := (topPositionRate / 100.0) * 20.0

	// Position score: Better position = higher score
	// Position 1 = 10 points, Position 10 = 0 points
	positionScore := 0.0
	if avgPosition > 0 {
		positionScore = (1.0 - (avgPosition / 10.0)) * 10.0
		if positionScore < 0 {
			positionScore = 0
		}
	}

	totalScore := visibilityScore + mentionScore + topPosScore + positionScore

	// Ensure score is between 0 and 100
	if totalScore > 100 {
		totalScore = 100
	}
	if totalScore < 0 {
		totalScore = 0
	}

	return totalScore
}

// getEffectivenessGrade converts score to letter grade
func getEffectivenessGrade(score float64) string {
	switch {
	case score >= 90:
		return "A+"
	case score >= 85:
		return "A"
	case score >= 80:
		return "A-"
	case score >= 75:
		return "B+"
	case score >= 70:
		return "B"
	case score >= 65:
		return "B-"
	case score >= 60:
		return "C+"
	case score >= 55:
		return "C"
	case score >= 50:
		return "C-"
	case score >= 45:
		return "D+"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}

// getPerformanceStatus categorizes prompt performance
func getPerformanceStatus(score float64) string {
	switch {
	case score >= 75:
		return "high_performer"
	case score >= 50:
		return "average_performer"
	case score >= 30:
		return "low_performer"
	default:
		return "very_low_performer"
	}
}

// getPromptRecommendation provides actionable recommendations
func getPromptRecommendation(score, visibility, mentionRate, position float64) string {
	switch {
	case score >= 80:
		return "Excellent performance! Keep using this prompt frequently. Consider creating similar prompts. Optimize content around this topic."
	case score >= 65:
		return "Good performance. Run this prompt regularly. Monitor for improvement opportunities. Consider A/B testing variations."
	case score >= 50:
		return "Average performance. Analyze why mentions are moderate. Improve content quality around this topic. Test different phrasing."
	case score >= 35:
		if visibility < 3.0 {
			return "Low visibility. Create more authoritative content on this topic. Build citations from high-quality sources."
		} else if mentionRate < 30 {
			return "Low mention rate despite some visibility. Optimize content to be more relevant to this specific question."
		} else if position > 5 {
			return "Brand mentioned but ranked low. Improve competitive positioning. Get more positive reviews and testimonials."
		}
		return "Below average performance. Consider rephrasing or deprioritizing this prompt. Focus resources on higher-performing prompts."
	default:
		return "Very low performance. Consider removing this prompt or completely changing approach. May not be relevant to your brand positioning."
	}
}

// roundToTwo rounds a float to 2 decimal places
func roundToTwo(val float64) float64 {
	return float64(int(val*100+0.5)) / 100
}

// promptPerformanceData is internal helper struct
type promptPerformanceData struct {
	promptID  string
	responses []*models.Response
}


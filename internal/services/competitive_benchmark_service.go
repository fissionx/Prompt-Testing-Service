package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AI2HU/gego/internal/db"
	"github.com/AI2HU/gego/internal/models"
	"github.com/AI2HU/gego/internal/shared"
)

// CompetitiveBenchmarkService provides competitive analysis
type CompetitiveBenchmarkService struct {
	db                    db.Database
	recommendationsEngine *RecommendationsEngine
}

// NewCompetitiveBenchmarkService creates a new competitive benchmark service
func NewCompetitiveBenchmarkService(database db.Database) *CompetitiveBenchmarkService {
	return &CompetitiveBenchmarkService{
		db:                    database,
		recommendationsEngine: NewRecommendationsEngine(),
	}
}

// GetCompetitiveBenchmark performs competitive analysis
// Analyzes how your brand and competitors appear in the SAME responses/prompts
func (s *CompetitiveBenchmarkService) GetCompetitiveBenchmark(
	ctx context.Context,
	mainBrand string,
	competitors []string,
	promptIDs, llmIDs []string,
	startTime, endTime *time.Time,
	region string,
) (*models.CompetitiveBenchmarkResponse, error) {
	// Fetch all responses for the main brand's campaigns
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}
	
	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch responses: %w", err)
	}
	
	// Filter to main brand's campaign responses
	var responses []*models.Response
	for _, resp := range allResponses {
		// Must match main brand
		if resp.Brand != mainBrand {
			continue
		}
		
		// Region filter: only apply if BOTH are specified
		// If response has no region, or request has no region filter, include it
		if region != "" && resp.Region != "" && !strings.EqualFold(resp.Region, region) {
			continue
		}
		
		// Filter by prompt IDs if specified
		if len(promptIDs) > 0 && !contains(promptIDs, resp.PromptID) {
			continue
		}
		
		// Filter by LLM IDs if specified
		if len(llmIDs) > 0 && !contains(llmIDs, resp.LLMID) {
			continue
		}
		
		responses = append(responses, resp)
	}
	
	if len(responses) == 0 {
		return nil, fmt.Errorf("no responses found for brand %s", mainBrand)
	}
	
	// If competitors not specified, auto-detect from responses
	if len(competitors) == 0 {
		competitorSet := make(map[string]bool)
		for _, resp := range responses {
			for _, comp := range resp.CompetitorsMention {
				// Normalize competitor name
				normalized := strings.TrimSpace(comp)
				if normalized != "" && !strings.EqualFold(normalized, mainBrand) {
					competitorSet[normalized] = true
				}
			}
		}
		for comp := range competitorSet {
			competitors = append(competitors, comp)
		}
	}
	
	// Analyze ALL brands mentioned across all responses
	// This gives us the real "share of voice" in AI responses
	allBrands := append([]string{mainBrand}, competitors...)
	brandStats := make(map[string]*brandMentionStats)
	
	for _, brand := range allBrands {
		brandStats[brand] = &brandMentionStats{
			brand:           brand,
			mentionCount:    0,
			totalVisibility: 0,
			totalPosition:   0,
			positionCount:   0,
			sentimentScores: []float64{},
		}
	}
	
	// Analyze each response
	for _, resp := range responses {
		// Main brand stats (from the actual analysis)
		if stats, ok := brandStats[mainBrand]; ok {
			if resp.BrandMentioned {
				stats.mentionCount++
				stats.totalVisibility += float64(resp.VisibilityScore)
				
				if resp.BrandPosition > 0 {
					stats.totalPosition += float64(resp.BrandPosition)
					stats.positionCount++
				}
				
				if resp.Sentiment != "" {
					stats.sentimentScores = append(stats.sentimentScores, calculateSentimentScore(resp.Sentiment))
				}
			}
		}
		
		// Competitor stats (from competitors_mention field)
		for _, mentioned := range resp.CompetitorsMention {
			// Check against all known competitors (case-insensitive)
			for compName, stats := range brandStats {
				if compName != mainBrand && strings.EqualFold(mentioned, compName) {
					stats.mentionCount++
					break
				}
			}
		}
	}
	
	// Build performance objects for all brands
	var allPerformances []models.BrandPerformance
	totalMentions := 0
	
	for _, brand := range allBrands {
		stats := brandStats[brand]
		totalMentions += stats.mentionCount
	}
	
	for _, brand := range allBrands {
		stats := brandStats[brand]
		
		perf := models.BrandPerformance{
			Brand:         brand,
			ResponseCount: stats.mentionCount,
		}
		
		// Calculate rates
		if len(responses) > 0 {
			perf.MentionRate = float64(stats.mentionCount) / float64(len(responses)) * 100
		}
		
		// Market share (share of total mentions)
		if totalMentions > 0 {
			perf.MarketSharePct = float64(stats.mentionCount) / float64(totalMentions) * 100
		}
		
		// Main brand gets additional metrics from actual analysis
		if brand == mainBrand && stats.mentionCount > 0 {
			perf.Visibility = stats.totalVisibility / float64(stats.mentionCount)
			
			if stats.positionCount > 0 {
				perf.AveragePosition = stats.totalPosition / float64(stats.positionCount)
			}
			
			if len(stats.sentimentScores) > 0 {
				sum := 0.0
				for _, s := range stats.sentimentScores {
					sum += s
				}
				perf.SentimentScore = sum / float64(len(stats.sentimentScores))
			}
		}
		
		allPerformances = append(allPerformances, perf)
	}
	
	// Sort by mention count to determine rankings
	sort.Slice(allPerformances, func(i, j int) bool {
		if allPerformances[i].ResponseCount != allPerformances[j].ResponseCount {
			return allPerformances[i].ResponseCount > allPerformances[j].ResponseCount
		}
		return allPerformances[i].MentionRate > allPerformances[j].MentionRate
	})
	
	// Find main brand and competitors
	var mainBrandPerf models.BrandPerformance
	var competitorPerfs []models.BrandPerformance
	mainBrandRank := 1
	
	for i, perf := range allPerformances {
		if perf.Brand == mainBrand {
			mainBrandPerf = perf
			mainBrandRank = i + 1
		} else {
			competitorPerfs = append(competitorPerfs, perf)
		}
	}
	
	// Market leader
	marketLeader := allPerformances[0].Brand
	
	// Generate prompt-level breakdown
	promptBreakdown := s.generatePromptBreakdown(responses, mainBrand, competitors)
	
	// Generate recommendations
	recommendations := s.recommendationsEngine.GenerateCompetitiveRecommendations(
		mainBrandPerf,
		competitorPerfs,
		marketLeader,
	)
	
	return &models.CompetitiveBenchmarkResponse{
		MainBrand:       mainBrandPerf,
		Competitors:     competitorPerfs,
		MarketLeader:    marketLeader,
		YourRank:        mainBrandRank,
		TotalBrands:     len(allPerformances),
		PromptBreakdown: promptBreakdown,
		Recommendations: recommendations,
		AnalyzedAt:      time.Now(),
	}, nil
}

// generatePromptBreakdown creates per-prompt competitive analysis
func (s *CompetitiveBenchmarkService) generatePromptBreakdown(
	responses []*models.Response,
	mainBrand string,
	competitors []string,
) []models.PromptCompetitiveAnalysis {
	var breakdown []models.PromptCompetitiveAnalysis
	
	for _, resp := range responses {
		// Main brand result
		mainResult := models.PromptBrandResult{
			Mentioned:       resp.BrandMentioned,
			VisibilityScore: resp.VisibilityScore,
			Position:        resp.BrandPosition,
			Sentiment:       resp.Sentiment,
			InSources:       resp.InGroundingSources,
		}
		
		// Competitor mentions
		var competitorMentions []models.PromptCompetitorMention
		mentionedBrands := make(map[string]bool)
		
		for _, mentioned := range resp.CompetitorsMention {
			mentionedBrands[strings.ToLower(mentioned)] = true
		}
		
		for _, comp := range competitors {
			isMentioned := mentionedBrands[strings.ToLower(comp)]
			competitorMentions = append(competitorMentions, models.PromptCompetitorMention{
				Brand:    comp,
				Mentioned: isMentioned,
			})
		}
		
		// Determine winner (brand with best position or visibility)
		winner := ""
		if resp.BrandMentioned {
			winner = mainBrand
			// If main brand has lower position (worse), a competitor might be winning
			if resp.BrandPosition > 1 {
				// Check if any competitors were mentioned (they might be ahead)
				for _, comp := range resp.CompetitorsMention {
					winner = comp // First competitor mentioned might be the leader
					break
				}
			}
		} else if len(resp.CompetitorsMention) > 0 {
			winner = resp.CompetitorsMention[0] // First mentioned competitor wins
		}
		
		// Get prompt type from database
		promptType := ""
		if prompt, err := s.db.GetPrompt(context.Background(), resp.PromptID); err == nil {
			promptType = string(prompt.PromptType)
		}
		
		// Count total brands mentioned
		totalBrands := 0
		if resp.BrandMentioned {
			totalBrands++
		}
		totalBrands += len(resp.CompetitorsMention)
		
		breakdown = append(breakdown, models.PromptCompetitiveAnalysis{
			PromptID:             resp.PromptID,
			PromptText:           resp.PromptText,
			PromptType:           promptType,
			MainBrandResult:      mainResult,
			CompetitorsMentioned: competitorMentions,
			Winner:               winner,
			TotalBrandsMentioned: totalBrands,
			ExecutedAt:           resp.CreatedAt,
		})
	}
	
	return breakdown
}

// brandMentionStats tracks mention statistics for a brand
type brandMentionStats struct {
	brand           string
	mentionCount    int
	totalVisibility float64
	totalPosition   float64
	positionCount   int
	sentimentScores []float64
}

// analyzeBrandPerformance analyzes performance for a single brand
func (s *CompetitiveBenchmarkService) analyzeBrandPerformance(
	ctx context.Context,
	brand string,
	promptIDs, llmIDs []string,
	startTime, endTime *time.Time,
	region string,
) (models.BrandPerformance, error) {
	// Fetch responses
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}
	
	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return models.BrandPerformance{}, err
	}
	
	// Filter responses
	var filteredResponses []*models.Response
	for _, resp := range allResponses {
		// Filter by brand
		if resp.Brand != brand {
			continue
		}
		
		// Filter by region if specified
		if region != "" && resp.Region != region {
			continue
		}
		
		// Filter by prompt IDs if specified
		if len(promptIDs) > 0 && !contains(promptIDs, resp.PromptID) {
			continue
		}
		
		// Filter by LLM IDs if specified
		if len(llmIDs) > 0 && !contains(llmIDs, resp.LLMID) {
			continue
		}
		
		filteredResponses = append(filteredResponses, resp)
	}
	
	if len(filteredResponses) == 0 {
		return models.BrandPerformance{
			Brand:         brand,
			ResponseCount: 0,
		}, nil
	}
	
	// Calculate metrics
	totalVisibility := 0.0
	mentionCount := 0
	groundingCount := 0
	totalPosition := 0.0
	positionCount := 0
	topPositionCount := 0
	sentimentSum := 0.0
	sentimentCount := 0
	
	for _, resp := range filteredResponses {
		totalVisibility += float64(resp.VisibilityScore)
		
		if resp.BrandMentioned {
			mentionCount++
		}
		
		if resp.InGroundingSources {
			groundingCount++
		}
		
		if resp.BrandPosition > 0 {
			totalPosition += float64(resp.BrandPosition)
			positionCount++
			
			if resp.BrandPosition <= 3 {
				topPositionCount++
			}
		}
		
		if resp.Sentiment != "" {
			sentimentSum += calculateSentimentScore(resp.Sentiment)
			sentimentCount++
		}
	}
	
	perf := models.BrandPerformance{
		Brand:         brand,
		Visibility:    totalVisibility / float64(len(filteredResponses)),
		MentionRate:   float64(mentionCount) / float64(len(filteredResponses)) * 100,
		GroundingRate: float64(groundingCount) / float64(len(filteredResponses)) * 100,
		ResponseCount: len(filteredResponses),
	}
	
	if positionCount > 0 {
		perf.AveragePosition = totalPosition / float64(positionCount)
		perf.TopPositionRate = float64(topPositionCount) / float64(positionCount) * 100
	}
	
	if sentimentCount > 0 {
		perf.SentimentScore = sentimentSum / float64(sentimentCount)
	}
	
	return perf, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


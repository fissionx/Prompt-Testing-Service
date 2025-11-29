package services

import (
	"context"
	"fmt"
	"sort"
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
func (s *CompetitiveBenchmarkService) GetCompetitiveBenchmark(
	ctx context.Context,
	mainBrand string,
	competitors []string,
	promptIDs, llmIDs []string,
	startTime, endTime *time.Time,
	region string,
) (*models.CompetitiveBenchmarkResponse, error) {
	// Collect all brands to analyze
	allBrands := append([]string{mainBrand}, competitors...)
	
	// Analyze each brand
	brandPerformances := make([]models.BrandPerformance, 0, len(allBrands))
	
	for _, brand := range allBrands {
		perf, err := s.analyzeBrandPerformance(ctx, brand, promptIDs, llmIDs, startTime, endTime, region)
		if err != nil {
			// Log error but continue with other brands
			fmt.Printf("Warning: failed to analyze brand %s: %v\n", brand, err)
			continue
		}
		brandPerformances = append(brandPerformances, perf)
	}
	
	if len(brandPerformances) == 0 {
		return nil, fmt.Errorf("no brand data available for analysis")
	}
	
	// Calculate market shares
	totalVisibility := 0.0
	for _, perf := range brandPerformances {
		totalVisibility += perf.Visibility * float64(perf.ResponseCount)
	}
	
	for i := range brandPerformances {
		if totalVisibility > 0 {
			brandPerformances[i].MarketSharePct = (brandPerformances[i].Visibility * float64(brandPerformances[i].ResponseCount) / totalVisibility) * 100
		}
	}
	
	// Sort by visibility to find market leader and rank
	sort.Slice(brandPerformances, func(i, j int) bool {
		return brandPerformances[i].Visibility > brandPerformances[j].Visibility
	})
	
	// Find main brand and separate competitors
	var mainBrandPerf models.BrandPerformance
	var competitorPerfs []models.BrandPerformance
	mainBrandRank := 0
	
	for i, perf := range brandPerformances {
		if perf.Brand == mainBrand {
			mainBrandPerf = perf
			mainBrandRank = i + 1
		} else {
			competitorPerfs = append(competitorPerfs, perf)
		}
	}
	
	// Market leader is the top ranked brand
	marketLeader := brandPerformances[0].Brand
	
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
		TotalBrands:     len(brandPerformances),
		Recommendations: recommendations,
		AnalyzedAt:      time.Now(),
	}, nil
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


package services

import (
	"context"
	"sort"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// SourceAnalyticsService provides source citation analytics
type SourceAnalyticsService struct {
	db                    db.Database
	recommendationsEngine *RecommendationsEngine
	logoService           *LogoService
}

// NewSourceAnalyticsService creates a new source analytics service
func NewSourceAnalyticsService(database db.Database) *SourceAnalyticsService {
	return &SourceAnalyticsService{
		db:                    database,
		recommendationsEngine: NewRecommendationsEngine(),
		logoService:           NewLogoService(database),
	}
}

// GetSourceAnalytics analyzes citation sources for a brand
func (s *SourceAnalyticsService) GetSourceAnalytics(
	ctx context.Context,
	brand string,
	startTime, endTime *time.Time,
	topN int,
) (*models.SourceAnalyticsResponse, error) {
	if topN <= 0 {
		topN = 20
	}
	
	// Fetch all responses for the brand
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}
	
	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	// Filter for brand
	var brandResponses []*models.Response
	for _, resp := range allResponses {
		if resp.Brand == brand {
			brandResponses = append(brandResponses, resp)
		}
	}
	
	// Aggregate source data
	domainStats := make(map[string]*sourceStats)
	totalCitations := 0
	brandInSources := false
	
	for _, resp := range brandResponses {
		// Count grounding domains
		for _, domain := range resp.GroundingDomains {
			if domain == "" {
				continue
			}
			
			if _, exists := domainStats[domain]; !exists {
				domainStats[domain] = &sourceStats{
					domain:       domain,
					llmBreakdown: make(map[string]int),
				}
			}
			
			domainStats[domain].citationCount++
			domainStats[domain].llmBreakdown[resp.LLMName]++
			totalCitations++
		}
		
		// Check if brand is in sources
		if resp.InGroundingSources {
			brandInSources = true
		}
	}
	
	// Convert to SourceInsight slice
	var sourceInsights []models.SourceInsight
	totalResponses := len(brandResponses)
	
	for domain, stats := range domainStats {
		insight := models.SourceInsight{
			Domain:        domain,
			CitationCount: stats.citationCount,
			MentionRate:   float64(stats.citationCount) / float64(totalResponses) * 100,
			LLMBreakdown:  stats.llmBreakdown,
			Categories:    categorizeSource(domain),
		}
		sourceInsights = append(sourceInsights, insight)
	}
	
	// Sort by citation count
	sort.Slice(sourceInsights, func(i, j int) bool {
		return sourceInsights[i].CitationCount > sourceInsights[j].CitationCount
	})
	
	// Limit to topN
	if len(sourceInsights) > topN {
		sourceInsights = sourceInsights[:topN]
	}
	
	// Generate recommendations
	recommendations := s.recommendationsEngine.GenerateSourceRecommendations(
		brand,
		sourceInsights,
		totalCitations,
		brandInSources,
	)
	
	// Determine period
	period := "all-time"
	if startTime != nil && endTime != nil {
		period = startTime.Format("2006-01-02") + " to " + endTime.Format("2006-01-02")
	}
	
	// Get brand logo
	brandLogo := s.logoService.GetBrandLogo(ctx, brand, "")
	
	return &models.SourceAnalyticsResponse{
		Brand:           brand,
		LogoURL:         brandLogo.LogoURL,
		FallbackLogoURL: brandLogo.FallbackLogoURL,
		Period:          period,
		TopSources:      sourceInsights,
		Recommendations: recommendations,
		TotalSources:    len(domainStats),
		TotalCitations:  totalCitations,
	}, nil
}

type sourceStats struct {
	domain        string
	citationCount int
	llmBreakdown  map[string]int
}


package services

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// DashboardService provides dashboard analytics and overview data
type DashboardService struct {
	db          db.Database
	logoService *LogoService
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(database db.Database) *DashboardService {
	return &DashboardService{
		db:          database,
		logoService: NewLogoService(database),
	}
}

// GetDashboardOverview returns a comprehensive dashboard overview
func (s *DashboardService) GetDashboardOverview(
	ctx context.Context,
	brandID string,
	orgID string,
	brand string,
	startTime, endTime *time.Time,
) (*models.DashboardOverviewResponse, error) {
	// Determine period
	period := "all-time"
	if startTime != nil && endTime != nil {
		period = startTime.Format("2006-01-02") + " to " + endTime.Format("2006-01-02")
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

	// Get brand logo
	brandLogo := s.logoService.GetBrandLogo(ctx, brandID, orgID, brand, "")

	response := &models.DashboardOverviewResponse{
		Brand:           brand,
		LogoURL:         brandLogo.LogoURL,
		FallbackLogoURL: brandLogo.FallbackLogoURL,
		Period:          period,
		TotalResponses:  len(brandResponses),
	}

	if len(brandResponses) == 0 {
		response.Visibility = models.DashboardMetric{Trend: "stable"}
		response.Sentiment = models.DashboardMetric{Trend: "stable"}
		response.Position = models.DashboardMetric{Trend: "stable"}
		response.GroundingRate = models.DashboardMetric{Trend: "stable"}
		// Still check for last run info even if no responses
		s.setLastRunInfo(ctx, brand, response)
		return response, nil
	}

	// Get competitors for ranking calculation
	competitors := s.getCompetitorsForRanking(ctx, brand, brandResponses)

	// Calculate metrics with rankings
	metrics := s.calculateMetricsWithRankings(ctx, brand, competitors, allResponses, startTime, endTime)
	response.Visibility = metrics.visibility
	response.Sentiment = metrics.sentiment
	response.Position = metrics.position
	response.GroundingRate = metrics.grounding

	// Count unique prompts and LLMs
	promptSet := make(map[string]bool)
	llmSet := make(map[string]bool)
	for _, resp := range brandResponses {
		promptSet[resp.PromptID] = true
		llmSet[resp.LLMID] = true
	}
	response.TotalPrompts = len(promptSet)
	response.TotalLLMs = len(llmSet)

	// Get trend data (last 7 days)
	response.TrendData = s.calculateTrendData(brandResponses)

	// Get top competitors
	// Use a map to ensure deterministic counting (order doesn't matter for counting)
	competitorCounts := make(map[string]int)
	for _, resp := range brandResponses {
		for _, comp := range resp.CompetitorsMention {
			if comp != "" {
				competitorCounts[comp]++
			}
		}
	}
	response.TopCompetitors = getTopCompetitors(competitorCounts, 5)

	// Get top citation sources - filter by OpenAI model only
	// Use a map to ensure deterministic counting (order doesn't matter for counting)
	sourceCounts := make(map[string]int)
	for _, resp := range brandResponses {
		// Only count citations from OpenAI models
		if resp.LLMProvider == "openai" {
			for _, domain := range resp.GroundingDomains {
				if domain != "" {
					sourceCounts[domain]++
				}
			}
		}
	}
	response.TopCitationSources = getTopCitationSources(sourceCounts, 5)

	// Get top performing prompts
	response.TopPerformingPrompts = s.getTopPerformingPrompts(brandResponses)

	// Get active campaigns count
	campaigns, _ := s.db.ListScheduledCampaigns(ctx, "active")
	response.ActiveCampaigns = len(campaigns)

	// Get last run date and status (check all responses, not just filtered ones)
	s.setLastRunInfo(ctx, brand, response)

	return response, nil
}

type metricsResult struct {
	visibility models.DashboardMetric
	sentiment  models.DashboardMetric
	position   models.DashboardMetric
	grounding  models.DashboardMetric
}

func (s *DashboardService) calculateMetrics(responses []*models.Response) metricsResult {
	var totalVisibility float64
	var mentionCount, groundingCount int
	var totalPosition float64
	var positionCount int
	var sentimentSum float64
	var sentimentCount int

	for _, resp := range responses {
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
		}
		if resp.Sentiment != "" {
			sentimentSum += calculateSentimentScore(resp.Sentiment)
			sentimentCount++
		}
	}

	total := float64(len(responses))

	result := metricsResult{
		visibility: models.DashboardMetric{
			Value: roundToTwo(totalVisibility / total),
			Trend: "stable",
		},
		sentiment: models.DashboardMetric{
			Trend: "stable",
		},
		position: models.DashboardMetric{
			Trend: "stable",
		},
		grounding: models.DashboardMetric{
			Value: roundToTwo(float64(groundingCount) / total * 100),
			Trend: "stable",
		},
	}

	result.visibility.Value = roundToTwo(float64(mentionCount) / total * 100)

	if sentimentCount > 0 {
		// Convert from -1 to 1 scale to 0 to 100 scale
		result.sentiment.Value = roundToTwo((sentimentSum/float64(sentimentCount) + 1) * 50)
	}

	if positionCount > 0 {
		result.position.Value = roundToTwo(totalPosition / float64(positionCount))
	}

	return result
}

// getCompetitorsForRanking gets competitors for ranking calculation
// First tries saved competitors, then falls back to auto-detection from responses
func (s *DashboardService) getCompetitorsForRanking(ctx context.Context, brand string, brandResponses []*models.Response) []string {
	// Try to get saved competitors first
	savedCompetitors, err := s.db.GetBrandCompetitors(ctx, brand)
	if err == nil && savedCompetitors != nil && len(savedCompetitors.Competitors) > 0 {
		parsedCompetitors, _ := parseCompetitorStrings(savedCompetitors.Competitors)
		return parsedCompetitors
	}

	// Fall back to auto-detection from responses
	competitorSet := make(map[string]bool)
	for _, resp := range brandResponses {
		for _, comp := range resp.CompetitorsMention {
			normalized := strings.TrimSpace(comp)
			if normalized != "" && !strings.EqualFold(normalized, brand) {
				competitorSet[normalized] = true
			}
		}
	}

	competitors := make([]string, 0, len(competitorSet))
	for comp := range competitorSet {
		competitors = append(competitors, comp)
	}

	return competitors
}

// brandMetrics represents metrics for a single brand
type brandMetrics struct {
	brand         string
	visibility    float64
	sentiment     float64
	position      float64
	groundingRate float64
}

// calculateMetricsWithRankings calculates metrics for the main brand and determines rankings
func (s *DashboardService) calculateMetricsWithRankings(
	ctx context.Context,
	mainBrand string,
	competitors []string,
	allResponses []*models.Response,
	startTime, endTime *time.Time,
) metricsResult {
	// Calculate base metrics for main brand
	mainBrandResponses := make([]*models.Response, 0)
	for _, resp := range allResponses {
		if resp.Brand == mainBrand {
			mainBrandResponses = append(mainBrandResponses, resp)
		}
	}

	baseMetrics := s.calculateMetrics(mainBrandResponses)

	// If no competitors, return base metrics without rankings
	if len(competitors) == 0 {
		return baseMetrics
	}

	// Calculate metrics for all brands (main brand + competitors)
	allBrands := append([]string{mainBrand}, competitors...)
	brandMetricsMap := make(map[string]*brandMetrics)

	// Initialize metrics for all brands
	for _, brand := range allBrands {
		brandMetricsMap[brand] = &brandMetrics{brand: brand}
	}

	// Filter responses by time if needed
	var filteredResponses []*models.Response
	for _, resp := range allResponses {
		// Check if response matches time filter
		if startTime != nil && resp.CreatedAt.Before(*startTime) {
			continue
		}
		if endTime != nil && resp.CreatedAt.After(*endTime) {
			continue
		}
		filteredResponses = append(filteredResponses, resp)
	}

	// Aggregate metrics for each brand
	// We'll use main brand's responses as the baseline for comparison
	brandResponseCounts := make(map[string]int)
	brandMentionCounts := make(map[string]int)
	brandGroundingCounts := make(map[string]int)
	brandPositionSums := make(map[string]float64)
	brandPositionCounts := make(map[string]int)
	brandSentimentSums := make(map[string]float64)
	brandSentimentCounts := make(map[string]int)

	// Track which brands have full response data (not just mentions)
	brandsWithFullData := make(map[string]bool)

	// First, collect metrics from actual responses (main brand and competitors if they have responses)
	for _, resp := range filteredResponses {
		if _, exists := brandMetricsMap[resp.Brand]; exists {
			brandResponseCounts[resp.Brand]++
			brandsWithFullData[resp.Brand] = true

			if resp.BrandMentioned {
				brandMentionCounts[resp.Brand]++
			}
			if resp.InGroundingSources {
				brandGroundingCounts[resp.Brand]++
			}
			if resp.BrandPosition > 0 {
				brandPositionSums[resp.Brand] += float64(resp.BrandPosition)
				brandPositionCounts[resp.Brand]++
			}
			if resp.Sentiment != "" {
				brandSentimentSums[resp.Brand] += calculateSentimentScore(resp.Sentiment)
				brandSentimentCounts[resp.Brand]++
			}
		}
	}

	// For visibility ranking, count mentions of competitors in main brand's responses
	// This gives us a fair comparison: how often each brand appears in the same set of responses
	// Count filtered main brand responses
	filteredMainBrandResponses := 0
	for _, resp := range filteredResponses {
		if resp.Brand == mainBrand {
			filteredMainBrandResponses++
		}
	}
	totalResponsesForComparison := filteredMainBrandResponses
	if totalResponsesForComparison == 0 {
		totalResponsesForComparison = len(filteredResponses)
	}

	for _, resp := range filteredResponses {
		// Count competitor mentions for visibility calculation
		for _, comp := range resp.CompetitorsMention {
			if _, exists := brandMetricsMap[comp]; exists {
				// Count mention
				brandMentionCounts[comp]++
				// Count this as a response where competitor could appear (for visibility calculation)
				if brandResponseCounts[comp] == 0 {
					// Initialize if not already set from actual responses
					brandResponseCounts[comp] = 0
				}
			}
		}
	}

	// Set response counts for competitors based on total responses (for fair visibility comparison)
	for _, comp := range competitors {
		if !brandsWithFullData[comp] {
			// Competitor doesn't have its own responses, use total responses for comparison
			brandResponseCounts[comp] = totalResponsesForComparison
		}
	}

	// Calculate metrics for each brand
	var allBrandMetrics []*brandMetrics
	for _, brand := range allBrands {
		bm := brandMetricsMap[brand]
		total := float64(brandResponseCounts[brand])

		if total > 0 {
			bm.visibility = roundToTwo(float64(brandMentionCounts[brand]) / total * 100)
			bm.groundingRate = roundToTwo(float64(brandGroundingCounts[brand]) / total * 100)

			if brandPositionCounts[brand] > 0 {
				bm.position = roundToTwo(brandPositionSums[brand] / float64(brandPositionCounts[brand]))
			}

			if brandSentimentCounts[brand] > 0 {
				// Convert from -1 to 1 scale to 0 to 100 scale
				bm.sentiment = roundToTwo((brandSentimentSums[brand]/float64(brandSentimentCounts[brand]) + 1) * 50)
			}
		}

		allBrandMetrics = append(allBrandMetrics, bm)
	}

	// Calculate rankings for each metric
	totalBrands := len(allBrandMetrics)

	// Visibility ranking (higher is better) - we can rank all brands
	sort.Slice(allBrandMetrics, func(i, j int) bool {
		return allBrandMetrics[i].visibility > allBrandMetrics[j].visibility
	})
	visibilityRank := s.findRank(allBrandMetrics, mainBrand, func(bm *brandMetrics) float64 { return bm.visibility }, true)

	// For other metrics, only rank if we have full data for at least 2 brands
	brandsWithSentimentData := 0
	brandsWithPositionData := 0
	brandsWithGroundingData := 0

	for _, bm := range allBrandMetrics {
		if bm.sentiment > 0 {
			brandsWithSentimentData++
		}
		if bm.position > 0 {
			brandsWithPositionData++
		}
		if bm.groundingRate > 0 {
			brandsWithGroundingData++
		}
	}

	// Sentiment ranking (higher is better) - only if we have data for multiple brands
	var sentimentRank int
	if brandsWithSentimentData >= 2 {
		sort.Slice(allBrandMetrics, func(i, j int) bool {
			return allBrandMetrics[i].sentiment > allBrandMetrics[j].sentiment
		})
		sentimentRank = s.findRank(allBrandMetrics, mainBrand, func(bm *brandMetrics) float64 { return bm.sentiment }, true)
	}

	// Position ranking (lower is better) - only if we have data for multiple brands
	var positionRank int
	if brandsWithPositionData >= 2 {
		sort.Slice(allBrandMetrics, func(i, j int) bool {
			// Handle zero values - put them at the end
			if allBrandMetrics[i].position == 0 && allBrandMetrics[j].position == 0 {
				return false
			}
			if allBrandMetrics[i].position == 0 {
				return false
			}
			if allBrandMetrics[j].position == 0 {
				return true
			}
			return allBrandMetrics[i].position < allBrandMetrics[j].position
		})
		positionRank = s.findRank(allBrandMetrics, mainBrand, func(bm *brandMetrics) float64 { return bm.position }, false)
	}

	// Grounding rate ranking (higher is better) - only if we have data for multiple brands
	var groundingRank int
	if brandsWithGroundingData >= 2 {
		sort.Slice(allBrandMetrics, func(i, j int) bool {
			return allBrandMetrics[i].groundingRate > allBrandMetrics[j].groundingRate
		})
		groundingRank = s.findRank(allBrandMetrics, mainBrand, func(bm *brandMetrics) float64 { return bm.groundingRate }, true)
	}

	// Update base metrics with rankings
	baseMetrics.visibility.Rank = visibilityRank
	baseMetrics.visibility.TotalBrands = totalBrands
	baseMetrics.sentiment.Rank = sentimentRank
	baseMetrics.sentiment.TotalBrands = totalBrands
	baseMetrics.position.Rank = positionRank
	baseMetrics.position.TotalBrands = totalBrands
	baseMetrics.grounding.Rank = groundingRank
	baseMetrics.grounding.TotalBrands = totalBrands

	return baseMetrics
}

// findRank finds the rank of the main brand in a sorted list
// higherIsBetter: true if higher values are better (visibility, sentiment), false if lower is better (position)
func (s *DashboardService) findRank(
	brandMetrics []*brandMetrics,
	mainBrand string,
	getValue func(*brandMetrics) float64,
	higherIsBetter bool,
) int {
	for i, bm := range brandMetrics {
		if bm.brand == mainBrand {
			// Check for ties - if multiple brands have the same value, they share the same rank
			mainBrandValue := getValue(bm)
			rank := i + 1

			// Adjust for ties (count how many brands before this one have the same value)
			for j := i - 1; j >= 0; j-- {
				if getValue(brandMetrics[j]) == mainBrandValue {
					rank = j + 1
				} else {
					break
				}
			}

			return rank
		}
	}
	return 0
}

func (s *DashboardService) calculateTrendData(responses []*models.Response) []models.TrendDataPoint {
	// Group by date
	dailyData := make(map[string]*dailyMetrics)

	for _, resp := range responses {
		date := resp.CreatedAt.Format("2006-01-02")
		if _, exists := dailyData[date]; !exists {
			dailyData[date] = &dailyMetrics{}
		}

		dm := dailyData[date]
		dm.count++
		dm.totalVisibility += float64(resp.VisibilityScore)
		if resp.BrandMentioned {
			dm.mentionCount++
		}
		if resp.BrandPosition > 0 {
			dm.totalPosition += float64(resp.BrandPosition)
			dm.positionCount++
		}
		if resp.Sentiment != "" {
			dm.sentimentSum += calculateSentimentScore(resp.Sentiment)
			dm.sentimentCount++
		}
	}

	// Convert to trend points
	var trendData []models.TrendDataPoint
	for date, dm := range dailyData {
		point := models.TrendDataPoint{
			Date:       date,
			Visibility: roundToTwo(float64(dm.mentionCount) / float64(dm.count) * 100),
		}
		if dm.sentimentCount > 0 {
			point.Sentiment = roundToTwo((dm.sentimentSum/float64(dm.sentimentCount) + 1) * 50)
		}
		if dm.positionCount > 0 {
			point.Position = roundToTwo(dm.totalPosition / float64(dm.positionCount))
		}
		trendData = append(trendData, point)
	}

	// Sort by date and limit to last 7
	sort.Slice(trendData, func(i, j int) bool {
		return trendData[i].Date < trendData[j].Date
	})

	if len(trendData) > 7 {
		trendData = trendData[len(trendData)-7:]
	}

	return trendData
}

type dailyMetrics struct {
	count           int
	mentionCount    int
	totalVisibility float64
	totalPosition   float64
	positionCount   int
	sentimentSum    float64
	sentimentCount  int
}

// GetModelAnalytics returns analytics broken down by AI model
func (s *DashboardService) GetModelAnalytics(
	ctx context.Context,
	brandID string,
	orgID string,
	brand string,
	startTime, endTime *time.Time,
) (*models.ModelAnalyticsResponse, error) {
	period := "all-time"
	if startTime != nil && endTime != nil {
		period = startTime.Format("2006-01-02") + " to " + endTime.Format("2006-01-02")
	}

	// Fetch responses
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter for brand and group by model
	modelStats := make(map[string]*modelAggregator)
	for _, resp := range allResponses {
		if resp.Brand != brand {
			continue
		}

		key := resp.LLMID
		if _, exists := modelStats[key]; !exists {
			modelStats[key] = &modelAggregator{
				id:       resp.LLMID,
				name:     resp.LLMName,
				provider: resp.LLMProvider,
			}
		}

		ms := modelStats[key]
		ms.count++
		ms.totalVisibility += float64(resp.VisibilityScore)
		if resp.BrandMentioned {
			ms.mentionCount++
		}
		if resp.BrandPosition > 0 {
			ms.totalPosition += float64(resp.BrandPosition)
			ms.positionCount++
			if resp.BrandPosition <= 3 {
				ms.topPositionCount++
			}
		}
		if resp.InGroundingSources {
			ms.groundingCount++
		}
		if resp.Sentiment != "" {
			ms.sentimentSum += calculateSentimentScore(resp.Sentiment)
			ms.sentimentCount++
		}
		ms.totalLatency += resp.LatencyMs
	}

	// Get brand logo
	brandLogo := s.logoService.GetBrandLogo(ctx, brandID, orgID, brand, "")

	response := &models.ModelAnalyticsResponse{
		Brand:           brand,
		LogoURL:         brandLogo.LogoURL,
		FallbackLogoURL: brandLogo.FallbackLogoURL,
		Period:          period,
	}

	// Convert to model performance
	var bestScore, worstScore float64 = -1, 101
	for _, ms := range modelStats {
		if ms.count == 0 {
			continue
		}

		perf := models.ModelPerformance{
			ModelID:       ms.id,
			ModelName:     ms.name,
			Provider:      ms.provider,
			ResponseCount: ms.count,
			Visibility:    roundToTwo(ms.totalVisibility / float64(ms.count)),
			MentionRate:   roundToTwo(float64(ms.mentionCount) / float64(ms.count) * 100),
			GroundingRate: roundToTwo(float64(ms.groundingCount) / float64(ms.count) * 100),
			AvgLatencyMs:  ms.totalLatency / int64(ms.count),
		}

		if ms.positionCount > 0 {
			perf.AvgPosition = roundToTwo(ms.totalPosition / float64(ms.positionCount))
			perf.TopPositionPct = roundToTwo(float64(ms.topPositionCount) / float64(ms.positionCount) * 100)
		}

		if ms.sentimentCount > 0 {
			perf.SentimentScore = roundToTwo((ms.sentimentSum/float64(ms.sentimentCount) + 1) * 50)
		}

		// Calculate composite score for best/worst
		score := perf.MentionRate*0.4 + perf.Visibility*0.3 + perf.SentimentScore*0.3
		if score > bestScore {
			bestScore = score
			response.BestModel = ms.name
		}
		if score < worstScore {
			worstScore = score
			response.WorstModel = ms.name
		}

		response.Models = append(response.Models, perf)
	}

	// Sort by mention rate
	sort.Slice(response.Models, func(i, j int) bool {
		return response.Models[i].MentionRate > response.Models[j].MentionRate
	})

	return response, nil
}

type modelAggregator struct {
	id               string
	name             string
	provider         string
	count            int
	mentionCount     int
	totalVisibility  float64
	totalPosition    float64
	positionCount    int
	topPositionCount int
	groundingCount   int
	sentimentSum     float64
	sentimentCount   int
	totalLatency     int64
}

// GetCompetitorMatrix returns the visibility vs sentiment quadrant matrix
func (s *DashboardService) GetCompetitorMatrix(
	ctx context.Context,
	mainBrand string,
	competitors []string,
	startTime, endTime *time.Time,
	competitorMap map[string]string, // name -> domain mapping
) (*models.CompetitorMatrixResponse, error) {
	period := "all-time"
	if startTime != nil && endTime != nil {
		period = startTime.Format("2006-01-02") + " to " + endTime.Format("2006-01-02")
	}

	// Fetch responses for main brand
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter main brand responses
	var mainBrandResponses []*models.Response
	for _, resp := range allResponses {
		if resp.Brand == mainBrand {
			mainBrandResponses = append(mainBrandResponses, resp)
		}
	}

	// Auto-detect competitors if not provided
	if len(competitors) == 0 {
		// Try to get saved competitors first
		savedCompetitors, err := s.db.GetBrandCompetitors(ctx, mainBrand)
		if err == nil && savedCompetitors != nil && len(savedCompetitors.Competitors) > 0 {
			// Use saved competitors - parse "name|domain" format
			parsedCompetitors, parsedMap := parseCompetitorStrings(savedCompetitors.Competitors)
			competitors = parsedCompetitors
			// Merge parsed domains into competitorMap
			if competitorMap == nil {
				competitorMap = make(map[string]string)
			}
			for name, domain := range parsedMap {
				competitorMap[name] = domain
			}
		} else {
			// Fall back to auto-detection from responses
			competitorSet := make(map[string]bool)
			for _, resp := range mainBrandResponses {
				for _, comp := range resp.CompetitorsMention {
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
	} else {
		// Parse competitors if they're in "name|domain" format
		parsedCompetitors, parsedMap := parseCompetitorStrings(competitors)
		competitors = parsedCompetitors
		// Merge parsed domains into competitorMap
		if competitorMap == nil {
			competitorMap = make(map[string]string)
		}
		for name, domain := range parsedMap {
			competitorMap[name] = domain
		}
	}

	// Calculate metrics for all brands
	allBrands := append([]string{mainBrand}, competitors...)
	brandStats := make(map[string]*brandMatrixStats)

	for _, brand := range allBrands {
		brandStats[brand] = &brandMatrixStats{brand: brand}
	}

	// Aggregate main brand stats
	for _, resp := range mainBrandResponses {
		if stats, ok := brandStats[mainBrand]; ok {
			stats.count++
			if resp.BrandMentioned {
				stats.mentionCount++
				if resp.Sentiment != "" {
					stats.sentimentSum += calculateSentimentScore(resp.Sentiment)
					stats.sentimentCount++
				}
				if resp.BrandPosition > 0 {
					stats.totalPosition += float64(resp.BrandPosition)
					stats.positionCount++
				}
			}
		}

		// Count competitor mentions
		for _, comp := range resp.CompetitorsMention {
			if stats, ok := brandStats[comp]; ok {
				stats.mentionCount++
			}
		}
	}

	// Build matrix brands
	var matrixBrands []models.CompetitorMatrixBrand
	var allVisibilities, allSentiments []float64

	// Get logos for all brands
	logoRequests := make([]BrandLogoRequest, 0, len(allBrands))
	for _, brand := range allBrands {
		logoRequests = append(logoRequests, BrandLogoRequest{Name: brand})
	}
	brandLogos := s.logoService.GetMultipleLogos(ctx, logoRequests)
	logoMap := make(map[string]models.BrandWithLogo)
	for _, logo := range brandLogos {
		logoMap[logo.Brand] = logo
	}

	for brand, stats := range brandStats {
		if stats.count == 0 && brand == mainBrand {
			stats.count = len(mainBrandResponses)
		}

		if stats.count == 0 {
			continue
		}

		visibility := float64(stats.mentionCount) / float64(len(mainBrandResponses)) * 100
		sentiment := 50.0 // Default neutral
		if stats.sentimentCount > 0 {
			sentiment = (stats.sentimentSum/float64(stats.sentimentCount) + 1) * 50
		}

		avgPosition := 0.0
		if stats.positionCount > 0 {
			avgPosition = stats.totalPosition / float64(stats.positionCount)
		}

		logo := logoMap[brand]

		// Get domain from map or derive it
		domain := ""
		if competitorMap != nil {
			if d, ok := competitorMap[brand]; ok {
				domain = d
			}
		}
		if domain == "" {
			domain = deriveCompetitorDomainFromName(brand)
		}

		matrixBrand := models.CompetitorMatrixBrand{
			Brand:           brand,
			Domain:          domain,
			LogoURL:         logo.LogoURL,
			FallbackLogoURL: logo.FallbackLogoURL,
			Visibility:      roundToTwo(visibility),
			Sentiment:       roundToTwo(sentiment),
			Position:        roundToTwo(avgPosition),
			ResponseCount:   stats.mentionCount,
			IsMainBrand:     brand == mainBrand,
		}

		allVisibilities = append(allVisibilities, visibility)
		allSentiments = append(allSentiments, sentiment)
		matrixBrands = append(matrixBrands, matrixBrand)
	}

	// Calculate medians for quadrant division
	visibilityMedian := calculateMedian(allVisibilities)
	sentimentMedian := calculateMedian(allSentiments)

	// Assign quadrants
	quadrants := models.CompetitorQuadrants{}
	for i := range matrixBrands {
		brand := &matrixBrands[i]
		if brand.Visibility >= visibilityMedian && brand.Sentiment >= sentimentMedian {
			brand.Quadrant = "leader"
			quadrants.Leaders = append(quadrants.Leaders, brand.Brand)
		} else if brand.Visibility < visibilityMedian && brand.Sentiment >= sentimentMedian {
			brand.Quadrant = "niche_player"
			quadrants.NichePlayers = append(quadrants.NichePlayers, brand.Brand)
		} else if brand.Visibility < visibilityMedian && brand.Sentiment < sentimentMedian {
			brand.Quadrant = "lagger"
			quadrants.Laggers = append(quadrants.Laggers, brand.Brand)
		} else {
			brand.Quadrant = "controversial"
			quadrants.Controversial = append(quadrants.Controversial, brand.Brand)
		}
	}

	return &models.CompetitorMatrixResponse{
		MainBrand: mainBrand,
		Period:    period,
		Quadrants: quadrants,
		Brands:    matrixBrands,
		AxisInfo: models.MatrixAxisInfo{
			VisibilityMedian: roundToTwo(visibilityMedian),
			SentimentMedian:  roundToTwo(sentimentMedian),
			VisibilityMax:    100,
			SentimentMax:     100,
		},
	}, nil
}

type brandMatrixStats struct {
	brand          string
	count          int
	mentionCount   int
	sentimentSum   float64
	sentimentCount int
	totalPosition  float64
	positionCount  int
}

// GetTrendComparison returns trend data comparing multiple brands
func (s *DashboardService) GetTrendComparison(
	ctx context.Context,
	mainBrand string,
	competitors []string,
	metric string,
	startTime, endTime *time.Time,
	granularity string,
	competitorMap map[string]string, // name -> domain mapping
) (*models.TrendComparisonResponse, error) {
	if granularity == "" {
		granularity = "daily"
	}
	if metric == "" {
		metric = "visibility"
	}

	period := "all-time"
	if startTime != nil && endTime != nil {
		period = startTime.Format("2006-01-02") + " to " + endTime.Format("2006-01-02")
	}

	// Fetch responses
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter main brand responses
	var mainBrandResponses []*models.Response
	for _, resp := range allResponses {
		if resp.Brand == mainBrand {
			mainBrandResponses = append(mainBrandResponses, resp)
		}
	}

	// Auto-detect competitors if not provided
	if len(competitors) == 0 {
		// Try to get saved competitors first
		savedCompetitors, err := s.db.GetBrandCompetitors(ctx, mainBrand)
		if err == nil && savedCompetitors != nil && len(savedCompetitors.Competitors) > 0 {
			// Use saved competitors - parse "name|domain" format
			parsedCompetitors, parsedMap := parseCompetitorStrings(savedCompetitors.Competitors)
			competitors = parsedCompetitors
			// Merge parsed domains into competitorMap
			if competitorMap == nil {
				competitorMap = make(map[string]string)
			}
			for name, domain := range parsedMap {
				competitorMap[name] = domain
			}
		} else {
			// Fall back to auto-detection from responses
			competitorSet := make(map[string]bool)
			for _, resp := range mainBrandResponses {
				for _, comp := range resp.CompetitorsMention {
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
		// Limit to top 5 competitors
		if len(competitors) > 5 {
			competitors = competitors[:5]
		}
	} else {
		// Parse competitors if they're in "name|domain" format
		parsedCompetitors, parsedMap := parseCompetitorStrings(competitors)
		competitors = parsedCompetitors
		// Merge parsed domains into competitorMap
		if competitorMap == nil {
			competitorMap = make(map[string]string)
		}
		for name, domain := range parsedMap {
			competitorMap[name] = domain
		}
	}

	allBrands := append([]string{mainBrand}, competitors...)

	// Get logos
	logoRequests := make([]BrandLogoRequest, 0, len(allBrands))
	for _, brand := range allBrands {
		logoRequests = append(logoRequests, BrandLogoRequest{Name: brand})
	}
	brandLogos := s.logoService.GetMultipleLogos(ctx, logoRequests)
	logoMap := make(map[string]models.BrandWithLogo)
	for _, logo := range brandLogos {
		logoMap[logo.Brand] = logo
	}

	// Group data by date and brand
	brandDateData := make(map[string]map[string]*trendAggregator)
	dateSet := make(map[string]bool)

	for _, brand := range allBrands {
		brandDateData[brand] = make(map[string]*trendAggregator)
	}

	// Aggregate main brand data
	for _, resp := range mainBrandResponses {
		var date string
		switch granularity {
		case "daily":
			date = resp.CreatedAt.Format("2006-01-02")
		case "weekly":
			date = s.formatISOWeek(resp.CreatedAt)
		case "monthly":
			date = resp.CreatedAt.Format("2006-01")
		default:
			date = resp.CreatedAt.Format("2006-01-02")
		}
		dateSet[date] = true

		// Main brand
		if _, exists := brandDateData[mainBrand][date]; !exists {
			brandDateData[mainBrand][date] = &trendAggregator{}
		}
		agg := brandDateData[mainBrand][date]
		agg.count++
		if resp.BrandMentioned {
			agg.mentionCount++
			if resp.Sentiment != "" {
				agg.sentimentSum += calculateSentimentScore(resp.Sentiment)
				agg.sentimentCount++
			}
			if resp.BrandPosition > 0 {
				agg.totalPosition += float64(resp.BrandPosition)
				agg.positionCount++
			}
		}

		// Competitors
		for _, comp := range resp.CompetitorsMention {
			if _, ok := brandDateData[comp]; ok {
				if _, exists := brandDateData[comp][date]; !exists {
					brandDateData[comp][date] = &trendAggregator{}
				}
				brandDateData[comp][date].mentionCount++
			}
		}
	}

	// Sort dates
	var dates []string
	for date := range dateSet {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Build trend data
	var trends []models.BrandTrendData
	for _, brand := range allBrands {
		logo := logoMap[brand]

		// Get domain from map or derive it
		domain := ""
		if competitorMap != nil {
			if d, ok := competitorMap[brand]; ok {
				domain = d
			}
		}
		if domain == "" {
			domain = deriveCompetitorDomainFromName(brand)
		}

		trend := models.BrandTrendData{
			Brand:           brand,
			Domain:          domain,
			LogoURL:         logo.LogoURL,
			FallbackLogoURL: logo.FallbackLogoURL,
			IsMainBrand:     brand == mainBrand,
		}

		var trendValues []models.TrendValue
		for _, date := range dates {
			agg := brandDateData[brand][date]
			var value float64

			if agg != nil && agg.count > 0 {
				switch metric {
				case "visibility":
					value = float64(agg.mentionCount) / float64(brandDateData[mainBrand][date].count) * 100
				case "sentiment":
					if agg.sentimentCount > 0 {
						value = (agg.sentimentSum/float64(agg.sentimentCount) + 1) * 50
					}
				case "position":
					if agg.positionCount > 0 {
						value = agg.totalPosition / float64(agg.positionCount)
					}
				}
			}

			// Convert date to appropriate format based on granularity
			displayDate := s.convertDateForDisplay(date, granularity)

			trendValues = append(trendValues, models.TrendValue{
				Value: roundToTwo(value),
				Date:  displayDate,
			})
		}

		trend.Values = trendValues
		if len(trendValues) > 0 {
			trend.CurrentValue = trendValues[len(trendValues)-1].Value
			if len(trendValues) > 1 && trendValues[0].Value > 0 {
				trend.Change = roundToTwo((trendValues[len(trendValues)-1].Value - trendValues[0].Value) / trendValues[0].Value * 100)
			}
		}

		trends = append(trends, trend)
	}

	// Convert dates to display format for the Dates array
	displayDates := make([]string, len(dates))
	for i, date := range dates {
		displayDates[i] = s.convertDateForDisplay(date, granularity)
	}

	return &models.TrendComparisonResponse{
		MainBrand:   mainBrand,
		Metric:      metric,
		Period:      period,
		Granularity: granularity,
		Trends:      trends,
		Dates:       displayDates,
	}, nil
}

type trendAggregator struct {
	count          int
	mentionCount   int
	sentimentSum   float64
	sentimentCount int
	totalPosition  float64
	positionCount  int
}

// Helper functions
func getTopKeys(m map[string]int, n int) []string {
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	var result []string
	for i := 0; i < n && i < len(sorted); i++ {
		result = append(result, sorted[i].Key)
	}
	return result
}

// getTopCompetitors returns top competitors with name and domain
func getTopCompetitors(m map[string]int, n int) []models.TopCompetitor {
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	// Sort by count (descending), then by name (ascending) for deterministic ordering
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Value != sorted[j].Value {
			return sorted[i].Value > sorted[j].Value
		}
		// Secondary sort: alphabetical by name for ties
		return sorted[i].Key < sorted[j].Key
	})

	var result []models.TopCompetitor
	for i := 0; i < n && i < len(sorted); i++ {
		competitorName := sorted[i].Key
		domain := deriveCompetitorDomain(competitorName)
		result = append(result, models.TopCompetitor{
			Name:   competitorName,
			Domain: domain,
		})
	}
	return result
}

// getTopCitationSources returns top citation sources with name and domain
func getTopCitationSources(m map[string]int, n int) []models.TopCitationSource {
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	// Sort by count (descending), then by domain (ascending) for deterministic ordering
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Value != sorted[j].Value {
			return sorted[i].Value > sorted[j].Value
		}
		// Secondary sort: alphabetical by domain for ties
		return sorted[i].Key < sorted[j].Key
	})

	var result []models.TopCitationSource
	for i := 0; i < n && i < len(sorted); i++ {
		domain := sorted[i].Key
		name := extractSourceName(domain)
		normalizedDomain := normalizeCitationDomain(domain)
		result = append(result, models.TopCitationSource{
			Name:   name,
			Domain: normalizedDomain,
		})
	}
	return result
}

// deriveCompetitorDomain derives a domain from a competitor name
// Example: "windsurf" -> "www.windsurf.com", "Windsurf (by Codeium)" -> "www.windsurf.com"
func deriveCompetitorDomain(competitorName string) string {
	// If it already looks like a domain, normalize and return
	if strings.Contains(competitorName, ".") {
		normalized := strings.ToLower(strings.TrimSpace(competitorName))
		// Remove protocol if present
		normalized = strings.TrimPrefix(normalized, "http://")
		normalized = strings.TrimPrefix(normalized, "https://")
		// Remove path if present
		if idx := strings.Index(normalized, "/"); idx != -1 {
			normalized = normalized[:idx]
		}
		// Add www. if not present
		if !strings.HasPrefix(normalized, "www.") {
			return "www." + normalized
		}
		return normalized
	}

	// Clean the competitor name to extract core brand name
	cleaned := cleanCompetitorName(competitorName)

	// Convert to lowercase and remove spaces
	normalized := strings.ToLower(strings.TrimSpace(cleaned))

	// Check if the cleaned name already looks like a domain (e.g., from special cases)
	if strings.Contains(normalized, ".") {
		// It's already a domain-like string, add www. prefix and .com suffix if needed
		if !strings.HasPrefix(normalized, "www.") {
			normalized = "www." + normalized
		}
		// Add .com if it doesn't already have a TLD
		if !strings.HasSuffix(normalized, ".com") && !strings.HasSuffix(normalized, ".org") &&
			!strings.HasSuffix(normalized, ".net") && !strings.HasSuffix(normalized, ".io") &&
			!strings.HasSuffix(normalized, ".ai") && !strings.HasSuffix(normalized, ".co") {
			normalized = normalized + ".com"
		}
		return normalized
	}

	// Remove spaces for single-word or multi-word names
	normalized = strings.ReplaceAll(normalized, " ", "")

	// Remove any remaining invalid characters for domain names
	normalized = sanitizeDomainName(normalized)

	// Construct www.{name}.com
	return "www." + normalized + ".com"
}

// cleanCompetitorName removes parentheses, special characters, and extracts core brand name
func cleanCompetitorName(name string) string {
	// Remove content in parentheses (e.g., "Windsurf (by Codeium)" -> "Windsurf")
	cleaned := name
	for {
		openIdx := strings.Index(cleaned, "(")
		if openIdx == -1 {
			break
		}
		closeIdx := strings.Index(cleaned[openIdx:], ")")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx
		cleaned = cleaned[:openIdx] + cleaned[closeIdx+1:]
	}

	// Remove common prefixes/suffixes that might not be part of the domain
	cleaned = strings.TrimSpace(cleaned)

	// Remove common suffixes like " (VS Code)", " - ", etc.
	cleaned = strings.Split(cleaned, " - ")[0]
	cleaned = strings.Split(cleaned, " | ")[0]
	cleaned = strings.TrimSpace(cleaned)

	// For multi-word names, try to extract the main brand name
	// Special handling for known cases
	cleaned = handleSpecialBrandNames(cleaned)

	// If it's a special case that returned a domain-like string, return as-is
	if strings.Contains(cleaned, ".") {
		return cleaned
	}

	// For simple names, extract the first word
	// This handles cases like "Windsurf (by Codeium)" -> "Windsurf"
	words := strings.Fields(cleaned)
	if len(words) >= 1 {
		return words[0]
	}

	return strings.TrimSpace(cleaned)
}

// handleSpecialBrandNames handles special cases for well-known brands
func handleSpecialBrandNames(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))

	// Special cases for known brands
	specialCases := map[string]string{
		"visual studio code": "code.visualstudio",
		"vs code":            "code.visualstudio",
		"vscode":             "code.visualstudio",
	}

	if domain, ok := specialCases[name]; ok {
		return domain
	}

	// For other multi-word names, try to extract the main identifier
	// Usually the first word or first two words
	words := strings.Fields(name)
	if len(words) > 3 {
		// Take first two words for very long names
		return strings.Join(words[:2], "")
	}

	// Return the original name (will be processed further)
	return name
}

// sanitizeDomainName removes invalid characters for domain names
func sanitizeDomainName(name string) string {
	var result strings.Builder
	for _, r := range name {
		// Allow alphanumeric, hyphens, and dots
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			result.WriteRune(r)
		}
	}
	sanitized := result.String()

	// Remove consecutive dots or hyphens
	sanitized = strings.ReplaceAll(sanitized, "..", ".")
	sanitized = strings.ReplaceAll(sanitized, "--", "-")

	// Remove leading/trailing dots or hyphens
	sanitized = strings.Trim(sanitized, ".-")

	return sanitized
}

// parseCompetitorStrings parses competitor strings that may be in "name|domain" format
// Returns the list of names and a map of name -> domain
func parseCompetitorStrings(competitorStrings []string) ([]string, map[string]string) {
	names := make([]string, 0, len(competitorStrings))
	domainMap := make(map[string]string)

	for _, str := range competitorStrings {
		var name, domain string

		// Check if it's in "name|domain" format
		if idx := strings.Index(str, "|"); idx != -1 {
			name = strings.TrimSpace(str[:idx])
			domain = strings.TrimSpace(str[idx+1:])
		} else {
			// Old format: just the name
			name = strings.TrimSpace(str)
		}

		if name != "" {
			names = append(names, name)
			if domain != "" {
				domainMap[name] = domain
			}
		}
	}

	return names, domainMap
}

// normalizeCitationDomain normalizes a citation source domain to include www. prefix
// Example: "qodo.ai" -> "www.qodo.ai", "https://betterstack.com" -> "www.betterstack.com"
func normalizeCitationDomain(domain string) string {
	// Remove protocol if present
	normalized := strings.TrimPrefix(domain, "http://")
	normalized = strings.TrimPrefix(normalized, "https://")

	// Remove path if present (everything after first /)
	if idx := strings.Index(normalized, "/"); idx != -1 {
		normalized = normalized[:idx]
	}

	// Remove port if present (everything after :)
	if idx := strings.Index(normalized, ":"); idx != -1 {
		normalized = normalized[:idx]
	}

	// Add www. prefix if not present
	if !strings.HasPrefix(normalized, "www.") {
		normalized = "www." + normalized
	}

	return normalized
}

// extractSourceName extracts a readable name from a domain
// Example: "betterstack.com" -> "betterstack", "https://www.example.com" -> "example"
func extractSourceName(domain string) string {
	// Remove protocol if present
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")

	// Remove www. prefix if present
	domain = strings.TrimPrefix(domain, "www.")

	// Remove path if present (everything after first /)
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// Remove port if present (everything after :)
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Extract the main domain name (before first dot if it's a subdomain)
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		// For domains like "subdomain.example.com", return "example"
		// For domains like "example.com", return "example"
		if len(parts) > 2 {
			return parts[len(parts)-2]
		}
		return parts[0]
	}

	return domain
}

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 50.0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// setLastRunInfo sets the lastRunDate and lastRunStatus for the dashboard response
func (s *DashboardService) setLastRunInfo(ctx context.Context, brand string, response *models.DashboardOverviewResponse) {
	// Fetch responses for the brand (without time filter) to get the true last run date
	allFilter := shared.ResponseFilter{
		Limit: 1000, // Fetch enough to find the most recent one
	}
	allResponses, err := s.db.ListResponses(ctx, allFilter)
	var lastRunDate *time.Time
	var brandID string // Extract brandID from responses
	if err == nil {
		// Find the most recent response for this brand and extract brandID
		for _, resp := range allResponses {
			if resp.Brand == brand {
				if lastRunDate == nil || resp.CreatedAt.After(*lastRunDate) {
					lastRunDate = &resp.CreatedAt
				}
				// Extract brandID from the first matching response
				if brandID == "" && resp.BrandID != "" {
					brandID = resp.BrandID
				}
			}
		}
		if lastRunDate != nil {
			response.LastRunDate = lastRunDate
		}
	}

	// First, check if there's a running GEO campaign for this brand
	// The query filters for status="running" AND completed_at=null
	runningCampaign, err := s.db.GetRunningGEOCampaignByBrand(ctx, brand)
	if err == nil && runningCampaign != nil {
		// Double-check: if CompletedAt is set, it's not actually running
		if runningCampaign.CompletedAt == nil {
			// Additional check: if we have responses more recent than campaign creation,
			// and campaign has been running for more than 5 minutes, it's likely completed
			// but status wasn't updated (stuck campaign)
			if lastRunDate != nil && lastRunDate.After(runningCampaign.CreatedAt) {
				// Check if campaign has been "running" for more than 5 minutes
				// and we have responses after campaign start - likely completed
				timeSinceCampaign := time.Since(runningCampaign.CreatedAt)
				if timeSinceCampaign > 5*time.Minute {
					// Campaign likely completed but status wasn't updated
					// Use last response date and mark as completed
					response.LastRunStatus = "completed"
					return
				}
			}
			// Campaign is actively running
			response.LastRunStatus = "running"
			response.LastRunDate = &runningCampaign.CreatedAt
			return
		}
		// If CompletedAt is set, the campaign completed - fall through to set completed status
	}

	// Check if there's a scheduled campaign for this brand
	scheduledCampaign, err := s.db.GetScheduledCampaignByBrand(ctx, brand)
	if err == nil && scheduledCampaign != nil {
		// Check if the campaign status is "running" (actively executing)
		if scheduledCampaign.Status == "running" {
			// Similar check: if we have responses more recent than campaign start,
			// and it's been running for more than 5 minutes, it's likely completed
			if lastRunDate != nil && scheduledCampaign.LastRunAt != nil {
				if lastRunDate.After(*scheduledCampaign.LastRunAt) {
					timeSinceLastRun := time.Since(*scheduledCampaign.LastRunAt)
					if timeSinceLastRun > 5*time.Minute {
						// Likely completed but status not updated
						response.LastRunStatus = "completed"
						return
					}
				}
			}
			response.LastRunStatus = "running"
			// Update lastRunDate to campaign's created time if it's more recent
			if response.LastRunDate == nil || scheduledCampaign.CreatedAt.After(*response.LastRunDate) {
				response.LastRunDate = &scheduledCampaign.CreatedAt
			}
			// Also check LastRunAt if available (more accurate for scheduled campaigns)
			if scheduledCampaign.LastRunAt != nil {
				if response.LastRunDate == nil || scheduledCampaign.LastRunAt.After(*response.LastRunDate) {
					response.LastRunDate = scheduledCampaign.LastRunAt
				}
			}
			return
		}
		// If scheduled campaign exists but is "active", use its LastRunAt as the last run date
		// but status should be "completed" (the campaign is active but last execution completed)
		if scheduledCampaign.Status == "active" && scheduledCampaign.LastRunAt != nil {
			if response.LastRunDate == nil || scheduledCampaign.LastRunAt.After(*response.LastRunDate) {
				response.LastRunDate = scheduledCampaign.LastRunAt
			}
		}
	}

	// Check if there are enabled schedules for this brand
	// Query schedules by brandID if we have it, otherwise check all enabled schedules
	if brandID != "" {
		enabled := true
		brandSchedules, err := s.db.ListSchedules(ctx, brandID, &enabled)
		if err == nil && len(brandSchedules) > 0 {
			// Find the most recent last_run from enabled schedules for this brand
			var mostRecentSchedule *models.Schedule
			for _, schedule := range brandSchedules {
				if schedule.Enabled && schedule.LastRun != nil {
					if mostRecentSchedule == nil || schedule.LastRun.After(*mostRecentSchedule.LastRun) {
						mostRecentSchedule = schedule
					}
				}
			}

			// If we found a schedule with last_run, use it to update the status
			if mostRecentSchedule != nil && mostRecentSchedule.LastRun != nil {
				// Update last run date if schedule's last_run is more recent
				if response.LastRunDate == nil || mostRecentSchedule.LastRun.After(*response.LastRunDate) {
					response.LastRunDate = mostRecentSchedule.LastRun
				}

				// Check if schedule is currently executing (next_run is in the past and last_run is recent)
				// This indicates the schedule might be running
				if mostRecentSchedule.NextRun != nil {
					now := time.Now()
					// If next_run is in the past, the schedule should have run
					// If last_run is very recent (within last 5 minutes), it might still be running
					if mostRecentSchedule.NextRun.Before(now) {
						timeSinceLastRun := time.Since(*mostRecentSchedule.LastRun)
						if timeSinceLastRun < 5*time.Minute {
							// Schedule execution might still be in progress
							response.LastRunStatus = "running"
							return
						}
					}
				}
			}
		}
	}

	// Default to completed if no running campaign found
	// This means either:
	// 1. No campaigns exist
	// 2. All campaigns have completed (have CompletedAt set)
	// 3. All campaigns have status != "running"
	if response.LastRunDate != nil {
		response.LastRunStatus = "completed"
	}
}

// GetPromptExecutionStatus returns the prompt execution status for a given brand
func (s *DashboardService) GetPromptExecutionStatus(
	ctx context.Context,
	brandID string,
	brand string,
) (*models.PromptExecutionStatusResponse, error) {
	response := &models.PromptExecutionStatusResponse{
		BrandID: brandID,
		Brand:   brand,
	}

	// Use a temporary DashboardOverviewResponse to reuse setLastRunInfo logic
	tempResponse := &models.DashboardOverviewResponse{
		Brand: brand,
	}
	s.setLastRunInfo(ctx, brand, tempResponse)

	// Also check schedules directly using brandID for more accurate results
	if brandID != "" {
		enabled := true
		brandSchedules, err := s.db.ListSchedules(ctx, brandID, &enabled)
		if err == nil && len(brandSchedules) > 0 {
			// Find the most recent last_run from enabled schedules for this brand
			var mostRecentSchedule *models.Schedule
			for _, schedule := range brandSchedules {
				if schedule.Enabled && schedule.LastRun != nil {
					if mostRecentSchedule == nil || schedule.LastRun.After(*mostRecentSchedule.LastRun) {
						mostRecentSchedule = schedule
					}
				}
			}

			// If we found a schedule with last_run, use it to update the status
			if mostRecentSchedule != nil && mostRecentSchedule.LastRun != nil {
				// Update last run date if schedule's last_run is more recent
				if response.LastRunDate == nil || mostRecentSchedule.LastRun.After(*response.LastRunDate) {
					response.LastRunDate = mostRecentSchedule.LastRun
				}

				// Check if schedule is currently executing (next_run is in the past and last_run is recent)
				if mostRecentSchedule.NextRun != nil {
					now := time.Now()
					// If next_run is in the past, the schedule should have run
					// If last_run is very recent (within last 5 minutes), it might still be running
					if mostRecentSchedule.NextRun.Before(now) {
						timeSinceLastRun := time.Since(*mostRecentSchedule.LastRun)
						if timeSinceLastRun < 5*time.Minute {
							// Schedule execution might still be in progress
							response.LastRunStatus = "running"
							return response, nil
						}
					}
				}
			}
		}
	}

	response.LastRunDate = tempResponse.LastRunDate
	response.LastRunStatus = tempResponse.LastRunStatus

	return response, nil
}

// getTopPerformingPrompts calculates the top performing prompts based on visibility and mention rate
func (s *DashboardService) getTopPerformingPrompts(responses []*models.Response) []string {
	if len(responses) == 0 {
		return []string{}
	}

	// Aggregate performance metrics per prompt
	type promptStats struct {
		promptID      string
		promptText    string
		totalCount    int
		mentionCount  int
		visibilitySum float64
	}

	promptMap := make(map[string]*promptStats)

	for _, resp := range responses {
		if resp.PromptID == "" {
			continue
		}

		stats, exists := promptMap[resp.PromptID]
		if !exists {
			stats = &promptStats{
				promptID:   resp.PromptID,
				promptText: resp.PromptText,
			}
			promptMap[resp.PromptID] = stats
		}

		stats.totalCount++
		if resp.BrandMentioned {
			stats.mentionCount++
		}
		stats.visibilitySum += float64(resp.VisibilityScore)
	}

	// Calculate performance score for each prompt
	type promptScore struct {
		promptText string
		score      float64
	}

	var scores []promptScore
	for _, stats := range promptMap {
		if stats.totalCount == 0 {
			continue
		}

		// Calculate composite score: 60% mention rate + 40% average visibility
		mentionRate := float64(stats.mentionCount) / float64(stats.totalCount) * 100
		avgVisibility := stats.visibilitySum / float64(stats.totalCount)
		compositeScore := mentionRate*0.6 + avgVisibility*0.4

		// Use prompt text, or fallback to prompt ID if text is empty
		promptText := stats.promptText
		if promptText == "" {
			promptText = stats.promptID
		}

		scores = append(scores, promptScore{
			promptText: promptText,
			score:      compositeScore,
		})
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Get top 5 prompt texts
	topPrompts := make([]string, 0, 5)
	for i := 0; i < len(scores) && i < 5; i++ {
		topPrompts = append(topPrompts, scores[i].promptText)
	}

	return topPrompts
}

// formatISOWeek formats a time to ISO week format (e.g., "2025-W48")
// ISO week: Week 1 is the week containing January 4th, weeks start on Monday
func (s *DashboardService) formatISOWeek(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// convertDateForDisplay converts the date string to the appropriate display format
// based on granularity:
// - daily: returns the date as-is (already in "2006-01-02" format)
// - weekly: converts "2006-W02" to the start date of that week (Monday) in "2006-01-02" format
// - monthly: converts "2006-01" to "2006-01-01" (first day of month)
func (s *DashboardService) convertDateForDisplay(dateStr, granularity string) string {
	switch granularity {
	case "daily":
		// Already in correct format: "2006-01-02"
		return dateStr

	case "weekly":
		// Parse ISO week format: "2006-W02"
		// Format: "2006-W02" where 02 is the week number
		parts := strings.Split(dateStr, "-W")
		if len(parts) != 2 {
			return dateStr // Return as-is if format is unexpected
		}

		yearStr := parts[0]
		weekStr := parts[1]

		year, err := strconv.Atoi(yearStr)
		if err != nil {
			return dateStr
		}

		week, err := strconv.Atoi(weekStr)
		if err != nil {
			return dateStr
		}

		// Calculate the start date of the ISO week
		// ISO week: Week 1 is the week containing January 4th
		// Weeks start on Monday
		jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)

		// Find the Monday of the week containing Jan 4 (this is week 1)
		daysFromMonday := (int(jan4.Weekday()) + 6) % 7 // Convert Sunday=0 to Monday=0
		week1Monday := jan4.AddDate(0, 0, -daysFromMonday)

		// Calculate the start date of the requested week
		weekStart := week1Monday.AddDate(0, 0, (week-1)*7)

		return weekStart.Format("2006-01-02")

	case "monthly":
		// Parse format: "2006-01"
		parts := strings.Split(dateStr, "-")
		if len(parts) != 2 {
			return dateStr
		}

		// Return first day of month: "2006-01-01"
		return dateStr + "-01"

	default:
		return dateStr
	}
}

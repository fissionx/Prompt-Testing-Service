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
	brandLogo := s.logoService.GetBrandLogo(ctx, brand, "")

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
		return response, nil
	}

	// Calculate metrics
	metrics := s.calculateMetrics(brandResponses)
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
	competitorCounts := make(map[string]int)
	for _, resp := range brandResponses {
		for _, comp := range resp.CompetitorsMention {
			competitorCounts[comp]++
		}
	}
	response.TopCompetitors = getTopKeys(competitorCounts, 5)

	// Get top citation sources
	sourceCounts := make(map[string]int)
	for _, resp := range brandResponses {
		for _, domain := range resp.GroundingDomains {
			if domain != "" {
				sourceCounts[domain]++
			}
		}
	}
	response.TopCitationSources = getTopKeys(sourceCounts, 5)

	// Get active campaigns count
	campaigns, _ := s.db.ListScheduledCampaigns(ctx, "active")
	response.ActiveCampaigns = len(campaigns)

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
	brandLogo := s.logoService.GetBrandLogo(ctx, brand, "")

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
			// Use saved competitors
			competitors = savedCompetitors.Competitors
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
		matrixBrand := models.CompetitorMatrixBrand{
			Brand:           brand,
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
			// Use saved competitors
			competitors = savedCompetitors.Competitors
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
	dateFormat := "2006-01-02"
	if granularity == "weekly" {
		dateFormat = "2006-W02"
	} else if granularity == "monthly" {
		dateFormat = "2006-01"
	}

	brandDateData := make(map[string]map[string]*trendAggregator)
	dateSet := make(map[string]bool)

	for _, brand := range allBrands {
		brandDateData[brand] = make(map[string]*trendAggregator)
	}

	// Aggregate main brand data
	for _, resp := range mainBrandResponses {
		date := resp.CreatedAt.Format(dateFormat)
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
		trend := models.BrandTrendData{
			Brand:           brand,
			LogoURL:         logo.LogoURL,
			FallbackLogoURL: logo.FallbackLogoURL,
			IsMainBrand:     brand == mainBrand,
		}

		var values []float64
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

			values = append(values, roundToTwo(value))
		}

		trend.Values = values
		if len(values) > 0 {
			trend.CurrentValue = values[len(values)-1]
			if len(values) > 1 && values[0] > 0 {
				trend.Change = roundToTwo((values[len(values)-1] - values[0]) / values[0] * 100)
			}
		}

		trends = append(trends, trend)
	}

	return &models.TrendComparisonResponse{
		MainBrand:   mainBrand,
		Metric:      metric,
		Period:      period,
		Granularity: granularity,
		Trends:      trends,
		Dates:       dates,
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

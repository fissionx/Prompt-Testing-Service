package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/logger"
	"github.com/fissionx/gego/internal/models"
)

// ScheduledCampaignManager manages scheduled campaign executions
type ScheduledCampaignManager struct {
	db          db.Database
	llmRegistry *llm.Registry
	cron        *cron.Cron
	running     bool
	mu          sync.RWMutex
	// Track registered campaign IDs to cron entry IDs
	campaignEntries map[string]cron.EntryID
	entriesMu       sync.RWMutex
}

// NewScheduledCampaignManager creates a new scheduled campaign manager
func NewScheduledCampaignManager(database db.Database, llmRegistry *llm.Registry) *ScheduledCampaignManager {
	c := cron.New(
		cron.WithLocation(time.UTC),
		cron.WithLogger(cron.DefaultLogger),
		cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		),
	)

	return &ScheduledCampaignManager{
		db:              database,
		llmRegistry:     llmRegistry,
		cron:            c,
		campaignEntries: make(map[string]cron.EntryID),
	}
}

// Start starts the campaign manager and loads all active scheduled campaigns
func (m *ScheduledCampaignManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("scheduled campaign manager already running")
	}

	// Load all active scheduled campaigns
	campaigns, err := m.db.ListScheduledCampaigns(ctx, "active")
	if err != nil {
		logger.Error("Failed to load scheduled campaigns: %v", err)
		// Don't fail startup if we can't load campaigns
	} else {
		logger.Info("Loaded %d active scheduled campaign(s)", len(campaigns))
		for _, campaign := range campaigns {
			if err := m.registerCampaign(ctx, campaign); err != nil {
				logger.Error("Failed to register campaign %s: %v", campaign.ID, err)
			}
		}
	}

	m.cron.Start()
	m.running = true

	logger.Info("Scheduled campaign manager started successfully")
	return nil
}

// Stop stops the campaign manager
func (m *ScheduledCampaignManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.cron.Stop()
	m.running = false

	m.entriesMu.Lock()
	m.campaignEntries = make(map[string]cron.EntryID)
	m.entriesMu.Unlock()

	logger.Info("Scheduled campaign manager stopped")
}

// CreateScheduledCampaign creates and registers a new scheduled campaign
func (m *ScheduledCampaignManager) CreateScheduledCampaign(
	ctx context.Context,
	campaignName, brand string,
	promptIDs, llmIDs []string,
	temperature float64,
	scheduleCron string,
	totalRuns int,
) (*models.ScheduledCampaign, error) {
	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(scheduleCron)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time
	nextRun := schedule.Next(time.Now())

	// Calculate total runs: (promptIDs * llmIDs * runsPerPrompt)
	if totalRuns == 0 {
		totalRuns = 1
	}
	calculatedTotalRuns := len(promptIDs) * len(llmIDs) * totalRuns

	campaign := &models.ScheduledCampaign{
		ID:           uuid.New().String(),
		CampaignName: campaignName,
		Brand:        brand,
		PromptIDs:    promptIDs,
		LLMIDs:       llmIDs,
		Temperature:  temperature,
		ScheduleCron: scheduleCron,
		Status:       "active",
		TotalRuns:    calculatedTotalRuns,
		RunCount:     0,
		NextRunAt:    &nextRun,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save to database
	if err := m.db.SaveScheduledCampaign(ctx, campaign); err != nil {
		return nil, fmt.Errorf("failed to save scheduled campaign: %w", err)
	}

	// Register with cron if manager is running
	m.mu.RLock()
	isRunning := m.running
	m.mu.RUnlock()

	if isRunning {
		if err := m.registerCampaign(ctx, campaign); err != nil {
			logger.Error("Failed to register campaign %s with cron: %v", campaign.ID, err)
		}
	}

	// NOTE: We do NOT execute immediately anymore.
	// The campaign will only run at the scheduled times defined by scheduleCron.
	// If you want immediate execution, use the bulk API without scheduleCron.

	return campaign, nil
}

// registerCampaign registers a campaign with the cron scheduler
func (m *ScheduledCampaignManager) registerCampaign(_ context.Context, campaign *models.ScheduledCampaign) error {
	jobFunc := func() {
		logger.Info("Executing scheduled campaign: %s", campaign.CampaignName)
		if err := m.executeCampaign(context.Background(), campaign); err != nil {
			logger.Error("Failed to execute campaign %s: %v", campaign.ID, err)
		}
	}

	entryID, err := m.cron.AddFunc(campaign.ScheduleCron, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	m.entriesMu.Lock()
	m.campaignEntries[campaign.ID] = entryID
	m.entriesMu.Unlock()

	logger.Info("Registered scheduled campaign %s with cron expression: %s (Entry ID: %d)",
		campaign.ID, campaign.ScheduleCron, entryID)
	return nil
}

// executeCampaign executes a scheduled campaign
func (m *ScheduledCampaignManager) executeCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error {
	log.Printf("========== STARTING SCHEDULED CAMPAIGN: %s ==========", campaign.CampaignName)
	log.Printf("Brand: %s, Prompts: %d, LLMs: %d, Total Runs: %d",
		campaign.Brand, len(campaign.PromptIDs), len(campaign.LLMIDs), campaign.TotalRuns)

	startTime := time.Now()

	// Update campaign status
	campaign.Status = "running"
	campaign.UpdatedAt = time.Now()
	if err := m.db.UpdateScheduledCampaign(ctx, campaign); err != nil {
		logger.Error("Failed to update campaign status: %v", err)
	}

	// Create bulk execution service
	bulkService := NewBulkExecutionService(m.db, m.llmRegistry)

	// Execute campaign prompts with LLMs
	err := m.executePrompts(ctx, bulkService, campaign)
	if err != nil {
		logger.Error("Campaign execution failed: %v", err)
	}

	endTime := time.Now()

	// Update campaign after execution
	campaign.RunCount++
	campaign.Status = "active"
	now := time.Now()
	campaign.LastRunAt = &now

	// Calculate next run time
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if schedule, err := parser.Parse(campaign.ScheduleCron); err == nil {
		nextRun := schedule.Next(time.Now())
		campaign.NextRunAt = &nextRun
	}

	campaign.UpdatedAt = time.Now()
	if err := m.db.UpdateScheduledCampaign(ctx, campaign); err != nil {
		logger.Error("Failed to update campaign after execution: %v", err)
	}

	// Cache analytics data after execution completes
	go m.cacheAnalyticsData(context.Background(), campaign, &startTime, &endTime)

	log.Printf("========== SCHEDULED CAMPAIGN COMPLETED: %s ==========", campaign.CampaignName)
	return nil
}

// executePrompts executes all prompt-LLM combinations
func (m *ScheduledCampaignManager) executePrompts(ctx context.Context, bulkService *BulkExecutionService, campaign *models.ScheduledCampaign) error {
	// Fetch prompts
	prompts, err := bulkService.getPrompts(ctx, campaign.PromptIDs)
	if err != nil {
		return fmt.Errorf("failed to fetch prompts: %w", err)
	}

	// Fetch LLMs
	llms, err := bulkService.getLLMs(ctx, campaign.LLMIDs)
	if err != nil {
		return fmt.Errorf("failed to fetch LLMs: %w", err)
	}

	// Execute with concurrency control
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Max 3 concurrent executions
	completed := 0
	mu := sync.Mutex{}
	totalRuns := len(prompts) * len(llms)

	for _, prompt := range prompts {
		for _, llmConfig := range llms {
			wg.Add(1)

			go func(p *models.Prompt, l *models.LLMConfig) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Execute single prompt-LLM pair
				err := bulkService.executeSingle(ctx, p, l, campaign.Brand, campaign.Temperature)

				mu.Lock()
				completed++
				if completed%10 == 0 || completed == totalRuns {
					log.Printf("Scheduled Campaign %s: %d/%d completed", campaign.CampaignName, completed, totalRuns)
				}
				mu.Unlock()

				if err != nil {
					log.Printf("Execution failed for prompt %s with LLM %s: %v", p.ID, l.ID, err)
				}
			}(prompt, llmConfig)
		}
	}

	wg.Wait()
	return nil
}

// cacheAnalyticsData computes and caches all analytics data after campaign execution
func (m *ScheduledCampaignManager) cacheAnalyticsData(ctx context.Context, campaign *models.ScheduledCampaign, startTime, endTime *time.Time) {
	logger.Info("Caching analytics data for campaign: %s", campaign.CampaignName)

	// Create analytics services
	geoAnalyticsService := NewGEOAnalyticsService(m.db)
	sourceAnalyticsService := NewSourceAnalyticsService(m.db)
	competitiveService := NewCompetitiveBenchmarkService(m.db)
	promptPerformanceService := NewPromptPerformanceService(m.db)
	promptTimeSeriesService := NewPromptTimeSeriesService(m.db)

	// Cache GEO Insights
	if err := m.cacheGEOInsights(ctx, campaign, geoAnalyticsService, startTime, endTime); err != nil {
		logger.Error("Failed to cache GEO insights: %v", err)
	}

	// Cache Source Analytics
	if err := m.cacheSourceAnalytics(ctx, campaign, sourceAnalyticsService, startTime, endTime); err != nil {
		logger.Error("Failed to cache source analytics: %v", err)
	}

	// Cache Competitive Benchmark
	if err := m.cacheCompetitiveBenchmark(ctx, campaign, competitiveService, startTime, endTime); err != nil {
		logger.Error("Failed to cache competitive benchmark: %v", err)
	}

	// Cache Prompt Performance
	if err := m.cachePromptPerformance(ctx, campaign, promptPerformanceService, startTime, endTime); err != nil {
		logger.Error("Failed to cache prompt performance: %v", err)
	}

	// Cache Prompt Time Series for all prompts in the campaign
	if err := m.cachePromptTimeSeries(ctx, campaign, promptTimeSeriesService, startTime, endTime); err != nil {
		logger.Error("Failed to cache prompt time series: %v", err)
	}

	logger.Info("Analytics data cached successfully for campaign: %s", campaign.CampaignName)
}

// cacheGEOInsights computes and caches GEO insights
func (m *ScheduledCampaignManager) cacheGEOInsights(
	ctx context.Context,
	campaign *models.ScheduledCampaign,
	service *GEOAnalyticsService,
	startTime, endTime *time.Time,
) error {
	insights, err := service.GetGEOInsights(ctx, campaign.Brand, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to compute GEO insights: %w", err)
	}

	cachedInsights := &models.CachedGEOInsights{
		ID:                    uuid.New().String(),
		CampaignID:            campaign.ID,
		Brand:                 campaign.Brand,
		StartTime:             *startTime,
		EndTime:               *endTime,
		LogoURL:               insights.LogoURL,
		FallbackLogoURL:       insights.FallbackLogoURL,
		AverageVisibility:     insights.AverageVisibility,
		MentionRate:           insights.MentionRate,
		GroundingRate:         insights.GroundingRate,
		SentimentBreakdown:    insights.SentimentBreakdown,
		TopCompetitors:        insights.TopCompetitors,
		PerformanceByLLM:      insights.PerformanceByLLM,
		PerformanceByCategory: insights.PerformanceByCategory,
		Trends:                insights.Trends,
		TotalResponses:        insights.TotalResponses,
	}

	return m.db.SaveCachedGEOInsights(ctx, cachedInsights)
}

// cacheSourceAnalytics computes and caches source analytics
func (m *ScheduledCampaignManager) cacheSourceAnalytics(
	ctx context.Context,
	campaign *models.ScheduledCampaign,
	service *SourceAnalyticsService,
	startTime, endTime *time.Time,
) error {
	analytics, err := service.GetSourceAnalytics(ctx, campaign.Brand, startTime, endTime, 50)
	if err != nil {
		return fmt.Errorf("failed to compute source analytics: %w", err)
	}

	cachedAnalytics := &models.CachedSourceAnalytics{
		ID:              uuid.New().String(),
		CampaignID:      campaign.ID,
		Brand:           campaign.Brand,
		StartTime:       *startTime,
		EndTime:         *endTime,
		LogoURL:         analytics.LogoURL,
		FallbackLogoURL: analytics.FallbackLogoURL,
		Period:          analytics.Period,
		TopSources:      analytics.TopSources,
		Recommendations: analytics.Recommendations,
		TotalSources:    analytics.TotalSources,
		TotalCitations:  analytics.TotalCitations,
	}

	return m.db.SaveCachedSourceAnalytics(ctx, cachedAnalytics)
}

// cacheCompetitiveBenchmark computes and caches competitive benchmark
func (m *ScheduledCampaignManager) cacheCompetitiveBenchmark(
	ctx context.Context,
	campaign *models.ScheduledCampaign,
	service *CompetitiveBenchmarkService,
	startTime, endTime *time.Time,
) error {
	// Auto-detect competitors from responses
	benchmark, err := service.GetCompetitiveBenchmark(
		ctx,
		campaign.Brand,
		nil, // Auto-detect competitors
		campaign.PromptIDs,
		campaign.LLMIDs,
		startTime,
		endTime,
		"", // No region filter
		nil, // No competitor map for auto-detected
	)
	if err != nil {
		// Competitive benchmark may fail if no data, don't fail the whole caching
		logger.Warning("Could not compute competitive benchmark: %v", err)
		return nil
	}

	cachedBenchmark := &models.CachedCompetitiveBenchmark{
		ID:              uuid.New().String(),
		CampaignID:      campaign.ID,
		MainBrand:       campaign.Brand,
		StartTime:       *startTime,
		EndTime:         *endTime,
		MainBrandPerf:   benchmark.MainBrand,
		Competitors:     benchmark.Competitors,
		MarketLeader:    benchmark.MarketLeader,
		YourRank:        benchmark.YourRank,
		TotalBrands:     benchmark.TotalBrands,
		PromptBreakdown: benchmark.PromptBreakdown,
		Recommendations: benchmark.Recommendations,
		AnalyzedAt:      benchmark.AnalyzedAt,
	}

	return m.db.SaveCachedCompetitiveBenchmark(ctx, cachedBenchmark)
}

// cachePromptPerformance computes and caches prompt performance
func (m *ScheduledCampaignManager) cachePromptPerformance(
	ctx context.Context,
	campaign *models.ScheduledCampaign,
	service *PromptPerformanceService,
	startTime, endTime *time.Time,
) error {
	performance, err := service.GetPromptPerformance(ctx, campaign.Brand, startTime, endTime, 1)
	if err != nil {
		return fmt.Errorf("failed to compute prompt performance: %w", err)
	}

	cachedPerformance := &models.CachedPromptPerformance{
		ID:                   uuid.New().String(),
		CampaignID:           campaign.ID,
		Brand:                campaign.Brand,
		StartTime:            *startTime,
		EndTime:              *endTime,
		LogoURL:              performance.LogoURL,
		FallbackLogoURL:      performance.FallbackLogoURL,
		Period:               performance.Period,
		Prompts:              performance.Prompts,
		TopPerformers:        performance.TopPerformers,
		LowPerformers:        performance.LowPerformers,
		AvgEffectiveness:     performance.AvgEffectiveness,
		TotalPromptsAnalyzed: performance.TotalPromptsAnalyzed,
	}

	return m.db.SaveCachedPromptPerformance(ctx, cachedPerformance)
}

// cachePromptTimeSeries computes and caches prompt time series for all prompts in the campaign
func (m *ScheduledCampaignManager) cachePromptTimeSeries(
	ctx context.Context,
	campaign *models.ScheduledCampaign,
	service *PromptTimeSeriesService,
	startTime, endTime *time.Time,
) error {
	// Cache time series for each prompt in the campaign
	for _, promptID := range campaign.PromptIDs {
		timeSeries, err := service.GetPromptTimeSeries(ctx, promptID, campaign.Brand, startTime, endTime)
		if err != nil {
			logger.Warning("Failed to compute time series for prompt %s: %v", promptID, err)
			continue
		}

		cachedTimeSeries := &models.CachedPromptTimeSeries{
			ID:         uuid.New().String(),
			CampaignID: campaign.ID,
			PromptID:   promptID,
			Brand:      campaign.Brand,
			StartTime:  *startTime,
			EndTime:    *endTime,
			PromptText: timeSeries.PromptText,
			PromptType: timeSeries.PromptType,
			Category:   timeSeries.Category,
			Overview:   convertOverviewForCache(timeSeries.Overview),
			TimeSeries: convertTimeSeriesForCache(timeSeries.TimeSeries),
		}

		if err := m.db.SaveCachedPromptTimeSeries(ctx, cachedTimeSeries); err != nil {
			logger.Warning("Failed to save cached time series for prompt %s: %v", promptID, err)
		}
	}

	return nil
}

// Helper functions to convert API models to cache models
func convertOverviewForCache(overview models.PromptTimeSeriesOverview) models.PromptTimeSeriesOverview {
	llmBreakdown := make(map[string]models.PromptLLMStats)
	for k, v := range overview.LLMBreakdown {
		llmBreakdown[k] = models.PromptLLMStats{
			LLMName:       v.LLMName,
			LLMProvider:   v.LLMProvider,
			ResponseCount: v.ResponseCount,
			MentionRate:   v.MentionRate,
			AvgVisibility: v.AvgVisibility,
		}
	}

	return models.PromptTimeSeriesOverview{
		TotalResponses:     overview.TotalResponses,
		TotalMentions:      overview.TotalMentions,
		AvgVisibility:      overview.AvgVisibility,
		AvgPosition:        overview.AvgPosition,
		MentionRate:        overview.MentionRate,
		TopPositionRate:    overview.TopPositionRate,
		GroundingRate:      overview.GroundingRate,
		EffectivenessScore: overview.EffectivenessScore,
		EffectivenessGrade: overview.EffectivenessGrade,
		PositiveSentiment:  overview.PositiveSentiment,
		NeutralSentiment:   overview.NeutralSentiment,
		NegativeSentiment:  overview.NegativeSentiment,
		LLMBreakdown:       llmBreakdown,
		TopCompetitors:     overview.TopCompetitors,
	}
}

func convertTimeSeriesForCache(timeSeries []models.PromptTimeSeriesDataPoint) []models.PromptTimeSeriesDataPoint {
	result := make([]models.PromptTimeSeriesDataPoint, len(timeSeries))
	for i, dp := range timeSeries {
		result[i] = models.PromptTimeSeriesDataPoint{
			Date:           dp.Date,
			ResponseCount:  dp.ResponseCount,
			MentionCount:   dp.MentionCount,
			AvgVisibility:  dp.AvgVisibility,
			AvgPosition:    dp.AvgPosition,
			MentionRate:    dp.MentionRate,
			GroundingCount: dp.GroundingCount,
			PositiveCount:  dp.PositiveCount,
			NeutralCount:   dp.NeutralCount,
			NegativeCount:  dp.NegativeCount,
		}
	}
	return result
}

// PauseCampaign pauses a scheduled campaign
func (m *ScheduledCampaignManager) PauseCampaign(ctx context.Context, campaignID string) error {
	campaign, err := m.db.GetScheduledCampaign(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}
	if campaign == nil {
		return fmt.Errorf("campaign not found: %s", campaignID)
	}

	// Remove from cron
	m.entriesMu.Lock()
	if entryID, exists := m.campaignEntries[campaignID]; exists {
		m.cron.Remove(entryID)
		delete(m.campaignEntries, campaignID)
	}
	m.entriesMu.Unlock()

	// Update status
	campaign.Status = "paused"
	campaign.UpdatedAt = time.Now()
	return m.db.UpdateScheduledCampaign(ctx, campaign)
}

// ResumeCampaign resumes a paused campaign
func (m *ScheduledCampaignManager) ResumeCampaign(ctx context.Context, campaignID string) error {
	campaign, err := m.db.GetScheduledCampaign(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}
	if campaign == nil {
		return fmt.Errorf("campaign not found: %s", campaignID)
	}

	// Register with cron
	if err := m.registerCampaign(ctx, campaign); err != nil {
		return fmt.Errorf("failed to register campaign with cron: %w", err)
	}

	// Update status
	campaign.Status = "active"
	campaign.UpdatedAt = time.Now()
	return m.db.UpdateScheduledCampaign(ctx, campaign)
}

// GetCampaignStatus returns the status of a scheduled campaign
func (m *ScheduledCampaignManager) GetCampaignStatus(ctx context.Context, campaignID string) (*models.ScheduledCampaign, error) {
	return m.db.GetScheduledCampaign(ctx, campaignID)
}

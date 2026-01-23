package models

import (
	"time"
)

// CachedGEOInsights stores pre-computed GEO insights for fast retrieval
type CachedGEOInsights struct {
	ID         string    `json:"id" bson:"_id"`
	CampaignID string    `json:"campaignId" bson:"campaign_id"`
	BrandID    string    `json:"brandId" bson:"brand_id"`
	OrgID      string    `json:"orgId" bson:"org_id"`
	Brand      string    `json:"brand" bson:"brand"` // Brand name kept for backward compatibility
	StartTime  time.Time `json:"startTime" bson:"start_time"`
	EndTime    time.Time `json:"endTime" bson:"end_time"`

	// Cached analytics data
	LogoURL               string                `json:"logoUrl,omitempty" bson:"logo_url,omitempty"`
	FallbackLogoURL       string                `json:"fallbackLogoUrl,omitempty" bson:"fallback_logo_url,omitempty"`
	AverageVisibility     float64               `json:"averageVisibility" bson:"average_visibility"`
	MentionRate           float64               `json:"mentionRate" bson:"mention_rate"`
	GroundingRate         float64               `json:"groundingRate" bson:"grounding_rate"`
	SentimentBreakdown    map[string]int        `json:"sentimentBreakdown" bson:"sentiment_breakdown"`
	TopCompetitors        []CompetitorInsight   `json:"topCompetitors" bson:"top_competitors"`
	PerformanceByLLM      []LLMPerformance      `json:"performanceByLlm" bson:"performance_by_llm"`
	PerformanceByCategory []CategoryPerformance `json:"performanceByCategory" bson:"performance_by_category"`
	Trends                []TrendPoint          `json:"trends,omitempty" bson:"trends,omitempty"`
	TotalResponses        int                   `json:"totalResponses" bson:"total_responses"`

	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updated_at"`
}

// CachedSourceAnalytics stores pre-computed source analytics for fast retrieval
type CachedSourceAnalytics struct {
	ID         string    `json:"id" bson:"_id"`
	CampaignID string    `json:"campaignId" bson:"campaign_id"`
	BrandID    string    `json:"brandId" bson:"brand_id"`
	OrgID      string    `json:"orgId" bson:"org_id"`
	Brand      string    `json:"brand" bson:"brand"` // Brand name kept for backward compatibility
	StartTime  time.Time `json:"startTime" bson:"start_time"`
	EndTime    time.Time `json:"endTime" bson:"end_time"`

	// Cached analytics data
	LogoURL         string           `json:"logoUrl,omitempty" bson:"logo_url,omitempty"`
	FallbackLogoURL string           `json:"fallbackLogoUrl,omitempty" bson:"fallback_logo_url,omitempty"`
	Period          string           `json:"period,omitempty" bson:"period,omitempty"`
	TopSources      []SourceInsight  `json:"topSources" bson:"top_sources"`
	Recommendations []Recommendation `json:"recommendations" bson:"recommendations"`
	TotalSources    int              `json:"totalSources" bson:"total_sources"`
	TotalCitations  int              `json:"totalCitations" bson:"total_citations"`

	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updated_at"`
}

// CachedCompetitiveBenchmark stores pre-computed competitive benchmark for fast retrieval
type CachedCompetitiveBenchmark struct {
	ID         string    `json:"id" bson:"_id"`
	CampaignID string    `json:"campaignId" bson:"campaign_id"`
	BrandID    string    `json:"brandId" bson:"brand_id"`
	OrgID      string    `json:"orgId" bson:"org_id"`
	MainBrand  string    `json:"mainBrand" bson:"main_brand"` // Brand name kept for backward compatibility
	StartTime  time.Time `json:"startTime" bson:"start_time"`
	EndTime    time.Time `json:"endTime" bson:"end_time"`

	// Cached analytics data
	MainBrandPerf   BrandPerformance            `json:"mainBrand" bson:"main_brand_perf"`
	Competitors     []BrandPerformance          `json:"competitors" bson:"competitors"`
	OtherBrands     []BrandPerformance          `json:"otherBrands,omitempty" bson:"other_brands,omitempty"`
	MarketLeader    string                      `json:"marketLeader" bson:"market_leader"`
	YourRank        int                         `json:"yourRank" bson:"your_rank"`
	TotalBrands     int                         `json:"totalBrands" bson:"total_brands"`
	PromptBreakdown []PromptCompetitiveAnalysis `json:"promptBreakdown" bson:"prompt_breakdown"`
	AnalyzedAt      time.Time                   `json:"analyzedAt" bson:"analyzed_at"`

	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updated_at"`
}

// CachedPromptPerformance stores pre-computed prompt performance analytics for fast retrieval
type CachedPromptPerformance struct {
	ID         string    `json:"id" bson:"_id"`
	CampaignID string    `json:"campaignId" bson:"campaign_id"`
	BrandID    string    `json:"brandId" bson:"brand_id"`
	OrgID      string    `json:"orgId" bson:"org_id"`
	Brand      string    `json:"brand" bson:"brand"` // Brand name kept for backward compatibility
	StartTime  time.Time `json:"startTime" bson:"start_time"`
	EndTime    time.Time `json:"endTime" bson:"end_time"`

	// Cached analytics data
	LogoURL              string              `json:"logoUrl,omitempty" bson:"logo_url,omitempty"`
	FallbackLogoURL      string              `json:"fallbackLogoUrl,omitempty" bson:"fallback_logo_url,omitempty"`
	Period               string              `json:"period,omitempty" bson:"period,omitempty"`
	Prompts              []PromptPerformance `json:"prompts" bson:"prompts"`
	TopPerformers        []string            `json:"topPerformers" bson:"top_performers"`
	LowPerformers        []string            `json:"lowPerformers" bson:"low_performers"`
	AvgEffectiveness     float64             `json:"avgEffectiveness" bson:"avg_effectiveness"`
	TotalPromptsAnalyzed int                 `json:"totalPromptsAnalyzed" bson:"total_prompts_analyzed"`

	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updated_at"`
}

// ScheduledCampaign represents a campaign with scheduled periodic execution
type ScheduledCampaign struct {
	ID           string     `json:"id" bson:"_id"`
	BrandID      string     `json:"brandId,omitempty" bson:"brand_id,omitempty"` // Brand ID (UUID)
	OrgID        string     `json:"orgId,omitempty" bson:"org_id,omitempty"`     // Organization ID
	CampaignName string     `json:"campaignName" bson:"campaign_name"`
	Brand        string     `json:"brand" bson:"brand"` // Brand name (kept for backward compatibility)
	PromptIDs    []string   `json:"promptIds" bson:"prompt_ids"`
	LLMIDs       []string   `json:"llmIds" bson:"llm_ids"`
	Temperature  float64    `json:"temperature" bson:"temperature"`
	ScheduleCron string     `json:"scheduleCron" bson:"schedule_cron"`      // Cron expression for periodic execution
	Status       string     `json:"status" bson:"status"`                   // active, paused, completed
	TotalRuns    int        `json:"totalRuns" bson:"total_runs"`            // Total runs per execution
	RunCount     int        `json:"runCount" bson:"run_count"`              // How many times the campaign has run
	LastRunAt    *time.Time `json:"lastRunAt,omitempty" bson:"last_run_at"` // Last execution time
	NextRunAt    *time.Time `json:"nextRunAt,omitempty" bson:"next_run_at"` // Next scheduled execution
	CreatedAt    time.Time  `json:"createdAt" bson:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" bson:"updated_at"`
}

// ==================== Scheduled Campaigns API ====================

// ListScheduledCampaignsRequest represents the request to list scheduled campaigns
type ListScheduledCampaignsRequest struct {
	Brand  string `json:"brand" form:"brand"`   // Filter by brand (optional)
	Status string `json:"status" form:"status"` // Filter by status: active, paused, completed (optional)
}

// ScheduledCampaignDetail represents a scheduled campaign with full details
type ScheduledCampaignDetail struct {
	ID           string         `json:"id"`
	CampaignName string         `json:"campaignName"`
	Brand        string         `json:"brand"`
	Prompts      []PromptDetail `json:"prompts"` // Full prompt details
	LLMs         []LLMDetail    `json:"llms"`    // Full LLM details
	Temperature  float64        `json:"temperature"`
	ScheduleCron string         `json:"scheduleCron"`
	ScheduleDesc string         `json:"scheduleDescription"` // Human-readable schedule
	Status       string         `json:"status"`
	TotalRuns    int            `json:"totalRuns"`
	RunCount     int            `json:"runCount"`
	LastRunAt    *time.Time     `json:"lastRunAt,omitempty"`
	NextRunAt    *time.Time     `json:"nextRunAt,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// PromptDetail represents prompt details for scheduled campaign
type PromptDetail struct {
	ID                      string     `json:"id"`
	Template                string     `json:"template"`
	PromptType              PromptType  `json:"promptType"`
	Category                string     `json:"category,omitempty"`
	TargetingSearchKeywords []string   `json:"targetingSearchKeywords,omitempty"` // Keywords for targeting
	SupportingFanoutQueries []string   `json:"supportingFanoutQueries,omitempty"`   // Related queries
}

// LLMDetail represents LLM details for scheduled campaign
type LLMDetail struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// ListScheduledCampaignsResponse represents the response with scheduled campaigns
type ListScheduledCampaignsResponse struct {
	Brand     string                    `json:"brand,omitempty"`
	Campaigns []ScheduledCampaignDetail `json:"campaigns"`
	Total     int                       `json:"total"`
	Active    int                       `json:"active"`
	Paused    int                       `json:"paused"`
	Completed int                       `json:"completed"`
}

// AnalyticsCacheQuery represents a query for retrieving cached analytics
type AnalyticsCacheQuery struct {
	CampaignID string     `json:"campaignId,omitempty"`
	BrandID    string     `json:"brandId,omitempty"`
	OrgID      string     `json:"orgId,omitempty"`
	PromptID   string     `json:"promptId,omitempty"`
	StartTime  *time.Time `json:"startTime,omitempty"`
	EndTime    *time.Time `json:"endTime,omitempty"`
}

// CachedPromptTimeSeries stores pre-computed prompt analytics time series for fast retrieval
// Uses PromptTimeSeriesOverview, PromptLLMStats, and PromptTimeSeriesDataPoint types defined in api.go
type CachedPromptTimeSeries struct {
	ID         string    `json:"id" bson:"_id"`
	CampaignID string    `json:"campaignId" bson:"campaign_id"`
	PromptID   string    `json:"promptId" bson:"prompt_id"`
	BrandID    string    `json:"brandId" bson:"brand_id"`
	OrgID      string    `json:"orgId" bson:"org_id"`
	Brand      string    `json:"brand" bson:"brand"` // Brand name kept for backward compatibility
	StartTime  time.Time `json:"startTime" bson:"start_time"`
	EndTime    time.Time `json:"endTime" bson:"end_time"`

	// Prompt metadata
	PromptText string `json:"promptText" bson:"prompt_text"`
	PromptType string `json:"promptType" bson:"prompt_type"`
	Category   string `json:"category" bson:"category"`

	// Overview statistics (uses PromptTimeSeriesOverview from api.go)
	Overview PromptTimeSeriesOverview `json:"overview" bson:"overview"`

	// Time series data (uses PromptTimeSeriesDataPoint from api.go)
	TimeSeries []PromptTimeSeriesDataPoint `json:"timeSeries" bson:"time_series"`

	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updated_at"`
}

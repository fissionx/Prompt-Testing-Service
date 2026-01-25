package models

import (
	"time"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// CreateLLMRequest represents the request to create a new LLM
type CreateLLMRequest struct {
	Name     string            `json:"name" binding:"required"`
	Provider string            `json:"provider" binding:"required"`
	Model    string            `json:"model" binding:"required"`
	APIKey   string            `json:"apiKey,omitempty"`
	BaseURL  string            `json:"baseUrl,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
	Enabled  bool              `json:"enabled"`
}

// UpdateLLMRequest represents the request to update an existing LLM
type UpdateLLMRequest struct {
	Name     string            `json:"name,omitempty"`
	Provider string            `json:"provider,omitempty"`
	Model    string            `json:"model,omitempty"`
	APIKey   string            `json:"apiKey,omitempty"`
	BaseURL  string            `json:"baseUrl,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
	Enabled  *bool             `json:"enabled,omitempty"`
}

// LLMResponse represents the response for LLM operations
type LLMResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Provider  string            `json:"provider"`
	Model     string            `json:"model"`
	APIKey    string            `json:"apiKey,omitempty"`
	BaseURL   string            `json:"baseUrl,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// CreatePromptRequest represents the request to create a new prompt
type CreatePromptRequest struct {
	Template string   `json:"template" binding:"required"`
	Tags     []string `json:"tags,omitempty"`
	Enabled  bool     `json:"enabled"`
}

// UpdatePromptRequest represents the request to update an existing prompt
type UpdatePromptRequest struct {
	Template string   `json:"template,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Enabled  *bool    `json:"enabled,omitempty"`
}

// PromptResponse represents the response for prompt operations
type PromptResponse struct {
	ID        string    `json:"id"`
	Template  string    `json:"template"`
	Tags      []string  `json:"tags,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateScheduleRequest represents the request to create a new schedule
// Note: brandId is extracted from the URL path parameter, not from the request body
type CreateScheduleRequest struct {
	Name        string   `json:"name" binding:"required"`
	PromptIDs   []string `json:"promptIds" binding:"required"`
	LLMIDs      []string `json:"llmIds" binding:"required"`
	CronExpr    string   `json:"cronExpr" binding:"required"`
	Temperature float64  `json:"temperature,omitempty"`
	Enabled     bool     `json:"enabled"`
}

// UpdateScheduleRequest represents the request to update an existing schedule
type UpdateScheduleRequest struct {
	Name        string   `json:"name,omitempty"`
	PromptIDs   []string `json:"promptIds,omitempty"`
	LLMIDs      []string `json:"llmIds,omitempty"`
	CronExpr    string   `json:"cronExpr,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

// ScheduleResponse represents the response for schedule operations
type ScheduleResponse struct {
	ID          string     `json:"id"`
	BrandID     string     `json:"brandId"`
	Name        string     `json:"name"`
	PromptIDs   []string   `json:"promptIds"`
	LLMIDs      []string   `json:"llmIds"`
	CronExpr    string     `json:"cronExpr"`
	Temperature float64    `json:"temperature"`
	Enabled     bool       `json:"enabled"`
	LastRun     *time.Time `json:"lastRun,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// StatsResponse represents the response for statistics
type StatsResponse struct {
	TotalResponses int64             `json:"totalResponses"`
	TotalPrompts   int64             `json:"totalPrompts"`
	TotalLLMs      int64             `json:"totalLlms"`
	TotalSchedules int64             `json:"totalSchedules"`
	TopKeywords    []KeywordCount    `json:"topKeywords"`
	PromptStats    []*PromptStats    `json:"promptStats"`
	LLMStats       []*LLMStats       `json:"llmStats"`
	ResponseTrends []TimeSeriesPoint `json:"responseTrends"`
	LastUpdated    time.Time         `json:"lastUpdated"`
}

// SearchRequest represents the request to search responses
type SearchRequest struct {
	Keyword   string     `json:"keyword" binding:"required"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	Limit     int        `json:"limit,omitempty"`
}

// SearchResponse represents the response for search operations
type SearchResponse struct {
	Keyword       string         `json:"keyword"`
	TotalMentions int            `json:"totalMentions"`
	UniquePrompts int            `json:"uniquePrompts"`
	UniqueLLMs    int            `json:"uniqueLlms"`
	ByPrompt      map[string]int `json:"byPrompt"`
	ByLLM         map[string]int `json:"byLlm"`
	ByProvider    map[string]int `json:"byProvider"`
	FirstSeen     time.Time      `json:"firstSeen"`
	LastSeen      time.Time      `json:"lastSeen"`
	Responses     []*Response    `json:"responses,omitempty"`
}

// ExecuteRequest represents the request to execute a prompt against an LLM
type ExecuteRequest struct {
	Prompt      string   `json:"prompt" binding:"required"`
	LLMID       string   `json:"llmId" binding:"required"`
	Brand       string   `json:"brand,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	SavePrompt  bool     `json:"savePrompt,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Region      string   `json:"region,omitempty"`
	Language    string   `json:"language,omitempty"`
}

// ExecuteResponse represents the response from executing a prompt
type ExecuteResponse struct {
	ResponseID  string       `json:"responseId"`
	PromptID    string       `json:"promptId,omitempty"`
	Prompt      string       `json:"prompt"`
	Brand       string       `json:"brand,omitempty"`
	Response    string       `json:"response"`
	GEOAnalysis *GEOAnalysis `json:"geoAnalysis,omitempty"`
	LLMName     string       `json:"llmName"`
	LLMProvider string       `json:"llmProvider"`
	LLMModel    string       `json:"llmModel"`
	Temperature float64      `json:"temperature"`
	TokensUsed  int          `json:"tokensUsed"`
	LatencyMs   int64        `json:"latencyMs"`
	CreatedAt   time.Time    `json:"createdAt"`
}

// GEOAnalysis represents the GEO (Generative Engine Optimization) analysis results
type GEOAnalysis struct {
	VisibilityScore    int      `json:"visibilityScore"`
	BrandMentioned     bool     `json:"brandMentioned"`
	InGroundingSources bool     `json:"inGroundingSources"`
	MentionStatus      string   `json:"mentionStatus"`
	Reason             string   `json:"reason"`
	Insights           []string `json:"insights"`
	Actions            []string `json:"actions"`
	CompetitorInfo     string   `json:"competitorInfo,omitempty"`
	Competitors        []string `json:"competitors,omitempty"`
	Sentiment          string   `json:"sentiment,omitempty"`
}

// GeneratePromptsRequest represents the request to generate prompts for a brand
type GeneratePromptsRequest struct {
	BrandID     string `json:"brandId" binding:"required"` // Brand ID (UUID) from external API
	Website     string `json:"website,omitempty"`
	Category    string `json:"category,omitempty"`
	Domain      string `json:"domain,omitempty"` // Deprecated: will be fetched from brand API
	Description string `json:"description,omitempty"`
	Count       int    `json:"count,omitempty"`
	LLMID       string `json:"llmId,omitempty"` // Optional LLM ID to use for generation (defaults to Google/Gemini if not provided)
}

// GeneratePromptsResponse represents the response with generated prompts
type GeneratePromptsResponse struct {
	Brand         string                     `json:"brand"`
	BrandID       string                     `json:"brandId"`
	Category      string                     `json:"category"`
	Domain        string                     `json:"domain"`
	PromptsByType map[string][]PromptPreview `json:"promptsByType"`
	Existing      int                        `json:"existingPrompts"`
	Generated     int                        `json:"generatedPrompts"`
	TypeCounts    map[string]int             `json:"typeCounts"`
}

// PromptPreview represents a preview of a prompt
type PromptPreview struct {
	ID         string     `json:"id"`
	Template   string     `json:"template"`
	PromptType PromptType `json:"promptType,omitempty"`
	Category   string     `json:"category,omitempty"`
	Reused     bool       `json:"reused"`
}

// BulkExecuteRequest represents the request to execute multiple prompts across multiple LLMs
type BulkExecuteRequest struct {
	CampaignName string   `json:"campaignName" binding:"required"`
	BrandID      string   `json:"brandId" binding:"required"`   // Brand ID (UUID) from external API
	PromptIDs    []string `json:"promptIds" binding:"required"` // Existing prompt IDs from the database
	LLMIDs       []string `json:"llmIds" binding:"required"`
	Temperature  float64  `json:"temperature,omitempty"`
	ScheduleCron string   `json:"scheduleCron,omitempty"` // Optional cron expression for periodic execution (e.g., "0 */6 * * *" for every 6 hours)
	TotalRuns    int      `json:"totalRuns,omitempty"`    // Number of times to run each prompt (default: 1)
}

// CustomPrompt represents a user-created prompt for bulk execution
type CustomPrompt struct {
	Template   string   `json:"template" binding:"required"`
	PromptType string   `json:"promptType,omitempty"` // e.g., "custom", "top_best", "how_to", etc.
	Category   string   `json:"category,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// SaveCustomPromptsRequest represents the request to save custom prompts along with promptIds from suggested prompts
type SaveCustomPromptsRequest struct {
	BrandID       string         `json:"brandId,omitempty"`       // Brand ID (UUID) - now obtained from path parameter
	PromptIDs     []string       `json:"promptIds,omitempty"`     // Prompt IDs from suggested prompts
	CustomPrompts []CustomPrompt `json:"customPrompts,omitempty"` // New custom prompts to create
}

// SaveCustomPromptsResponse represents the response after saving custom prompts
type SaveCustomPromptsResponse struct {
	Brand          string   `json:"brand"`
	SavedPromptIDs []string `json:"savedPromptIds"` // All prompt IDs (both existing and newly created)
	CreatedCount   int      `json:"createdCount"`   // Number of new prompts created
	ExistingCount  int      `json:"existingCount"`  // Number of existing prompt IDs provided
}

// DeletePromptsRequest represents the request to delete prompts by IDs
type DeletePromptsRequest struct {
	PromptIDs []string `json:"promptIds" binding:"required"` // List of prompt IDs to delete (1 or more)
}

// DeletePromptsResponse represents the response after deleting prompts
type DeletePromptsResponse struct {
	DeletedCount int      `json:"deletedCount"`         // Number of prompts successfully deleted
	FailedCount  int      `json:"failedCount"`          // Number of prompts that failed to delete
	DeletedIDs   []string `json:"deletedIds,omitempty"` // IDs of successfully deleted prompts
	FailedIDs    []string `json:"failedIds,omitempty"`  // IDs that failed to delete
}

// BulkExecuteResponse represents the response from bulk execution
type BulkExecuteResponse struct {
	CampaignID   string     `json:"campaignId"`
	CampaignName string     `json:"campaignName"`
	Brand        string     `json:"brand"`
	TotalRuns    int        `json:"totalRuns"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"startedAt"`
	NextRunAt    *time.Time `json:"nextRunAt,omitempty"` // Next scheduled execution time (if scheduled)
	ScheduleCron string     `json:"scheduleCron,omitempty"`
	Message      string     `json:"message"`
}

// GEOInsightsRequest represents the request for GEO insights/analytics
type GEOInsightsRequest struct {
	BrandID      string     `json:"brandId" binding:"required"` // Brand ID (UUID) from external API
	CampaignID   string     `json:"campaignId,omitempty"`
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	ForceRefresh bool       `json:"forceRefresh,omitempty"` // Force re-computation even if cached
}

// GEOInsightsResponse represents comprehensive GEO analytics
type GEOInsightsResponse struct {
	Brand                 string                `json:"brand"`
	LogoURL               string                `json:"logoUrl,omitempty"`
	FallbackLogoURL       string                `json:"fallbackLogoUrl,omitempty"`
	AverageVisibility     float64               `json:"averageVisibility"`
	MentionRate           float64               `json:"mentionRate"`
	GroundingRate         float64               `json:"groundingRate"`
	SentimentBreakdown    map[string]int        `json:"sentimentBreakdown"`
	TopCompetitors        []CompetitorInsight   `json:"topCompetitors"`
	PerformanceByLLM      []LLMPerformance      `json:"performanceByLlm"`
	PerformanceByCategory []CategoryPerformance `json:"performanceByCategory"`
	Trends                []TrendPoint          `json:"trends,omitempty"`
	TotalResponses        int                   `json:"totalResponses"`
}

// CompetitorInsight represents competitor visibility data
type CompetitorInsight struct {
	Name            string  `json:"name"`
	Domain          string  `json:"domain,omitempty"`
	LogoURL         string  `json:"logoUrl,omitempty"`
	FallbackLogoURL string  `json:"fallbackLogoUrl,omitempty"`
	MentionCount    int     `json:"mentionCount"`
	VisibilityAvg   float64 `json:"visibilityAvg"`
}

// LLMPerformance represents brand performance per LLM
type LLMPerformance struct {
	LLMName       string  `json:"llmName"`
	LLMProvider   string  `json:"llmProvider"`
	Visibility    float64 `json:"visibility"`
	MentionRate   float64 `json:"mentionRate"`
	ResponseCount int     `json:"responseCount"`
}

// CategoryPerformance represents brand performance per category
type CategoryPerformance struct {
	Category      string  `json:"category"`
	Visibility    float64 `json:"visibility"`
	MentionRate   float64 `json:"mentionRate"`
	ResponseCount int     `json:"responseCount"`
}

// TrendPoint represents a time-series data point
type TrendPoint struct {
	Date       string  `json:"date"`
	Visibility float64 `json:"visibility"`
	Mentions   int     `json:"mentions"`
}

// SourceInsight represents analytics for a specific citation source
type SourceInsight struct {
	Domain        string         `json:"domain"`
	CitationCount int            `json:"citationCount"`
	MentionRate   float64        `json:"mentionRate"`
	LLMBreakdown  map[string]int `json:"llmBreakdown"`
	Categories    []string       `json:"categories"`
	GroundingURLs []string       `json:"groundingUrls,omitempty"`
}

// Recommendation represents an actionable insight for the user
type Recommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Impact      string `json:"impact"`
}

// SourceAnalyticsResponse represents source citation analytics
type SourceAnalyticsResponse struct {
	Brand           string           `json:"brand"`
	LogoURL         string           `json:"logoUrl,omitempty"`
	FallbackLogoURL string           `json:"fallbackLogoUrl,omitempty"`
	Period          string           `json:"period,omitempty"`
	TopSources      []SourceInsight  `json:"topSources"`
	Recommendations []Recommendation `json:"recommendations"`
	TotalSources    int              `json:"totalSources"`
	TotalCitations  int              `json:"totalCitations"`
}

// CompetitiveBenchmarkRequest represents request for competitive analysis
type CompetitiveBenchmarkRequest struct {
	BrandID      string       `json:"brandId" form:"brandId" binding:"required"` // Brand ID (UUID) from external API
	Competitors  []Competitor `json:"competitors,omitempty" form:"competitors"`
	PromptIDs    []string     `json:"promptIds,omitempty" form:"promptIds"`
	LLMIDs       []string     `json:"llmIds,omitempty" form:"llmIds"`
	StartTime    *time.Time   `json:"startTime,omitempty" form:"startTime"`
	EndTime      *time.Time   `json:"endTime,omitempty" form:"endTime"`
	Region       string       `json:"region,omitempty" form:"region"`
	ForceRefresh bool         `json:"forceRefresh,omitempty" form:"forceRefresh"` // Force re-computation even if cached
}

// BrandPerformance represents comprehensive brand performance metrics
type BrandPerformance struct {
	Brand           string  `json:"brand"`
	Domain          string  `json:"domain,omitempty"`
	LogoURL         string  `json:"logoUrl,omitempty"`
	FallbackLogoURL string  `json:"fallbackLogoUrl,omitempty"`
	Visibility      float64 `json:"visibility"`
	MentionRate     float64 `json:"mentionRate"`
	GroundingRate   float64 `json:"groundingRate"`
	AveragePosition float64 `json:"averagePosition"`
	TopPositionRate float64 `json:"topPositionRate"`
	SentimentScore  float64 `json:"sentimentScore"`
	ResponseCount   int     `json:"responseCount"`
	MarketSharePct  float64 `json:"marketSharePct"`
}

// CompetitiveBenchmarkResponse represents competitive analysis results
type CompetitiveBenchmarkResponse struct {
	MainBrand       BrandPerformance            `json:"mainBrand"`
	Competitors     []BrandPerformance          `json:"competitors"`
	OtherBrands     []BrandPerformance          `json:"otherBrands,omitempty"` // Brands mentioned but not in tracked competitors list
	MarketLeader    string                      `json:"marketLeader"`
	YourRank        int                         `json:"yourRank"`
	TotalBrands     int                         `json:"totalBrands"`
	PromptBreakdown []PromptCompetitiveAnalysis `json:"promptBreakdown"`
	AnalyzedAt      time.Time                   `json:"analyzedAt"`
}

// PromptCompetitiveAnalysis shows competitive performance for a specific prompt
type PromptCompetitiveAnalysis struct {
	PromptID                      string                       `json:"promptId"`
	PromptText                    string                       `json:"promptText"`
	PromptType                    string                       `json:"promptType,omitempty"`
	MainBrandResult               PromptBrandResult            `json:"mainBrandResult"`
	TrackedCompetitorsMentioned   []PromptCompetitorMention    `json:"trackedCompetitorsMentioned"`
	UntrackedCompetitorsMentioned []UntrackedCompetitorMention `json:"untrackedCompetitorsMentioned,omitempty"` // Brands mentioned but not in tracked list
	Winner                        string                       `json:"winner"`
	TotalBrandsMentioned          int                          `json:"totalBrandsMentioned"`
	ExecutedAt                    time.Time                    `json:"executedAt"`
	Insights                      *PromptInsights              `json:"insights,omitempty"`
}

// UntrackedCompetitorMention shows how an untracked competitor appeared in a prompt response
type UntrackedCompetitorMention struct {
	Brand           string  `json:"brand"`
	VisibilityScore int     `json:"visibilityScore"`
	Position        int     `json:"position"`
	Sentiment       string  `json:"sentiment"`
	InSources       bool    `json:"inSources"`
	ShareOfVoice    float64 `json:"shareOfVoice"` // Percentage of all brands mentioned in this prompt
}

// PromptInsights provides gap analysis for dashboard visualization
type PromptInsights struct {
	// Gap percentage: 0-100, where 0 = no gap, 100 = maximum gap
	GapPercentage float64 `json:"gapPercentage"`
	// Reason category: "blog_missing", "topic_not_discussed", "low_review", "poor_seo", "no_mention", "low_visibility", "poor_position", "missing_citations", "negative_sentiment", "none"
	ReasonCategory string `json:"reasonCategory"`
	// Gap severity: "none", "low", "medium", "high", "critical"
	Severity string `json:"severity"`
	// Leading competitor (if any)
	LeadingCompetitor string `json:"leadingCompetitor,omitempty"`
	// Recommendations to fill the gap (at prompt level)
	Recommendations []string `json:"recommendations,omitempty"`
}

// PromptBrandResult shows how the main brand performed on a specific prompt
type PromptBrandResult struct {
	Mentioned       bool    `json:"mentioned"`
	VisibilityScore int     `json:"visibilityScore"`
	Position        int     `json:"position"`
	Sentiment       string  `json:"sentiment"`
	InSources       bool    `json:"inSources"`
	ShareOfVoice    float64 `json:"shareOfVoice"` // Percentage of all brands mentioned in this prompt
}

// PromptCompetitorMention shows how a competitor appeared in a prompt response
type PromptCompetitorMention struct {
	Brand     string `json:"brand"`
	Mentioned bool   `json:"mentioned"`
}

// SourceAnalyticsRequest represents request for source analytics
type SourceAnalyticsRequest struct {
	BrandID      string     `json:"brandId" form:"brandId" binding:"required"` // Brand ID (UUID) from external API
	StartTime    *time.Time `json:"startTime" form:"startTime"`                // ISO 8601 format (e.g., "2024-01-01T00:00:00Z")
	EndTime      *time.Time `json:"endTime" form:"endTime"`                    // ISO 8601 format (e.g., "2024-01-31T23:59:59Z")
	TopN         int        `json:"topN" form:"topN"`                          // Number of top sources to return (default: 20)
	ForceRefresh bool       `json:"forceRefresh" form:"forceRefresh"`          // Force re-computation even if cached
}

// PositionAnalyticsResponse represents position/ranking analytics
type PositionAnalyticsResponse struct {
	Brand             string             `json:"brand"`
	LogoURL           string             `json:"logoUrl,omitempty"`
	FallbackLogoURL   string             `json:"fallbackLogoUrl,omitempty"`
	AveragePosition   float64            `json:"averagePosition"`
	TopPositionRate   float64            `json:"topPositionRate"`
	PositionBreakdown map[string]int     `json:"positionBreakdown"`
	ByPromptType      map[string]float64 `json:"byPromptType"`
	ByLLM             map[string]float64 `json:"byLlm"`
	TotalMentions     int                `json:"totalMentions"`
}

// PromptPerformanceRequest represents request for prompt performance analysis
type PromptPerformanceRequest struct {
	BrandID      string     `json:"brandId" form:"brandId" binding:"required"` // Brand ID (UUID) from external API
	StartTime    *time.Time `json:"startTime,omitempty" form:"startTime"`
	EndTime      *time.Time `json:"endTime,omitempty" form:"endTime"`
	MinResponses int        `json:"minResponses,omitempty" form:"minResponses"`
	ForceRefresh bool       `json:"forceRefresh,omitempty" form:"forceRefresh"` // Force re-computation even if cached
}

// PromptPerformanceResponse represents prompt performance analysis results
type PromptPerformanceResponse struct {
	Brand                string              `json:"brand"`
	LogoURL              string              `json:"logoUrl,omitempty"`
	FallbackLogoURL      string              `json:"fallbackLogoUrl,omitempty"`
	Period               string              `json:"period,omitempty"`
	Prompts              []PromptPerformance `json:"prompts"`
	TopPerformers        []string            `json:"topPerformers"`
	LowPerformers        []string            `json:"lowPerformers"`
	AvgEffectiveness     float64             `json:"avgEffectiveness"`
	TotalPromptsAnalyzed int                 `json:"totalPromptsAnalyzed"`
	TotalPromptsFound    int                 `json:"totalPromptsFound,omitempty"`    // Total prompts found (before filtering)
	FilteredOutCount     int                 `json:"filteredOutCount,omitempty"`     // Prompts filtered out due to minResponses
	MinResponsesRequired int                 `json:"minResponsesRequired,omitempty"` // The minResponses threshold used
}

// PromptPerformance represents detailed performance metrics for a single prompt
type PromptPerformance struct {
	PromptID   string `json:"promptId"`
	PromptText string `json:"promptText"`
	PromptType string `json:"promptType"`
	Category   string `json:"category,omitempty"`

	// Performance Metrics
	AvgVisibility   float64 `json:"avgVisibility"`
	AvgPosition     float64 `json:"avgPosition"`
	MentionRate     float64 `json:"mentionRate"`
	TopPositionRate float64 `json:"topPositionRate"`
	AvgSentiment    float64 `json:"avgSentiment"`

	// Volume Metrics
	TotalResponses int `json:"totalResponses"`
	BrandMentions  int `json:"brandMentions"`

	// Effectiveness Metrics
	EffectivenessScore float64 `json:"effectivenessScore"`
	EffectivenessGrade string  `json:"effectivenessGrade"`
	Status             string  `json:"status"`
	Recommendation     string  `json:"recommendation"`
}

// PromptTimeSeriesRequest represents the request for prompt time series analytics
type PromptTimeSeriesRequest struct {
	PromptID  string     `json:"promptId" form:"promptId" binding:"required"`
	Brand     string     `json:"brand" form:"brand"`
	StartTime *time.Time `json:"startTime" form:"startTime"`
	EndTime   *time.Time `json:"endTime" form:"endTime"`
}

// PromptTimeSeriesResponse represents the response with prompt time series analytics
type PromptTimeSeriesResponse struct {
	PromptID   string `json:"promptId"`
	PromptText string `json:"promptText"`
	PromptType string `json:"promptType"`
	Category   string `json:"category"`
	Brand      string `json:"brand,omitempty"`
	Period     string `json:"period"`

	// Overview statistics
	Overview PromptTimeSeriesOverview `json:"overview"`

	// Time series data
	TimeSeries []PromptTimeSeriesDataPoint `json:"timeSeries"`
}

// PromptTimeSeriesOverview contains aggregated statistics for the prompt
type PromptTimeSeriesOverview struct {
	TotalResponses     int     `json:"totalResponses"`
	TotalMentions      int     `json:"totalMentions"`
	AvgVisibility      float64 `json:"avgVisibility"`
	AvgPosition        float64 `json:"avgPosition"`
	MentionRate        float64 `json:"mentionRate"`
	TopPositionRate    float64 `json:"topPositionRate"`
	GroundingRate      float64 `json:"groundingRate"`
	EffectivenessScore float64 `json:"effectivenessScore"`
	EffectivenessGrade string  `json:"effectivenessGrade"`

	// Sentiment breakdown
	PositiveSentiment int `json:"positiveSentiment"`
	NeutralSentiment  int `json:"neutralSentiment"`
	NegativeSentiment int `json:"negativeSentiment"`

	// LLM breakdown
	LLMBreakdown map[string]PromptLLMStats `json:"llmBreakdown"`

	// Top competitors mentioned in responses
	TopCompetitors []TopCompetitor `json:"topCompetitors"`
}

// PromptLLMStats contains stats for a specific LLM
type PromptLLMStats struct {
	LLMName       string  `json:"llmName"`
	LLMProvider   string  `json:"llmProvider"`
	ResponseCount int     `json:"responseCount"`
	MentionRate   float64 `json:"mentionRate"`
	AvgVisibility float64 `json:"avgVisibility"`
}

// PromptTimeSeriesDataPoint represents a single data point in the time series
type PromptTimeSeriesDataPoint struct {
	Date           string  `json:"date"`
	ResponseCount  int     `json:"responseCount"`
	MentionCount   int     `json:"mentionCount"`
	AvgVisibility  float64 `json:"avgVisibility"`
	AvgPosition    float64 `json:"avgPosition"`
	MentionRate    float64 `json:"mentionRate"`
	GroundingCount int     `json:"groundingCount"`
	PositiveCount  int     `json:"positiveCount"`
	NeutralCount   int     `json:"neutralCount"`
	NegativeCount  int     `json:"negativeCount"`
}

// ==================== NEW PEEC-LIKE FEATURES ====================

// DashboardOverviewRequest represents the request for dashboard overview
type DashboardOverviewRequest struct {
	BrandID   string     `json:"brandId" form:"brandId" binding:"required"` // Brand ID (UUID) from external API
	StartTime *time.Time `json:"startTime" form:"startTime"`
	EndTime   *time.Time `json:"endTime" form:"endTime"`
}

// TopCompetitor represents a top competitor with name and domain
type TopCompetitor struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// TopCitationSource represents a top citation source with name and domain
type TopCitationSource struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// DashboardOverviewResponse represents the dashboard overview with key metrics
type DashboardOverviewResponse struct {
	Brand           string `json:"brand"`
	LogoURL         string `json:"logoUrl,omitempty"`
	FallbackLogoURL string `json:"fallbackLogoUrl,omitempty"`
	Period          string `json:"period"`

	// Key Metrics Summary
	Visibility      DashboardMetric `json:"visibility"`
	Sentiment       DashboardMetric `json:"sentiment"`
	Position        DashboardMetric `json:"position"`
	GroundingRate   DashboardMetric `json:"groundingRate"`
	TotalResponses  int             `json:"totalResponses"`
	TotalPrompts    int             `json:"totalPrompts"`
	TotalLLMs       int             `json:"totalLlms"`
	ActiveCampaigns int             `json:"activeCampaigns"`

	// Quick Trend (last 7 data points)
	TrendData []TrendDataPoint `json:"trendData"`

	// Top Insights
	TopPerformingPrompts []string            `json:"topPerformingPrompts"`
	TopCompetitors       []TopCompetitor     `json:"topCompetitor"`
	TopCitationSources   []TopCitationSource `json:"topCitationSources"`
	RecommendationsCount int                 `json:"recommendationsCount"`

	// Last Run Information
	LastRunDate   *time.Time `json:"lastRunDate,omitempty"`
	LastRunStatus string     `json:"lastRunStatus,omitempty"` // "completed" or "running"
}

// PromptExecutionStatusResponse represents the prompt execution status for a brand
type PromptExecutionStatusResponse struct {
	BrandID       string     `json:"brandId"`
	Brand         string     `json:"brand"`
	LastRunDate   *time.Time `json:"lastRunDate,omitempty"`
	LastRunStatus string     `json:"lastRunStatus,omitempty"` // "completed" or "running"
}

// DashboardMetric represents a single dashboard metric with trend
type DashboardMetric struct {
	Value       float64 `json:"value"`
	Change      float64 `json:"change"`      // Percentage change from previous period
	Trend       string  `json:"trend"`       // "up", "down", "stable"
	Rank        int     `json:"rank"`        // Rank among competitors (if applicable)
	TotalBrands int     `json:"totalBrands"` // Total brands in comparison
}

// TrendDataPoint represents a single point in trend data
type TrendDataPoint struct {
	Date       string  `json:"date"`
	Visibility float64 `json:"visibility"`
	Sentiment  float64 `json:"sentiment"`
	Position   float64 `json:"position"`
}

// ModelAnalyticsRequest represents the request for model-level analytics
type ModelAnalyticsRequest struct {
	BrandID   string     `json:"brandId" form:"brandId" binding:"required"` // Brand ID (UUID) from external API
	StartTime *time.Time `json:"startTime,omitempty" form:"startTime"`
	EndTime   *time.Time `json:"endTime,omitempty" form:"endTime"`
}

// ModelAnalyticsResponse represents analytics broken down by AI model
type ModelAnalyticsResponse struct {
	Brand           string             `json:"brand"`
	LogoURL         string             `json:"logoUrl,omitempty"`
	FallbackLogoURL string             `json:"fallbackLogoUrl,omitempty"`
	Period          string             `json:"period"`
	Models          []ModelPerformance `json:"models"`
	BestModel       string             `json:"bestModel"`
	WorstModel      string             `json:"worstModel"`
	Recommendations []Recommendation   `json:"recommendations"`
}

// ModelPerformance represents performance metrics for a specific AI model
type ModelPerformance struct {
	ModelID        string  `json:"modelId"`
	ModelName      string  `json:"modelName"`
	Provider       string  `json:"provider"`
	ResponseCount  int     `json:"responseCount"`
	Visibility     float64 `json:"visibility"`
	MentionRate    float64 `json:"mentionRate"`
	AvgPosition    float64 `json:"avgPosition"`
	TopPositionPct float64 `json:"topPositionPct"`
	GroundingRate  float64 `json:"groundingRate"`
	SentimentScore float64 `json:"sentimentScore"`
	AvgLatencyMs   int64   `json:"avgLatencyMs"`
}

// CompetitorMatrixRequest represents the request for competitor matrix
type CompetitorMatrixRequest struct {
	MainBrandID string       `json:"mainBrandId" binding:"required"` // Brand ID (UUID) from external API
	Competitors []Competitor `json:"competitors,omitempty"`
	StartTime   *time.Time   `json:"startTime,omitempty"`
	EndTime     *time.Time   `json:"endTime,omitempty"`
}

// CompetitorMatrixResponse represents the visibility vs sentiment quadrant matrix
type CompetitorMatrixResponse struct {
	MainBrand string                  `json:"mainBrand"`
	Period    string                  `json:"period"`
	Quadrants CompetitorQuadrants     `json:"quadrants"`
	Brands    []CompetitorMatrixBrand `json:"brands"`
	AxisInfo  MatrixAxisInfo          `json:"axisInfo"`
}

// CompetitorQuadrants categorizes brands into quadrants
type CompetitorQuadrants struct {
	Leaders       []string `json:"leaders"`       // High visibility, High sentiment
	NichePlayers  []string `json:"nichePlayers"`  // Low visibility, High sentiment
	Laggers       []string `json:"laggers"`       // Low visibility, Low sentiment
	Controversial []string `json:"controversial"` // High visibility, Low sentiment
}

// CompetitorMatrixBrand represents a brand's position in the matrix
type CompetitorMatrixBrand struct {
	Brand           string  `json:"brand"`
	Domain          string  `json:"domain,omitempty"`
	LogoURL         string  `json:"logoUrl,omitempty"`
	FallbackLogoURL string  `json:"fallbackLogoUrl,omitempty"`
	Visibility      float64 `json:"visibility"` // X-axis (0-100%)
	Sentiment       float64 `json:"sentiment"`  // Y-axis (0-1 or 0-100)
	Position        float64 `json:"position"`   // Average position
	ResponseCount   int     `json:"responseCount"`
	Quadrant        string  `json:"quadrant"` // "leader", "niche_player", "lagger", "controversial"
	IsMainBrand     bool    `json:"isMainBrand"`
}

// MatrixAxisInfo provides axis information for the matrix
type MatrixAxisInfo struct {
	VisibilityMedian float64 `json:"visibilityMedian"` // Median visibility for quadrant division
	SentimentMedian  float64 `json:"sentimentMedian"`  // Median sentiment for quadrant division
	VisibilityMax    float64 `json:"visibilityMax"`
	SentimentMax     float64 `json:"sentimentMax"`
}

// TrendComparisonRequest represents the request for trend comparison
type TrendComparisonRequest struct {
	BrandID     string       `json:"brandId" form:"brandId" binding:"required"` // Brand ID (UUID) from external API
	Competitors []Competitor `json:"competitors,omitempty" form:"competitors"`
	Metric      string       `json:"metric" form:"metric"` // "visibility", "sentiment", "position"
	StartTime   *time.Time   `json:"startTime,omitempty" form:"startTime"`
	EndTime     *time.Time   `json:"endTime,omitempty" form:"endTime"`
	Granularity string       `json:"granularity,omitempty" form:"granularity"` // "daily", "weekly", "monthly"
}

// TrendComparisonResponse represents trend data for multiple brands
type TrendComparisonResponse struct {
	MainBrand   string           `json:"mainBrand"`
	Metric      string           `json:"metric"`
	Period      string           `json:"period"`
	Granularity string           `json:"granularity"`
	Trends      []BrandTrendData `json:"trends"`
	Dates       []string         `json:"dates"`
}

// TrendValue represents a single data point in a trend
type TrendValue struct {
	Value float64 `json:"value"`
	Date  string  `json:"date"` // Date string: actual date (daily), start of week (weekly), start of month (monthly)
}

// BrandTrendData represents trend data for a single brand
type BrandTrendData struct {
	Brand           string       `json:"brand"`
	Domain          string       `json:"domain,omitempty"`
	LogoURL         string       `json:"logoUrl,omitempty"`
	FallbackLogoURL string       `json:"fallbackLogoUrl,omitempty"`
	Values          []TrendValue `json:"values"`
	IsMainBrand     bool         `json:"isMainBrand"`
	CurrentValue    float64      `json:"currentValue"`
	Change          float64      `json:"change"` // Percentage change
}

// ExportRequest represents the request for data export
type ExportRequest struct {
	BrandID    string     `json:"brandId" binding:"required"`    // Brand ID (UUID) from external API
	Brand      string     `json:"brand,omitempty"`               // Brand name (populated from brandId, used internally)
	ExportType string     `json:"exportType" binding:"required"` // "insights", "prompts", "responses", "sources", "competitive"
	Format     string     `json:"format,omitempty"`              // "csv", "json" (default: csv)
	StartTime  *time.Time `json:"startTime,omitempty"`
	EndTime    *time.Time `json:"endTime,omitempty"`
	PromptIDs  []string   `json:"promptIds,omitempty"`
	LLMIDs     []string   `json:"llmIds,omitempty"`
}

// ExportResponse represents the export response
type ExportResponse struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Data        string `json:"data"` // Base64 encoded for binary, raw for JSON
	RecordCount int    `json:"recordCount"`
}

// ==================== COMPETITOR MANAGEMENT ====================

// Competitor represents a competitor with name and domain/URL
type Competitor struct {
	Name   string `json:"name"`
	Domain string `json:"domain"` // URL or domain (e.g., "www.windsurf.com")
}

// SuggestCompetitorsRequest represents the request to suggest competitors for a brand
type SuggestCompetitorsRequest struct {
	BrandID      string `json:"brandId" form:"brandId" binding:"required"` // Brand ID (UUID) from external API
	Website      string `json:"website" form:"website"`
	Description  string `json:"description" form:"description"`
	Category     string `json:"category" form:"category"`
	ForceRefresh bool   `json:"forceRefresh" form:"forceRefresh"` // Force re-computation even if cached
}

// SuggestCompetitorsResponse represents the response with suggested competitors
type SuggestCompetitorsResponse struct {
	Brand       string       `json:"brand"`
	Competitors []Competitor `json:"competitors"`
	Source      string       `json:"source"` // "cached", "computed", "responses"
	Message     string       `json:"message,omitempty"`
	LLMDetails  *LLMDetails  `json:"llmDetails,omitempty"` // Information about the LLM used for suggestions
}

// LLMDetails represents information about the LLM used for competitor suggestions
type LLMDetails struct {
	ID       string `json:"id"`       // LLM ID
	Name     string `json:"name"`     // LLM name
	Provider string `json:"provider"` // Provider name (e.g., "openai", "google")
	Model    string `json:"model"`    // Model name
}

// SaveCompetitorsRequest represents the request to save user-defined competitors
type SaveCompetitorsRequest struct {
	BrandID     string       `json:"brandId,omitempty"`              // Brand ID (UUID) - now obtained from path parameter
	Competitors []Competitor `json:"competitors" binding:"required"` // User's final competitor list with name and domain
	Source      string       `json:"source,omitempty"`               // "suggested", "custom", "mixed"
}

// SaveCompetitorsResponse represents the response after saving competitors
type SaveCompetitorsResponse struct {
	Brand       string       `json:"brand"`
	Competitors []Competitor `json:"competitors"`
	Source      string       `json:"source"`
	SavedAt     time.Time    `json:"savedAt"`
	Message     string       `json:"message"`
}

// DeleteCompetitorResponse represents the response after deleting a competitor
type DeleteCompetitorResponse struct {
	Brand       string       `json:"brand"`
	DeletedName string       `json:"deletedName"`
	Competitors []Competitor `json:"competitors"` // Updated list after deletion
	Message     string       `json:"message"`
}

// GetCompetitorsResponse represents the response with brand's saved competitors
type GetCompetitorsResponse struct {
	BrandID       string       `json:"brandId"`
	Brand         string       `json:"brand"`
	Competitors   []Competitor `json:"competitors"`
	SuggestedList []Competitor `json:"suggestedList,omitempty"` // Original suggestions for reference
	Source        string       `json:"source"`
	UpdatedAt     time.Time    `json:"updatedAt"`
	LLMDetails    *LLMDetails  `json:"llmDetails,omitempty"` // Information about the LLM used for suggestions
}

// GetPromptsResponse represents the response with brand's active and suggested prompts
type GetPromptsResponse struct {
	Brand            string         `json:"brand"`
	ActivePrompts    []PromptDetail `json:"activePrompts"`
	SuggestedPrompts []PromptDetail `json:"suggestedPrompts,omitempty"`
	Source           string         `json:"source"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	LLMDetails       *LLMDetails    `json:"llmDetails,omitempty"` // Information about the LLM used for suggestions
}

// SuggestPromptsResponse represents the response with suggested prompts
type SuggestPromptsResponse struct {
	Brand      string         `json:"brand"`
	Prompts    []PromptDetail `json:"prompts"`
	Source     string         `json:"source"` // "cached", "llm"
	Message    string         `json:"message,omitempty"`
	LLMDetails *LLMDetails    `json:"llmDetails,omitempty"` // Information about the LLM used for suggestions
}

// SavePromptsResponse represents the response after saving prompts
type SavePromptsResponse struct {
	Brand          string   `json:"brand"`
	SavedPromptIDs []string `json:"savedPromptIds"` // All prompt IDs (both existing and newly created)
	CreatedCount   int      `json:"createdCount"`   // Number of new prompts created
	ExistingCount  int      `json:"existingCount"`  // Number of existing prompt IDs provided
}

// ==================== OPPORTUNITIES FEATURE ====================

// OpportunityFilter represents filters for listing opportunities
type OpportunityFilter struct {
	BrandID   string `json:"brandId" form:"brandId"`
	PromptID  string `json:"promptId" form:"promptId"`
	Status    string `json:"status" form:"status"`       // new, in_progress, completed, archived
	Type      string `json:"type" form:"type"`           // opportunity type filter
	MinImpact int    `json:"minImpact" form:"minImpact"` // minimum impact score
	Limit     int    `json:"limit" form:"limit"`
	Offset    int    `json:"offset" form:"offset"`
}

// ListOpportunitiesResponse represents the response for listing opportunities
type ListOpportunitiesResponse struct {
	Brand         string        `json:"brand"`
	BrandID       string        `json:"brandId"`
	Opportunities []Opportunity `json:"opportunities"`
	Total         int64         `json:"total"`
	Pagination    Pagination    `json:"pagination"`
}

// OpportunityDetailResponse represents a single opportunity with full details
type OpportunityDetailResponse struct {
	Opportunity Opportunity `json:"opportunity"`
	Action      *Action     `json:"action,omitempty"` // Linked action if exists
	Prompt      *Prompt     `json:"prompt,omitempty"` // Related prompt details
}

// SuppressOpportunityRequest represents the request to suppress/archive an opportunity
type SuppressOpportunityRequest struct {
	Reason string `json:"reason,omitempty"` // Optional reason for suppression
}

// SuppressOpportunityResponse represents the response after suppressing an opportunity
type SuppressOpportunityResponse struct {
	OpportunityID string    `json:"opportunityId"`
	IsArchived    bool      `json:"isArchived"`
	SuppressedAt  time.Time `json:"suppressedAt"`
	Message       string    `json:"message"`
}

// ConvertToActionRequest represents the request to convert an opportunity to an action
type ConvertToActionRequest struct {
	AdditionalContext string `json:"additionalContext,omitempty"` // Extra context for LLM
}

// ConvertToActionResponse represents the response after converting opportunity to action
type ConvertToActionResponse struct {
	OpportunityID string  `json:"opportunityId"`
	Action        *Action `json:"action"`
	Message       string  `json:"message"`
}

// BatchConvertToActionRequest represents the request to convert multiple opportunities to actions
type BatchConvertToActionRequest struct {
	OpportunityIDs    []string `json:"opportunityIds" binding:"required,min=1"` // List of opportunity IDs to convert
	AdditionalContext string   `json:"additionalContext,omitempty"`             // Extra context for LLM (applied to all)
}

// BatchConvertResult represents the result of converting a single opportunity in a batch operation
type BatchConvertResult struct {
	OpportunityID string  `json:"opportunityId"`
	Success       bool    `json:"success"`
	Action        *Action `json:"action,omitempty"`
	Error         string  `json:"error,omitempty"`
}

// BatchConvertToActionResponse represents the response after converting multiple opportunities to actions
type BatchConvertToActionResponse struct {
	Results      []BatchConvertResult `json:"results"`
	TotalCount   int                  `json:"totalCount"`
	SuccessCount int                  `json:"successCount"`
	FailureCount int                  `json:"failureCount"`
	Message      string               `json:"message"`
}

// BatchSuppressOpportunityRequest represents the request to suppress multiple opportunities
type BatchSuppressOpportunityRequest struct {
	OpportunityIDs []string `json:"opportunityIds" binding:"required,min=1"` // List of opportunity IDs to suppress
	Reason         string   `json:"reason,omitempty"`                        // Optional reason for suppression (applied to all)
}

// BatchSuppressResult represents the result of suppressing a single opportunity in a batch operation
type BatchSuppressResult struct {
	OpportunityID string    `json:"opportunityId"`
	Success       bool      `json:"success"`
	IsArchived    bool      `json:"isArchived"`
	SuppressedAt  time.Time `json:"suppressedAt,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// BatchSuppressOpportunityResponse represents the response after suppressing multiple opportunities
type BatchSuppressOpportunityResponse struct {
	Results      []BatchSuppressResult `json:"results"`
	TotalCount   int                   `json:"totalCount"`
	SuccessCount int                   `json:"successCount"`
	FailureCount int                   `json:"failureCount"`
	Message      string                `json:"message"`
}

// ListActionsResponse represents the response for listing actions
type ListActionsResponse struct {
	Brand      string     `json:"brand"`
	BrandID    string     `json:"brandId"`
	Actions    []Action   `json:"actions"`
	Total      int64      `json:"total"`
	Pagination Pagination `json:"pagination"`
}

// UpdateActionRequest represents the request to update an action
type UpdateActionRequest struct {
	Status        *ActionStatus `json:"status,omitempty"`
	CompletedStep *int          `json:"completedStep,omitempty"` // Mark a specific step as completed
}

// UpdateActionResponse represents the response after updating an action
type UpdateActionResponse struct {
	Action  *Action `json:"action"`
	Message string  `json:"message"`
}

// UpdateActionStatusRequest represents the request to update action status
type UpdateActionStatusRequest struct {
	Status   ActionStatus `json:"status" binding:"required"` // "completed", "in_progress", etc.
	Assignee *string      `json:"assignee,omitempty"`        // Optional: User ID or email to assign the action
}

// UpdateActionStatusResponse represents the response after updating action status
type UpdateActionStatusResponse struct {
	Action  *Action `json:"action"`
	Message string  `json:"message"`
}

// GetActionResponse represents the response for getting a single action
type GetActionResponse struct {
	Action *Action `json:"action,omitempty"`
	// Used when action is being prepared
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"` // "preparing" when action is being generated
}

// GetOpportunityActionsResponse represents the response for getting actions for an opportunity
type GetOpportunityActionsResponse struct {
	OpportunityID string   `json:"opportunity_id"`
	Actions       []Action `json:"actions"`
	Total         int      `json:"total"`
}

// ActionFilter represents filters for listing actions
type ActionFilter struct {
	BrandID       string `json:"brandId" form:"brandId"`
	OpportunityID string `json:"opportunityId" form:"opportunityId"`
	Status        string `json:"status" form:"status"`
	Limit         int    `json:"limit" form:"limit"`
	Offset        int    `json:"offset" form:"offset"`
}

// OpportunitySummary represents a summary of opportunities for dashboard
type OpportunitySummary struct {
	TotalOpportunities      int            `json:"totalOpportunities"`
	OpenOpportunities       int            `json:"openOpportunities"`
	InProgressOpportunities int            `json:"inProgressOpportunities"`
	CompletedOpportunities  int            `json:"completedOpportunities"`
	DismissedOpportunities  int            `json:"dismissedOpportunities"`
	ByType                  map[string]int `json:"byType"`
	AvgImpactScore          float64        `json:"avgImpactScore"`
	TopOpportunities        []Opportunity  `json:"topOpportunities"` // Top 5 by impact score
}

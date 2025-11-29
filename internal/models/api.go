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
	TotalPages int   `json:"total_pages"`
}

// CreateLLMRequest represents the request to create a new LLM
type CreateLLMRequest struct {
	Name     string            `json:"name" binding:"required"`
	Provider string            `json:"provider" binding:"required"`
	Model    string            `json:"model" binding:"required"`
	APIKey   string            `json:"api_key,omitempty"`
	BaseURL  string            `json:"base_url,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
	Enabled  bool              `json:"enabled"`
}

// UpdateLLMRequest represents the request to update an existing LLM
type UpdateLLMRequest struct {
	Name     string            `json:"name,omitempty"`
	Provider string            `json:"provider,omitempty"`
	Model    string            `json:"model,omitempty"`
	APIKey   string            `json:"api_key,omitempty"`
	BaseURL  string            `json:"base_url,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
	Enabled  *bool             `json:"enabled,omitempty"`
}

// LLMResponse represents the response for LLM operations
type LLMResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Provider  string            `json:"provider"`
	Model     string            `json:"model"`
	APIKey    string            `json:"api_key,omitempty"`
	BaseURL   string            `json:"base_url,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateScheduleRequest represents the request to create a new schedule
type CreateScheduleRequest struct {
	Name        string   `json:"name" binding:"required"`
	PromptIDs   []string `json:"prompt_ids" binding:"required"`
	LLMIDs      []string `json:"llm_ids" binding:"required"`
	CronExpr    string   `json:"cron_expr" binding:"required"`
	Temperature float64  `json:"temperature,omitempty"`
	Enabled     bool     `json:"enabled"`
}

// UpdateScheduleRequest represents the request to update an existing schedule
type UpdateScheduleRequest struct {
	Name        string   `json:"name,omitempty"`
	PromptIDs   []string `json:"prompt_ids,omitempty"`
	LLMIDs      []string `json:"llm_ids,omitempty"`
	CronExpr    string   `json:"cron_expr,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

// ScheduleResponse represents the response for schedule operations
type ScheduleResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	PromptIDs   []string   `json:"prompt_ids"`
	LLMIDs      []string   `json:"llm_ids"`
	CronExpr    string     `json:"cron_expr"`
	Temperature float64    `json:"temperature"`
	Enabled     bool       `json:"enabled"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// StatsResponse represents the response for statistics
type StatsResponse struct {
	TotalResponses int64             `json:"total_responses"`
	TotalPrompts   int64             `json:"total_prompts"`
	TotalLLMs      int64             `json:"total_llms"`
	TotalSchedules int64             `json:"total_schedules"`
	TopKeywords    []KeywordCount    `json:"top_keywords"`
	PromptStats    []*PromptStats    `json:"prompt_stats"`
	LLMStats       []*LLMStats       `json:"llm_stats"`
	ResponseTrends []TimeSeriesPoint `json:"response_trends"`
	LastUpdated    time.Time         `json:"last_updated"`
}

// SearchRequest represents the request to search responses
type SearchRequest struct {
	Keyword   string     `json:"keyword" binding:"required"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Limit     int        `json:"limit,omitempty"`
}

// SearchResponse represents the response for search operations
type SearchResponse struct {
	Keyword       string         `json:"keyword"`
	TotalMentions int            `json:"total_mentions"`
	UniquePrompts int            `json:"unique_prompts"`
	UniqueLLMs    int            `json:"unique_llms"`
	ByPrompt      map[string]int `json:"by_prompt"`
	ByLLM         map[string]int `json:"by_llm"`
	ByProvider    map[string]int `json:"by_provider"`
	FirstSeen     time.Time      `json:"first_seen"`
	LastSeen      time.Time      `json:"last_seen"`
	Responses     []*Response    `json:"responses,omitempty"`
}

// ExecuteRequest represents the request to execute a prompt against an LLM
type ExecuteRequest struct {
	Prompt      string   `json:"prompt" binding:"required"`
	LLMID       string   `json:"llm_id" binding:"required"`
	Brand       string   `json:"brand,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	SavePrompt  bool     `json:"save_prompt,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Region      string   `json:"region,omitempty"`   // NEW: Region/country code (e.g., "US", "UK", "DE")
	Language    string   `json:"language,omitempty"` // NEW: Language code (e.g., "en", "es", "de")
}

// ExecuteResponse represents the response from executing a prompt
type ExecuteResponse struct {
	ResponseID  string       `json:"response_id"`
	PromptID    string       `json:"prompt_id,omitempty"`
	Prompt      string       `json:"prompt"`
	Brand       string       `json:"brand,omitempty"`
	Response    string       `json:"response"`
	GEOAnalysis *GEOAnalysis `json:"geo_analysis,omitempty"`
	LLMName     string       `json:"llm_name"`
	LLMProvider string       `json:"llm_provider"`
	LLMModel    string       `json:"llm_model"`
	Temperature float64      `json:"temperature"`
	TokensUsed  int          `json:"tokens_used"`
	LatencyMs   int64        `json:"latency_ms"`
	CreatedAt   time.Time    `json:"created_at"`
}

// GEOAnalysis represents the GEO (Generative Engine Optimization) analysis results
type GEOAnalysis struct {
	VisibilityScore    int      `json:"visibility_score"`
	BrandMentioned     bool     `json:"brand_mentioned"`
	InGroundingSources bool     `json:"in_grounding_sources"`
	MentionStatus      string   `json:"mention_status"`
	Reason             string   `json:"reason"`
	Insights           []string `json:"insights"`
	Actions            []string `json:"actions"`
	CompetitorInfo     string   `json:"competitor_info,omitempty"`
	Competitors        []string `json:"competitors,omitempty"`
	Sentiment          string   `json:"sentiment,omitempty"`
}

// GeneratePromptsRequest represents the request to generate prompts for a brand
type GeneratePromptsRequest struct {
	Brand       string `json:"brand" binding:"required"`
	Website     string `json:"website,omitempty"` // Website URL for content scraping
	Category    string `json:"category,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Description string `json:"description,omitempty"` // Optional if website provided
	Count       int    `json:"count,omitempty"`       // Number of prompts to generate (default: 20)
}

// GeneratePromptsResponse represents the response with generated prompts
type GeneratePromptsResponse struct {
	Brand         string                     `json:"brand"`
	Category      string                     `json:"category"`
	Domain        string                     `json:"domain"`
	Prompts       []PromptPreview            `json:"prompts"`           // All prompts (flat list)
	PromptsByType map[string][]PromptPreview `json:"prompts_by_type"`   // Grouped by type
	Existing      int                        `json:"existing_prompts"`  // How many were reused from DB
	Generated     int                        `json:"generated_prompts"` // How many were newly generated
	TypeCounts    map[string]int             `json:"type_counts"`       // Count per type
}

// PromptPreview represents a preview of a prompt
type PromptPreview struct {
	ID         string     `json:"id"`
	Template   string     `json:"template"`
	PromptType PromptType `json:"prompt_type,omitempty"`
	Category   string     `json:"category,omitempty"`
	Reused     bool       `json:"reused"` // True if reused from database
}

// BulkExecuteRequest represents the request to execute multiple prompts across multiple LLMs
type BulkExecuteRequest struct {
	CampaignName string   `json:"campaign_name" binding:"required"`
	Brand        string   `json:"brand" binding:"required"`
	PromptIDs    []string `json:"prompt_ids" binding:"required"`
	LLMIDs       []string `json:"llm_ids" binding:"required"`
	Temperature  float64  `json:"temperature,omitempty"`
}

// BulkExecuteResponse represents the response from bulk execution
type BulkExecuteResponse struct {
	CampaignID   string    `json:"campaign_id"`
	CampaignName string    `json:"campaign_name"`
	Brand        string    `json:"brand"`
	TotalRuns    int       `json:"total_runs"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	Message      string    `json:"message"`
}

// GEOInsightsRequest represents the request for GEO insights/analytics
type GEOInsightsRequest struct {
	Brand      string     `json:"brand,omitempty"`
	CampaignID string     `json:"campaign_id,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
}

// GEOInsightsResponse represents comprehensive GEO analytics
type GEOInsightsResponse struct {
	Brand                 string                `json:"brand"`
	AverageVisibility     float64               `json:"average_visibility"`
	MentionRate           float64               `json:"mention_rate"`        // % of responses mentioning brand
	GroundingRate         float64               `json:"grounding_rate"`      // % of responses citing brand sources
	SentimentBreakdown    map[string]int        `json:"sentiment_breakdown"` // positive/neutral/negative counts
	TopCompetitors        []CompetitorInsight   `json:"top_competitors"`
	PerformanceByLLM      []LLMPerformance      `json:"performance_by_llm"`
	PerformanceByCategory []CategoryPerformance `json:"performance_by_category"`
	Trends                []TrendPoint          `json:"trends,omitempty"`
	TotalResponses        int                   `json:"total_responses"`
}

// CompetitorInsight represents competitor visibility data
type CompetitorInsight struct {
	Name          string  `json:"name"`
	MentionCount  int     `json:"mention_count"`
	VisibilityAvg float64 `json:"visibility_avg"`
}

// LLMPerformance represents brand performance per LLM
type LLMPerformance struct {
	LLMName       string  `json:"llm_name"`
	LLMProvider   string  `json:"llm_provider"`
	Visibility    float64 `json:"visibility"`
	MentionRate   float64 `json:"mention_rate"`
	ResponseCount int     `json:"response_count"`
}

// CategoryPerformance represents brand performance per category
type CategoryPerformance struct {
	Category      string  `json:"category"`
	Visibility    float64 `json:"visibility"`
	MentionRate   float64 `json:"mention_rate"`
	ResponseCount int     `json:"response_count"`
}

// TrendPoint represents a time-series data point
type TrendPoint struct {
	Date       string  `json:"date"`
	Visibility float64 `json:"visibility"`
	Mentions   int     `json:"mentions"`
}

// SourceInsight represents analytics for a specific citation source
type SourceInsight struct {
	Domain        string         `json:"domain"`         // "g2.com", "reddit.com", "nytimes.com"
	CitationCount int            `json:"citation_count"` // How many times this source was cited
	MentionRate   float64        `json:"mention_rate"`   // % of total responses citing this source
	LLMBreakdown  map[string]int `json:"llm_breakdown"`  // Which LLMs cite it most
	Categories    []string       `json:"categories"`     // Source categories (review_site, social_media, news, etc)
}

// Recommendation represents an actionable insight for the user
type Recommendation struct {
	Type        string `json:"type"`        // "source_opportunity", "content_gap", "competitor_threat"
	Priority    string `json:"priority"`    // "high", "medium", "low"
	Title       string `json:"title"`       // Short title
	Description string `json:"description"` // Detailed explanation
	Action      string `json:"action"`      // Specific action to take
	Impact      string `json:"impact"`      // Expected impact ("high", "medium", "low")
}

// SourceAnalyticsResponse represents source citation analytics
type SourceAnalyticsResponse struct {
	Brand           string           `json:"brand"`
	Period          string           `json:"period,omitempty"` // Optional time period
	TopSources      []SourceInsight  `json:"top_sources"`      // Most cited sources
	Recommendations []Recommendation `json:"recommendations"`  // Actionable insights
	TotalSources    int              `json:"total_sources"`    // Unique sources found
	TotalCitations  int              `json:"total_citations"`  // Total citation count
}

// CompetitiveBenchmarkRequest represents request for competitive analysis
type CompetitiveBenchmarkRequest struct {
	MainBrand   string     `json:"main_brand" binding:"required"`
	Competitors []string   `json:"competitors" binding:"required"`
	PromptIDs   []string   `json:"prompt_ids,omitempty"`
	LLMIDs      []string   `json:"llm_ids,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Region      string     `json:"region,omitempty"`
}

// BrandPerformance represents comprehensive brand performance metrics
type BrandPerformance struct {
	Brand           string  `json:"brand"`
	Visibility      float64 `json:"visibility"`        // Average visibility score
	MentionRate     float64 `json:"mention_rate"`      // % of responses mentioning brand
	GroundingRate   float64 `json:"grounding_rate"`    // % of responses citing brand sources
	AveragePosition float64 `json:"average_position"`  // Average rank when mentioned (1=best)
	TopPositionRate float64 `json:"top_position_rate"` // % of times in top 3
	SentimentScore  float64 `json:"sentiment_score"`   // -1 (negative) to +1 (positive)
	ResponseCount   int     `json:"response_count"`    // Total responses analyzed
	MarketSharePct  float64 `json:"market_share_pct"`  // % of total visibility in market
}

// CompetitiveBenchmarkResponse represents competitive analysis results
type CompetitiveBenchmarkResponse struct {
	MainBrand       BrandPerformance   `json:"main_brand"`
	Competitors     []BrandPerformance `json:"competitors"`
	MarketLeader    string             `json:"market_leader"`   // Brand with highest visibility
	YourRank        int                `json:"your_rank"`       // Rank among analyzed brands
	TotalBrands     int                `json:"total_brands"`    // Total brands in analysis
	Recommendations []Recommendation   `json:"recommendations"` // Strategic recommendations
	AnalyzedAt      time.Time          `json:"analyzed_at"`
}

// SourceAnalyticsRequest represents request for source analytics
type SourceAnalyticsRequest struct {
	Brand     string     `json:"brand" binding:"required"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	TopN      int        `json:"top_n,omitempty"` // Number of top sources to return (default: 20)
}

// PositionAnalyticsResponse represents position/ranking analytics
type PositionAnalyticsResponse struct {
	Brand             string             `json:"brand"`
	AveragePosition   float64            `json:"average_position"`   // Average rank when mentioned
	TopPositionRate   float64            `json:"top_position_rate"`  // % times in position 1-3
	PositionBreakdown map[string]int     `json:"position_breakdown"` // Count per position
	ByPromptType      map[string]float64 `json:"by_prompt_type"`     // Average position by prompt type
	ByLLM             map[string]float64 `json:"by_llm"`             // Average position by LLM
	TotalMentions     int                `json:"total_mentions"`     // Total times brand was mentioned
}

// PromptPerformanceRequest represents request for prompt performance analysis
type PromptPerformanceRequest struct {
	Brand        string     `json:"brand" binding:"required"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	MinResponses int        `json:"min_responses,omitempty"` // Minimum responses per prompt (default: 3)
}

// PromptPerformanceResponse represents prompt performance analysis results
type PromptPerformanceResponse struct {
	Brand                string              `json:"brand"`
	Period               string              `json:"period,omitempty"`
	Prompts              []PromptPerformance `json:"prompts"`
	TopPerformers        []string            `json:"top_performers"`        // Prompt IDs of high performers
	LowPerformers        []string            `json:"low_performers"`        // Prompt IDs of low performers
	AvgEffectiveness     float64             `json:"avg_effectiveness"`     // Average effectiveness across all prompts
	TotalPromptsAnalyzed int                 `json:"total_prompts_analyzed"`
}

// PromptPerformance represents detailed performance metrics for a single prompt
type PromptPerformance struct {
	PromptID           string  `json:"prompt_id"`
	PromptText         string  `json:"prompt_text"`
	PromptType         string  `json:"prompt_type"`         // what, how, comparison, top_best, brand
	Category           string  `json:"category,omitempty"`
	
	// Performance Metrics
	AvgVisibility      float64 `json:"avg_visibility"`       // Average visibility score (0-10)
	AvgPosition        float64 `json:"avg_position"`         // Average rank when mentioned (1=best)
	MentionRate        float64 `json:"mention_rate"`         // % of responses mentioning brand
	TopPositionRate    float64 `json:"top_position_rate"`    // % of times in top 3 positions
	AvgSentiment       float64 `json:"avg_sentiment"`        // Average sentiment (-1 to +1)
	
	// Volume Metrics
	TotalResponses     int     `json:"total_responses"`      // Total responses for this prompt
	BrandMentions      int     `json:"brand_mentions"`       // Times brand was mentioned
	
	// Effectiveness Metrics
	EffectivenessScore float64 `json:"effectiveness_score"`  // Composite score (0-100)
	EffectivenessGrade string  `json:"effectiveness_grade"`  // A+, A, A-, B+, B, B-, C+, C, C-, D+, D, F
	Status             string  `json:"status"`               // high_performer, average_performer, low_performer
	Recommendation     string  `json:"recommendation"`       // Actionable recommendation
}

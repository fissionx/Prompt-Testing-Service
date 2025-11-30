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
	Brand       string `json:"brand" binding:"required"`
	Website     string `json:"website,omitempty"`
	Category    string `json:"category,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Description string `json:"description,omitempty"`
	Count       int    `json:"count,omitempty"`
}

// GeneratePromptsResponse represents the response with generated prompts
type GeneratePromptsResponse struct {
	Brand         string                     `json:"brand"`
	Category      string                     `json:"category"`
	Domain        string                     `json:"domain"`
	Prompts       []PromptPreview            `json:"prompts"`
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
	Brand        string   `json:"brand" binding:"required"`
	PromptIDs    []string `json:"promptIds" binding:"required"`
	LLMIDs       []string `json:"llmIds" binding:"required"`
	Temperature  float64  `json:"temperature,omitempty"`
}

// BulkExecuteResponse represents the response from bulk execution
type BulkExecuteResponse struct {
	CampaignID   string    `json:"campaignId"`
	CampaignName string    `json:"campaignName"`
	Brand        string    `json:"brand"`
	TotalRuns    int       `json:"totalRuns"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"startedAt"`
	Message      string    `json:"message"`
}

// GEOInsightsRequest represents the request for GEO insights/analytics
type GEOInsightsRequest struct {
	Brand      string     `json:"brand,omitempty"`
	CampaignID string     `json:"campaignId,omitempty"`
	StartTime  *time.Time `json:"startTime,omitempty"`
	EndTime    *time.Time `json:"endTime,omitempty"`
}

// GEOInsightsResponse represents comprehensive GEO analytics
type GEOInsightsResponse struct {
	Brand                 string                `json:"brand"`
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
	Name          string  `json:"name"`
	MentionCount  int     `json:"mentionCount"`
	VisibilityAvg float64 `json:"visibilityAvg"`
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
	Period          string           `json:"period,omitempty"`
	TopSources      []SourceInsight  `json:"topSources"`
	Recommendations []Recommendation `json:"recommendations"`
	TotalSources    int              `json:"totalSources"`
	TotalCitations  int              `json:"totalCitations"`
}

// CompetitiveBenchmarkRequest represents request for competitive analysis
type CompetitiveBenchmarkRequest struct {
	MainBrand   string     `json:"mainBrand" binding:"required"`
	Competitors []string   `json:"competitors,omitempty"`
	PromptIDs   []string   `json:"promptIds,omitempty"`
	LLMIDs      []string   `json:"llmIds,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Region      string     `json:"region,omitempty"`
}

// BrandPerformance represents comprehensive brand performance metrics
type BrandPerformance struct {
	Brand           string  `json:"brand"`
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
	MarketLeader    string                      `json:"marketLeader"`
	YourRank        int                         `json:"yourRank"`
	TotalBrands     int                         `json:"totalBrands"`
	PromptBreakdown []PromptCompetitiveAnalysis `json:"promptBreakdown"`
	Recommendations []Recommendation            `json:"recommendations"`
	AnalyzedAt      time.Time                   `json:"analyzedAt"`
}

// PromptCompetitiveAnalysis shows competitive performance for a specific prompt
type PromptCompetitiveAnalysis struct {
	PromptID             string                    `json:"promptId"`
	PromptText           string                    `json:"promptText"`
	PromptType           string                    `json:"promptType,omitempty"`
	MainBrandResult      PromptBrandResult         `json:"mainBrandResult"`
	CompetitorsMentioned []PromptCompetitorMention `json:"competitorsMentioned"`
	Winner               string                    `json:"winner"`
	TotalBrandsMentioned int                       `json:"totalBrandsMentioned"`
	ExecutedAt           time.Time                 `json:"executedAt"`
}

// PromptBrandResult shows how the main brand performed on a specific prompt
type PromptBrandResult struct {
	Mentioned       bool   `json:"mentioned"`
	VisibilityScore int    `json:"visibilityScore"`
	Position        int    `json:"position"`
	Sentiment       string `json:"sentiment"`
	InSources       bool   `json:"inSources"`
}

// PromptCompetitorMention shows how a competitor appeared in a prompt response
type PromptCompetitorMention struct {
	Brand     string `json:"brand"`
	Mentioned bool   `json:"mentioned"`
}

// SourceAnalyticsRequest represents request for source analytics
type SourceAnalyticsRequest struct {
	Brand     string     `json:"brand" binding:"required"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	TopN      int        `json:"topN,omitempty"`
}

// PositionAnalyticsResponse represents position/ranking analytics
type PositionAnalyticsResponse struct {
	Brand             string             `json:"brand"`
	AveragePosition   float64            `json:"averagePosition"`
	TopPositionRate   float64            `json:"topPositionRate"`
	PositionBreakdown map[string]int     `json:"positionBreakdown"`
	ByPromptType      map[string]float64 `json:"byPromptType"`
	ByLLM             map[string]float64 `json:"byLlm"`
	TotalMentions     int                `json:"totalMentions"`
}

// PromptPerformanceRequest represents request for prompt performance analysis
type PromptPerformanceRequest struct {
	Brand        string     `json:"brand" binding:"required"`
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	MinResponses int        `json:"minResponses,omitempty"`
}

// PromptPerformanceResponse represents prompt performance analysis results
type PromptPerformanceResponse struct {
	Brand                string              `json:"brand"`
	Period               string              `json:"period,omitempty"`
	Prompts              []PromptPerformance `json:"prompts"`
	TopPerformers        []string            `json:"topPerformers"`
	LowPerformers        []string            `json:"lowPerformers"`
	AvgEffectiveness     float64             `json:"avgEffectiveness"`
	TotalPromptsAnalyzed int                 `json:"totalPromptsAnalyzed"`
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

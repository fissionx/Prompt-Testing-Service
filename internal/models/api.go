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
}

// ExecuteResponse represents the response from executing a prompt
type ExecuteResponse struct {
	ResponseID   string       `json:"response_id"`
	PromptID     string       `json:"prompt_id,omitempty"`
	Prompt       string       `json:"prompt"`
	Brand        string       `json:"brand,omitempty"`
	Response     string       `json:"response"`
	GEOAnalysis  *GEOAnalysis `json:"geo_analysis,omitempty"`
	LLMName      string       `json:"llm_name"`
	LLMProvider  string       `json:"llm_provider"`
	LLMModel     string       `json:"llm_model"`
	Temperature  float64      `json:"temperature"`
	TokensUsed   int          `json:"tokens_used"`
	LatencyMs    int64        `json:"latency_ms"`
	CreatedAt    time.Time    `json:"created_at"`
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
	Website     string `json:"website,omitempty"`     // Website URL for content scraping
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
	Prompts       []PromptPreview            `json:"prompts"`                  // All prompts (flat list)
	PromptsByType map[string][]PromptPreview `json:"prompts_by_type"`          // Grouped by type
	Existing      int                        `json:"existing_prompts"`         // How many were reused from DB
	Generated     int                        `json:"generated_prompts"`        // How many were newly generated
	TypeCounts    map[string]int             `json:"type_counts"`              // Count per type
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
	CampaignID    string    `json:"campaign_id"`
	CampaignName  string    `json:"campaign_name"`
	Brand         string    `json:"brand"`
	TotalRuns     int       `json:"total_runs"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
	Message       string    `json:"message"`
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
	Brand                string                  `json:"brand"`
	AverageVisibility    float64                 `json:"average_visibility"`
	MentionRate          float64                 `json:"mention_rate"`          // % of responses mentioning brand
	GroundingRate        float64                 `json:"grounding_rate"`        // % of responses citing brand sources
	SentimentBreakdown   map[string]int          `json:"sentiment_breakdown"`   // positive/neutral/negative counts
	TopCompetitors       []CompetitorInsight     `json:"top_competitors"`
	PerformanceByLLM     []LLMPerformance        `json:"performance_by_llm"`
	PerformanceByCategory []CategoryPerformance  `json:"performance_by_category"`
	Trends               []TrendPoint            `json:"trends,omitempty"`
	TotalResponses       int                     `json:"total_responses"`
}

// CompetitorInsight represents competitor visibility data
type CompetitorInsight struct {
	Name          string  `json:"name"`
	MentionCount  int     `json:"mention_count"`
	VisibilityAvg float64 `json:"visibility_avg"`
}

// LLMPerformance represents brand performance per LLM
type LLMPerformance struct {
	LLMName      string  `json:"llm_name"`
	LLMProvider  string  `json:"llm_provider"`
	Visibility   float64 `json:"visibility"`
	MentionRate  float64 `json:"mention_rate"`
	ResponseCount int    `json:"response_count"`
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

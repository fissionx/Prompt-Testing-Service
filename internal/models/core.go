package models

import (
	"time"
)

// Core domain models

// PromptType represents different categories of questions
type PromptType string

const (
	PromptTypeWhat       PromptType = "what"       // Definitional: "What is GEO?", "What are the benefits?"
	PromptTypeHow        PromptType = "how"        // Instructional: "How to appear in AI search?"
	PromptTypeComparison PromptType = "comparison" // Competitive: "X vs Y", "Which is better?"
	PromptTypeTopBest    PromptType = "top_best"   // List-based: "Best AI tools", "Top platforms"
	PromptTypeBrand      PromptType = "brand"      // Brand-specific: "What does Brand X do?"
	PromptTypeCustom     PromptType = "custom"     // Custom user-created prompts
)

// LLMConfig represents an LLM provider configuration
type LLMConfig struct {
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

// Prompt represents a prompt template
type Prompt struct {
	ID         string     `json:"id" bson:"_id"`
	BrandID    string     `json:"brandId,omitempty" bson:"brand_id,omitempty"` // Brand ID (UUID)
	OrgID      string     `json:"orgId,omitempty" bson:"org_id,omitempty"`     // Organization ID
	Template   string     `json:"template" bson:"template"`
	PromptType PromptType `json:"promptType,omitempty" bson:"prompt_type,omitempty"`
	Tags       []string   `json:"tags,omitempty" bson:"tags,omitempty"`
	Category   string     `json:"category,omitempty" bson:"category,omitempty"`
	Domain     string     `json:"domain,omitempty" bson:"domain,omitempty"`
	Brand      string     `json:"brand,omitempty" bson:"brand,omitempty"` // Brand name (kept for backward compatibility)
	SourceID   string     `json:"sourceId,omitempty" bson:"source_id,omitempty"`
	Generated  bool       `json:"generated" bson:"generated"`
	Enabled    bool       `json:"enabled" bson:"enabled"`
	CreatedAt  time.Time  `json:"createdAt" bson:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" bson:"updated_at"`
}

// Schedule represents a scheduler configuration
type Schedule struct {
	ID          string     `json:"id"`
	BrandID     string     `json:"brandId"`
	Name        string     `json:"name"`
	PromptIDs   []string   `json:"promptIds"`
	LLMIDs      []string   `json:"llmIds"`
	CronExpr    string     `json:"cronExpr"`
	Temperature float64    `json:"temperature,omitempty"`
	Enabled     bool       `json:"enabled"`
	LastRun     *time.Time `json:"lastRun,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Response represents an LLM response to a prompt
type Response struct {
	ID           string                 `json:"id" bson:"_id"`
	PromptID     string                 `json:"promptId" bson:"prompt_id"`
	PromptText   string                 `json:"promptText" bson:"prompt_text"`
	LLMID        string                 `json:"llmId" bson:"llm_id"`
	LLMName      string                 `json:"llmName" bson:"llm_name"`
	LLMProvider  string                 `json:"llmProvider" bson:"llm_provider"`
	LLMModel     string                 `json:"llmModel" bson:"llm_model"`
	ResponseText string                 `json:"responseText" bson:"response_text"`
	Brand        string                 `json:"brand,omitempty" bson:"brand,omitempty"`
	Temperature  float64                `json:"temperature,omitempty" bson:"temperature,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	ScheduleID   string                 `json:"scheduleId,omitempty" bson:"schedule_id,omitempty"`
	TokensUsed   int                    `json:"tokensUsed,omitempty" bson:"tokens_used,omitempty"`
	LatencyMs    int64                  `json:"latencyMs,omitempty" bson:"latency_ms,omitempty"`
	Error        string                 `json:"error,omitempty" bson:"error,omitempty"`

	// GEO Analysis fields
	VisibilityScore    int      `json:"visibilityScore,omitempty" bson:"visibility_score,omitempty"`
	BrandMentioned     bool     `json:"brandMentioned,omitempty" bson:"brand_mentioned,omitempty"`
	InGroundingSources bool     `json:"inGroundingSources,omitempty" bson:"in_grounding_sources,omitempty"`
	GroundingSources   []string `json:"groundingSources,omitempty" bson:"grounding_sources,omitempty"`
	Sentiment          string   `json:"sentiment,omitempty" bson:"sentiment,omitempty"`
	CompetitorsMention []string `json:"competitorsMention,omitempty" bson:"competitors_mention,omitempty"`

	// Position/Ranking tracking
	BrandPosition     int `json:"brandPosition,omitempty" bson:"brand_position,omitempty"`
	TotalBrandsListed int `json:"totalBrandsListed,omitempty" bson:"total_brands_listed,omitempty"`

	// Enhanced source analytics
	GroundingDomains []string `json:"groundingDomains,omitempty" bson:"grounding_domains,omitempty"`

	// Web Search Metadata (for ChatGPT/Gemini-like experience)
	WebSearchQueries []string               `json:"webSearchQueries,omitempty" bson:"web_search_queries,omitempty"` // Search queries used by the model
	WebSearchCalls   []WebSearchCallDetails `json:"webSearchCalls,omitempty" bson:"web_search_calls,omitempty"`     // Detailed search call information
	SearchAnswer     string                 `json:"searchAnswer,omitempty" bson:"search_answer,omitempty"`          // Original search answer before GEO analysis

	// Time-series support
	Week    string `json:"week,omitempty" bson:"week,omitempty"`
	Month   string `json:"month,omitempty" bson:"month,omitempty"`
	Quarter string `json:"quarter,omitempty" bson:"quarter,omitempty"`

	// Regional/Language support
	Region   string `json:"region,omitempty" bson:"region,omitempty"`
	Language string `json:"language,omitempty" bson:"language,omitempty"`

	CreatedAt time.Time `json:"createdAt" bson:"created_at"`
}

// ModelInfo represents information about an available model from a provider
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Brand represents a company/brand for GEO analysis
type Brand struct {
	ID          string    `json:"id" bson:"_id"`
	Name        string    `json:"name" bson:"name"`
	Domain      string    `json:"domain" bson:"domain"`
	Category    string    `json:"category" bson:"category"`
	Website     string    `json:"website,omitempty" bson:"website,omitempty"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	Competitors []string  `json:"competitors,omitempty" bson:"competitors,omitempty"`
	CreatedAt   time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" bson:"updated_at"`
}

// PromptLibrary represents a collection of reusable prompts organized by brand, domain, and category
type PromptLibrary struct {
	ID         string    `json:"id" bson:"_id"`
	Brand      string    `json:"brand" bson:"brand"`
	Domain     string    `json:"domain" bson:"domain"`
	Category   string    `json:"category" bson:"category"`
	PromptIDs  []string  `json:"promptIds" bson:"prompt_ids"`
	UsageCount int       `json:"usageCount" bson:"usage_count"`
	CreatedAt  time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt" bson:"updated_at"`
}

// BrandProfile represents metadata about a brand for better categorization
type BrandProfile struct {
	ID          string    `json:"id" bson:"_id"`
	BrandName   string    `json:"brandName" bson:"brand_name"`
	Domain      string    `json:"domain" bson:"domain"`
	Category    string    `json:"category" bson:"category"`
	Website     string    `json:"website,omitempty" bson:"website,omitempty"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	Competitors []string  `json:"competitors,omitempty" bson:"competitors,omitempty"`
	CreatedAt   time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" bson:"updated_at"`
}

// GEOCampaign represents a GEO analysis campaign for a brand
type GEOCampaign struct {
	ID          string     `json:"id" bson:"_id"`
	Name        string     `json:"name" bson:"name"`
	BrandID     string     `json:"brandId" bson:"brand_id"`
	OrgID       string     `json:"orgId" bson:"org_id"`
	Brand       string     `json:"brand" bson:"brand"`
	PromptIDs   []string   `json:"promptIds" bson:"prompt_ids"`
	LLMIDs      []string   `json:"llmIds" bson:"llm_ids"`
	Status      string     `json:"status" bson:"status"`
	TotalRuns   int        `json:"totalRuns" bson:"total_runs"`
	CompletedAt *time.Time `json:"completedAt,omitempty" bson:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"createdAt" bson:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" bson:"updated_at"`
}

// BrandCompetitors represents a brand's competitor list (both suggested and user-defined)
type BrandCompetitors struct {
	ID            string    `json:"id" bson:"_id"`
	Brand         string    `json:"brand" bson:"brand"`
	BrandID       string    `json:"brandId" bson:"brand_id"`
	OrgID         string    `json:"orgId" bson:"org_id"`                 // Organization ID
	Competitors   []string  `json:"competitors" bson:"competitors"`      // User-defined competitor list
	SuggestedList []string  `json:"suggestedList" bson:"suggested_list"` // Original AI-suggested competitors (cached)
	Source        string    `json:"source" bson:"source"`                // "suggested", "custom", or "mixed"
	CreatedAt     time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updated_at"`
}

// BrandPrompts represents a brand's prompt list (both suggested and active)
type BrandPrompts struct {
	ID                 string    `json:"id" bson:"_id"`
	Brand              string    `json:"brand" bson:"brand"`
	BrandID            string    `json:"brandId" bson:"brand_id"`
	OrgID              string    `json:"orgId" bson:"org_id"`                            // Organization ID
	ActivePromptIDs    []string  `json:"activePromptIds" bson:"active_prompt_ids"`       // Active prompt IDs (enabled prompts)
	SuggestedPromptIDs []string  `json:"suggestedPromptIds" bson:"suggested_prompt_ids"` // Suggested prompt IDs (cached from LLM)
	Source             string    `json:"source" bson:"source"`                           // "suggested", "custom", or "mixed"
	CreatedAt          time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt          time.Time `json:"updatedAt" bson:"updated_at"`
}

// WebSearchCallDetails represents detailed information about a web search call
type WebSearchCallDetails struct {
	Query       string    `json:"query" bson:"query"`              // The search query used
	Status      string    `json:"status" bson:"status"`            // Status: "completed", "in_progress", "failed"
	Sources     []string  `json:"sources" bson:"sources"`          // URLs found in this search
	ResultCount int       `json:"resultCount" bson:"result_count"` // Number of results
	Timestamp   time.Time `json:"timestamp" bson:"timestamp"`      // When the search was performed
}

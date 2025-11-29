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
)

// LLMConfig represents an LLM provider configuration
type LLMConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Provider  string            `json:"provider"` // openai, anthropic, ollama, custom
	Model     string            `json:"model"`
	APIKey    string            `json:"api_key,omitempty"`
	BaseURL   string            `json:"base_url,omitempty"`
	Config    map[string]string `json:"config,omitempty"` // Additional provider-specific config
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Prompt represents a prompt template
type Prompt struct {
	ID         string     `json:"id" bson:"_id"`
	Template   string     `json:"template" bson:"template"`
	PromptType PromptType `json:"prompt_type,omitempty" bson:"prompt_type,omitempty"` // Type of question
	Tags       []string   `json:"tags,omitempty" bson:"tags,omitempty"`
	Category   string     `json:"category,omitempty" bson:"category,omitempty"`     // e.g., "SEO Tools", "CRM Software"
	Domain     string     `json:"domain,omitempty" bson:"domain,omitempty"`         // e.g., "technology", "healthcare"
	Brand      string     `json:"brand,omitempty" bson:"brand,omitempty"`           // Target brand for GEO analysis
	Generated  bool       `json:"generated" bson:"generated"`                       // AI-generated vs manual
	Enabled    bool       `json:"enabled" bson:"enabled"`
	CreatedAt  time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" bson:"updated_at"`
}

// Schedule represents a scheduler configuration
type Schedule struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	PromptIDs   []string   `json:"prompt_ids"`
	LLMIDs      []string   `json:"llm_ids"`
	CronExpr    string     `json:"cron_expr"`             // Cron expression for scheduling
	Temperature float64    `json:"temperature,omitempty"` // Temperature for LLM generation (0-1, default 0.7)
	Enabled     bool       `json:"enabled"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Response represents an LLM response to a prompt
type Response struct {
	ID                 string                 `json:"id" bson:"_id"`
	PromptID           string                 `json:"prompt_id" bson:"prompt_id"`
	PromptText         string                 `json:"prompt_text" bson:"prompt_text"` // Actual prompt sent
	LLMID              string                 `json:"llm_id" bson:"llm_id"`
	LLMName            string                 `json:"llm_name" bson:"llm_name"`
	LLMProvider        string                 `json:"llm_provider" bson:"llm_provider"`
	LLMModel           string                 `json:"llm_model" bson:"llm_model"`
	ResponseText       string                 `json:"response_text" bson:"response_text"`
	Brand              string                 `json:"brand,omitempty" bson:"brand,omitempty"` // Brand being analyzed
	Temperature        float64                `json:"temperature,omitempty" bson:"temperature,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	ScheduleID         string                 `json:"schedule_id,omitempty" bson:"schedule_id,omitempty"`
	TokensUsed         int                    `json:"tokens_used,omitempty" bson:"tokens_used,omitempty"`
	LatencyMs          int64                  `json:"latency_ms,omitempty" bson:"latency_ms,omitempty"`
	Error              string                 `json:"error,omitempty" bson:"error,omitempty"`
	// GEO Analysis fields
	VisibilityScore    int      `json:"visibility_score,omitempty" bson:"visibility_score,omitempty"`
	BrandMentioned     bool     `json:"brand_mentioned,omitempty" bson:"brand_mentioned,omitempty"`
	InGroundingSources bool     `json:"in_grounding_sources,omitempty" bson:"in_grounding_sources,omitempty"`
	GroundingSources   []string `json:"grounding_sources,omitempty" bson:"grounding_sources,omitempty"`
	Sentiment          string   `json:"sentiment,omitempty" bson:"sentiment,omitempty"` // positive, neutral, negative
	CompetitorsMention []string `json:"competitors_mention,omitempty" bson:"competitors_mention,omitempty"`
	CreatedAt          time.Time              `json:"created_at" bson:"created_at"`
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
	Domain      string    `json:"domain" bson:"domain"`                   // e.g., "technology", "healthcare"
	Category    string    `json:"category" bson:"category"`               // e.g., "AI SEO Tools"
	Website     string    `json:"website,omitempty" bson:"website,omitempty"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	Competitors []string  `json:"competitors,omitempty" bson:"competitors,omitempty"` // List of competitor brand names
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// PromptLibrary represents a collection of reusable prompts organized by brand, domain, and category
type PromptLibrary struct {
	ID          string    `json:"id" bson:"_id"`
	Brand       string    `json:"brand" bson:"brand"`             // Brand name (can be generic for reuse)
	Domain      string    `json:"domain" bson:"domain"`           // e.g., "technology", "healthcare"
	Category    string    `json:"category" bson:"category"`       // e.g., "AI SEO Tools", "CRM Software"
	PromptIDs   []string  `json:"prompt_ids" bson:"prompt_ids"`   // References to Prompt documents
	UsageCount  int       `json:"usage_count" bson:"usage_count"` // How many times this library was used
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// BrandProfile represents metadata about a brand for better categorization
type BrandProfile struct {
	ID          string    `json:"id" bson:"_id"`
	BrandName   string    `json:"brand_name" bson:"brand_name"`
	Domain      string    `json:"domain" bson:"domain"`           // Derived domain/industry
	Category    string    `json:"category" bson:"category"`       // Derived category
	Website     string    `json:"website,omitempty" bson:"website,omitempty"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	Competitors []string  `json:"competitors,omitempty" bson:"competitors,omitempty"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// GEOCampaign represents a GEO analysis campaign for a brand
type GEOCampaign struct {
	ID          string    `json:"id" bson:"_id"`
	Name        string    `json:"name" bson:"name"`
	BrandID     string    `json:"brand_id" bson:"brand_id"`
	Brand       string    `json:"brand" bson:"brand"`
	PromptIDs   []string  `json:"prompt_ids" bson:"prompt_ids"`
	LLMIDs      []string  `json:"llm_ids" bson:"llm_ids"`
	Status      string    `json:"status" bson:"status"` // pending, running, completed, failed
	TotalRuns   int       `json:"total_runs" bson:"total_runs"`
	CompletedAt *time.Time `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

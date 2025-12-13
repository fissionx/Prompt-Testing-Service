package models

import (
	"time"
)

// Competitor represents a company competitor with its details
type Competitor struct {
	Name            string  `json:"name"`
	Website         string  `json:"website,omitempty"`
	Domain          string  `json:"domain,omitempty"`
	LogoURL         string  `json:"logoUrl,omitempty"`
	FallbackLogoURL string  `json:"fallbackLogoUrl,omitempty"`
	Description     string  `json:"description,omitempty"`
	Industry        string  `json:"industry,omitempty"`
	MentionCount    int     `json:"mentionCount,omitempty"`
	VisibilityScore float64 `json:"visibilityScore,omitempty"`
	IsCustom        bool    `json:"isCustom,omitempty"` // User-provided competitor
}

// CompetitorInsightMetrics contains detailed performance metrics for a competitor
type CompetitorInsightMetrics struct {
	MentionRate       float64 `json:"mentionRate"`       // Percentage of responses mentioning this competitor
	AvgVisibility     float64 `json:"avgVisibility"`     // Average visibility score when mentioned
	AvgPosition       float64 `json:"avgPosition"`       // Average position in lists
	TopPositionRate   float64 `json:"topPositionRate"`   // Percentage of times in top 3
	SentimentScore    float64 `json:"sentimentScore"`    // Sentiment score (-1 to 1)
	GroundingRate     float64 `json:"groundingRate"`     // Percentage appearing in sources
	MarketShare       float64 `json:"marketShare"`       // Share of voice among all competitors
	MentionCount      int     `json:"mentionCount"`      // Total mention count
	TrendDirection    string  `json:"trendDirection"`    // "up", "down", "stable"
	TrendPercentage   float64 `json:"trendPercentage"`   // Percentage change
	ResponseCount     int     `json:"responseCount"`     // Total responses analyzed
	FirstMentioned    string  `json:"firstMentioned,omitempty"`
	LastMentioned     string  `json:"lastMentioned,omitempty"`
}

// DetailedCompetitorInsight represents detailed insights for a single competitor
type DetailedCompetitorInsight struct {
	Competitor Competitor               `json:"competitor"`
	Metrics    CompetitorInsightMetrics `json:"metrics"`
	Strengths  []string                 `json:"strengths,omitempty"`
	Weaknesses []string                 `json:"weaknesses,omitempty"`
	ByLLM      map[string]float64       `json:"byLlm,omitempty"`      // Visibility by LLM
	ByCategory map[string]float64       `json:"byCategory,omitempty"` // Visibility by category
}

// CompetitorComparison shows head-to-head comparison with main brand
type CompetitorComparison struct {
	Competitor       Competitor `json:"competitor"`
	YourVisibility   float64    `json:"yourVisibility"`
	TheirVisibility  float64    `json:"theirVisibility"`
	VisibilityGap    float64    `json:"visibilityGap"`    // Positive = you're ahead
	YourMentionRate  float64    `json:"yourMentionRate"`
	TheirMentionRate float64    `json:"theirMentionRate"`
	MentionGap       float64    `json:"mentionGap"`
	WinRate          float64    `json:"winRate"`          // Percentage of prompts where you beat them
	CoMentionRate    float64    `json:"coMentionRate"`    // How often you're mentioned together
	Status           string     `json:"status"`           // "leading", "trailing", "even"
}

// ==================== API Request/Response Types ====================

// ListCompetitorsRequest represents the request to list competitors
type ListCompetitorsRequest struct {
	Brand        string   `json:"brand" form:"brand" binding:"required"`
	IncludeCustom bool    `json:"includeCustom" form:"includeCustom"` // Include user-defined competitors
	Limit        int      `json:"limit" form:"limit"`                 // Max competitors to return
}

// ListCompetitorsResponse contains the list of competitors
type ListCompetitorsResponse struct {
	Brand             string       `json:"brand"`
	LogoURL           string       `json:"logoUrl,omitempty"`
	FallbackLogoURL   string       `json:"fallbackLogoUrl,omitempty"`
	Competitors       []Competitor `json:"competitors"`
	TotalDiscovered   int          `json:"totalDiscovered"`
	CustomCompetitors int          `json:"customCompetitors"`
}

// DiscoverCompetitorsRequest represents the request to discover competitors
type DiscoverCompetitorsRequest struct {
	Brand     string     `json:"brand" binding:"required"`
	Website   string     `json:"website,omitempty"`
	Industry  string     `json:"industry,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	Limit     int        `json:"limit,omitempty"`
}

// DiscoverCompetitorsResponse contains discovered competitors
type DiscoverCompetitorsResponse struct {
	Brand           string       `json:"brand"`
	LogoURL         string       `json:"logoUrl,omitempty"`
	FallbackLogoURL string       `json:"fallbackLogoUrl,omitempty"`
	Competitors     []Competitor `json:"competitors"`
	TotalDiscovered int          `json:"totalDiscovered"`
	DiscoveredFrom  int          `json:"discoveredFrom"` // Number of responses analyzed
	AnalyzedAt      time.Time    `json:"analyzedAt"`
}

// AddCustomCompetitorsRequest allows users to add their own competitors
type AddCustomCompetitorsRequest struct {
	Brand       string             `json:"brand" binding:"required"`
	Competitors []CustomCompetitor `json:"competitors" binding:"required,min=1"`
}

// CustomCompetitor represents a user-defined competitor
type CustomCompetitor struct {
	Name        string `json:"name" binding:"required"`
	Website     string `json:"website,omitempty"`
	Description string `json:"description,omitempty"`
	Industry    string `json:"industry,omitempty"`
}

// AddCustomCompetitorsResponse confirms added competitors
type AddCustomCompetitorsResponse struct {
	Brand              string       `json:"brand"`
	AddedCompetitors   []Competitor `json:"addedCompetitors"`
	TotalCompetitors   int          `json:"totalCompetitors"`
}

// CompetitorInsightsRequest represents the request for competitor insights
type CompetitorInsightsRequest struct {
	Brand            string   `json:"brand" binding:"required"`
	Competitors      []string `json:"competitors,omitempty"` // Specific competitors to analyze
	CustomCompetitors []CustomCompetitor `json:"customCompetitors,omitempty"` // User-provided competitors
	IncludeAll       bool     `json:"includeAll,omitempty"`  // Include all discovered competitors
	StartTime        *time.Time `json:"startTime,omitempty"`
	EndTime          *time.Time `json:"endTime,omitempty"`
	PromptIDs        []string `json:"promptIds,omitempty"`
	LLMIDs           []string `json:"llmIds,omitempty"`
}

// CompetitorInsightsResponse contains comprehensive competitor insights
type CompetitorInsightsResponse struct {
	Brand               string                      `json:"brand"`
	LogoURL             string                      `json:"logoUrl,omitempty"`
	FallbackLogoURL     string                      `json:"fallbackLogoUrl,omitempty"`
	Period              string                      `json:"period"`
	YourMetrics         CompetitorInsightMetrics    `json:"yourMetrics"`
	Competitors         []DetailedCompetitorInsight `json:"competitors"`
	HeadToHead          []CompetitorComparison      `json:"headToHead"`
	MarketLeader        string                      `json:"marketLeader"`
	YourRank            int                         `json:"yourRank"`
	TotalBrands         int                         `json:"totalBrands"`
	Recommendations     []Recommendation            `json:"recommendations"`
	StrategicInsights   []StrategicInsight          `json:"strategicInsights"`
	AnalyzedAt          time.Time                   `json:"analyzedAt"`
}

// StrategicInsight represents an actionable strategic insight
type StrategicInsight struct {
	Type        string `json:"type"`        // "opportunity", "threat", "strength", "weakness"
	Priority    string `json:"priority"`    // "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	Competitor  string `json:"competitor,omitempty"` // Related competitor
	Action      string `json:"action"`
	Impact      string `json:"impact"`
}

// CompetitorTrendRequest represents the request for competitor trends
type CompetitorTrendRequest struct {
	Brand       string     `json:"brand" binding:"required"`
	Competitors []string   `json:"competitors,omitempty"`
	Metric      string     `json:"metric,omitempty"` // "visibility", "mentions", "sentiment"
	StartTime   *time.Time `json:"startTime,omitempty"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Granularity string     `json:"granularity,omitempty"` // "daily", "weekly", "monthly"
}

// CompetitorTrendResponse contains trend data for competitors
type CompetitorTrendResponse struct {
	Brand       string              `json:"brand"`
	Period      string              `json:"period"`
	Metric      string              `json:"metric"`
	Granularity string              `json:"granularity"`
	Trends      []CompetitorTrend   `json:"trends"`
	Dates       []string            `json:"dates"`
}

// CompetitorTrend represents trend data for a single competitor
type CompetitorTrend struct {
	Name            string    `json:"name"`
	LogoURL         string    `json:"logoUrl,omitempty"`
	FallbackLogoURL string    `json:"fallbackLogoUrl,omitempty"`
	Values          []float64 `json:"values"`
	IsMainBrand     bool      `json:"isMainBrand"`
	CurrentValue    float64   `json:"currentValue"`
	Change          float64   `json:"change"` // Percentage change
}

// SavedCompetitorList stores user's saved competitor list
type SavedCompetitorList struct {
	ID               string       `json:"id" bson:"_id"`
	Brand            string       `json:"brand" bson:"brand"`
	Competitors      []Competitor `json:"competitors" bson:"competitors"`
	CustomAdded      []Competitor `json:"customAdded" bson:"custom_added"`
	LastDiscoveredAt time.Time    `json:"lastDiscoveredAt" bson:"last_discovered_at"`
	CreatedAt        time.Time    `json:"createdAt" bson:"created_at"`
	UpdatedAt        time.Time    `json:"updatedAt" bson:"updated_at"`
}

// ==================== Auto-Suggest Competitors API ====================

// SuggestCompetitorsRequest represents the request to auto-suggest competitors using LLM
type SuggestCompetitorsRequest struct {
	Brand    string `json:"brand" binding:"required"`
	Website  string `json:"website,omitempty"`
	Industry string `json:"industry,omitempty"`
	Country  string `json:"country,omitempty"` // For regional competitors
	Limit    int    `json:"limit,omitempty"`   // Max competitors to suggest (default: 10)
}

// SuggestCompetitorsResponse contains AI-suggested competitors
type SuggestCompetitorsResponse struct {
	Brand           string                `json:"brand"`
	LogoURL         string                `json:"logoUrl,omitempty"`
	FallbackLogoURL string                `json:"fallbackLogoUrl,omitempty"`
	Industry        string                `json:"industry"`
	Competitors     []SuggestedCompetitor `json:"competitors"`
	TotalSuggested  int                   `json:"totalSuggested"`
	Source          string                `json:"source"` // "llm" or "cache"
	SuggestedAt     time.Time             `json:"suggestedAt"`
}

// SuggestedCompetitor represents an AI-suggested competitor with enriched details
type SuggestedCompetitor struct {
	Name            string `json:"name"`
	Website         string `json:"website"`
	Domain          string `json:"domain"`
	LogoURL         string `json:"logoUrl"`
	FallbackLogoURL string `json:"fallbackLogoUrl"`
	Description     string `json:"description"`
	Reason          string `json:"reason"`     // Why this is a competitor
	Relevance       string `json:"relevance"`  // "direct", "indirect", "emerging"
	MarketPosition  string `json:"marketPosition,omitempty"` // "leader", "challenger", "niche"
}


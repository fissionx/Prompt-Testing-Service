package models

import (
	"time"
)

// OpportunityType represents the category of opportunity identified
type OpportunityType string

const (
	OpportunityTypeContentGap          OpportunityType = "content_gap"
	OpportunityTypeContentFreshness    OpportunityType = "content_freshness"
	OpportunityTypeRedditParticipation OpportunityType = "reddit_participation"
	OpportunityTypeReviewSites         OpportunityType = "review_sites"
	OpportunityTypeCaseStudy           OpportunityType = "case_study"
	OpportunityTypeSEOImprovement      OpportunityType = "seo_improvement"
	OpportunityTypePROpportunity       OpportunityType = "pr_opportunity"
	OpportunityTypeLinkedInPresence    OpportunityType = "linkedin_presence"
	OpportunityTypeComparisonContent   OpportunityType = "comparison_content"
	OpportunityTypeCitationSource      OpportunityType = "citation_source"
	OpportunityTypeFAQContent          OpportunityType = "faq_content"
	OpportunityTypeCompetitorResponse  OpportunityType = "competitor_response"
	OpportunityTypeIntegrationContent  OpportunityType = "integration_content"
	OpportunityTypeOther               OpportunityType = "other"
)

// OpportunityStatus represents the current state of an opportunity
type OpportunityStatus string

const (
	OpportunityStatusNew        OpportunityStatus = "new"
	OpportunityStatusInProgress OpportunityStatus = "in_progress"
	OpportunityStatusCompleted  OpportunityStatus = "completed"
	OpportunityStatusArchived   OpportunityStatus = "archived" // suppressed
)

// Opportunity represents a gap or improvement area identified from AI response analysis
type Opportunity struct {
	ID          string            `json:"id" bson:"_id"`
	OrgID       string            `json:"orgId" bson:"org_id"`
	BrandID     string            `json:"brandId" bson:"brand_id"`
	PromptID    string            `json:"promptId" bson:"prompt_id"`
	ResponseID  string            `json:"responseId" bson:"response_id"`
	Type        OpportunityType   `json:"type" bson:"type"`
	Status      OpportunityStatus `json:"status" bson:"status"`

	// Core opportunity details
	Title       string `json:"title" bson:"title"`             // Short, actionable title
	Description string `json:"description" bson:"description"` // Detailed description of what to do

	// Context and evidence
	CurrentState   string `json:"currentState" bson:"current_state"`     // What the brand currently has/lacks
	SourceEvidence string `json:"sourceEvidence" bson:"source_evidence"` // Specific sources/citations that show the gap

	// Priority and scoring
	ImpactScore    int    `json:"impactScore" bson:"impact_score"`       // 1-100, LLM assigned
	Urgency        string `json:"urgency" bson:"urgency"`                // "high", "medium", "low" - time sensitivity
	EffortEstimate string `json:"effortEstimate" bson:"effort_estimate"` // "low", "medium", "high" - rough effort

	// Internal fields
	ContentHash string                 `json:"contentHash" bson:"content_hash"` // For deduplication
	ActionID    string                 `json:"actionId,omitempty" bson:"action_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"` // Additional flexible data
	CreatedAt   time.Time              `json:"createdAt" bson:"created_at"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updated_at"`
}

// ActionStatus represents the current state of an action
type ActionStatus string

const (
	ActionStatusPending    ActionStatus = "pending"
	ActionStatusInProgress ActionStatus = "in_progress"
	ActionStatusCompleted  ActionStatus = "completed"
)

// ActionStep represents a single step in an action plan
type ActionStep struct {
	Order       int    `json:"order" bson:"order"`
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	Completed   bool   `json:"completed" bson:"completed"`
}

// Action represents a detailed action plan generated from an opportunity
type Action struct {
	ID              string       `json:"id" bson:"_id"`
	OrgID           string       `json:"orgId" bson:"org_id"`
	OpportunityID   string       `json:"opportunityId" bson:"opportunity_id"`
	BrandID         string       `json:"brandId" bson:"brand_id"`
	Title           string       `json:"title" bson:"title"`
	Description     string       `json:"description" bson:"description"`
	Steps           []ActionStep `json:"steps" bson:"steps"`
	EstimatedEffort string       `json:"estimatedEffort" bson:"estimated_effort"` // "low", "medium", "high"
	Resources       []string     `json:"resources,omitempty" bson:"resources,omitempty"`
	Status          ActionStatus `json:"status" bson:"status"`
	CreatedAt       time.Time    `json:"createdAt" bson:"created_at"`
	UpdatedAt       time.Time    `json:"updatedAt" bson:"updated_at"`
}

// LLMOpportunity represents the opportunity structure returned by LLM
type LLMOpportunity struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`

	// Context and evidence
	CurrentState   string `json:"current_state"`   // What brand currently has/lacks
	SourceEvidence string `json:"source_evidence"` // Specific evidence from sources

	// Priority
	ImpactScore    int    `json:"impact_score"`    // 1-100
	Urgency        string `json:"urgency"`         // high, medium, low
	EffortEstimate string `json:"effort_estimate"` // low, medium, high

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LLMActionPlan represents the action plan structure returned by LLM
type LLMActionPlan struct {
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	Steps           []LLMActionStep `json:"steps"`
	EstimatedEffort string         `json:"estimated_effort"`
	Resources       []string       `json:"resources,omitempty"`
}

// LLMActionStep represents an action step from LLM response
type LLMActionStep struct {
	Order       int    `json:"order"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ValidOpportunityTypes returns all valid opportunity types
func ValidOpportunityTypes() []OpportunityType {
	return []OpportunityType{
		OpportunityTypeContentGap,
		OpportunityTypeContentFreshness,
		OpportunityTypeRedditParticipation,
		OpportunityTypeReviewSites,
		OpportunityTypeCaseStudy,
		OpportunityTypeSEOImprovement,
		OpportunityTypePROpportunity,
		OpportunityTypeLinkedInPresence,
		OpportunityTypeComparisonContent,
		OpportunityTypeCitationSource,
		OpportunityTypeFAQContent,
		OpportunityTypeCompetitorResponse,
		OpportunityTypeIntegrationContent,
		OpportunityTypeOther,
	}
}

// IsValidOpportunityType checks if the given type is valid
func IsValidOpportunityType(t string) bool {
	for _, valid := range ValidOpportunityTypes() {
		if string(valid) == t {
			return true
		}
	}
	return false
}

// ParseOpportunityType converts a string to OpportunityType with fallback
func ParseOpportunityType(t string) OpportunityType {
	if IsValidOpportunityType(t) {
		return OpportunityType(t)
	}
	return OpportunityTypeOther
}

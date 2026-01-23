package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/utils"
	"github.com/google/uuid"
)

// OpportunityService handles opportunity-related business logic
type OpportunityService struct {
	db          db.Database
	llmRegistry *llm.Registry
	// In-memory matchers per brand for semantic deduplication
	matchers   map[string]*utils.OpportunityMatcher
	matchersMu sync.RWMutex
}

// NewOpportunityService creates a new OpportunityService
func NewOpportunityService(database db.Database, llmRegistry *llm.Registry) *OpportunityService {
	return &OpportunityService{
		db:          database,
		llmRegistry: llmRegistry,
		matchers:    make(map[string]*utils.OpportunityMatcher),
	}
}

// getOrCreateMatcher gets or creates an opportunity matcher for a brand
func (s *OpportunityService) getOrCreateMatcher(ctx context.Context, brandID string) *utils.OpportunityMatcher {
	s.matchersMu.Lock()
	defer s.matchersMu.Unlock()

	if matcher, exists := s.matchers[brandID]; exists {
		return matcher
	}

	// Create new matcher with 0.75 similarity threshold
	matcher := utils.NewOpportunityMatcher(0.75)

	// Pre-populate with existing opportunities for this brand
	filter := models.OpportunityFilter{
		BrandID: brandID,
		Limit:   200, // Load recent opportunities
	}
	existingOpps, err := s.db.ListOpportunities(ctx, filter)
	if err == nil {
		for _, opp := range existingOpps {
			if opp.Status != models.OpportunityStatusArchived {
				matcher.AddOpportunity(opp.ID, string(opp.Type), opp.Title, opp.Description)
			}
		}
	}

	s.matchers[brandID] = matcher
	return matcher
}

// GEOAnalysisWithOpportunitiesResult represents the combined GEO analysis and opportunities result
type GEOAnalysisWithOpportunitiesResult struct {
	SearchAnswer  string                  `json:"search_answer"`
	GEOAnalysis   *OpportunityGEOAnalysis `json:"geo_analysis"`
	Opportunities []models.LLMOpportunity `json:"opportunities"`
}

// OpportunityGEOAnalysis represents the GEO analysis portion of the opportunity response
type OpportunityGEOAnalysis struct {
	VisibilityScore    int      `json:"visibility_score"`
	BrandMentioned     bool     `json:"brand_mentioned"`
	InGroundingSources bool     `json:"in_grounding_sources"`
	MentionStatus      string   `json:"mention_status"`
	Reason             string   `json:"reason"`
	Sentiment          string   `json:"sentiment"`
	Competitors        []string `json:"competitors"`
	Insights           []string `json:"insights"`
	Actions            []string `json:"actions"`
	CompetitorInfo     string   `json:"competitor_info"`
}

// AnalyzeAndGenerateOpportunities performs GEO analysis and generates opportunities in a single LLM call
// Uses the same LLM provider that was used for executing the prompt
func (s *OpportunityService) AnalyzeAndGenerateOpportunities(
	ctx context.Context,
	orgID string,
	brandID string,
	brandName string,
	promptID string,
	responseID string,
	searchQuery string,
	searchAnswer string,
	sourcesInfo string,
	competitors []string,
	llmProvider llm.Provider,
	llmModel string,
) (*GEOAnalysisWithOpportunitiesResult, []*models.Opportunity, error) {
	fmt.Printf("🎯 AnalyzeAndGenerateOpportunities called for brand: %s, prompt: %s\n", brandName, promptID)
	fmt.Printf("✅ Using same LLM model: %s for opportunity analysis\n", llmModel)

	// Use the provided LLM provider (same one that executed the prompt)
	provider := llmProvider
	model := llmModel

	// Generate the combined prompt
	prompt := utils.GEOAnalysisWithOpportunitiesPrompt(brandName, searchQuery, searchAnswer, sourcesInfo, competitors)

	// Call LLM
	fmt.Printf("📤 Calling LLM for opportunity analysis (prompt length: %d chars)\n", len(prompt))
	response, err := provider.Generate(ctx, prompt, llm.Config{
		Model:       model,
		Temperature: 0.3, // Lower temperature for more consistent analysis
		MaxTokens:   4096,
	})
	if err != nil {
		fmt.Printf("❌ LLM call failed: %v\n", err)
		return nil, nil, fmt.Errorf("LLM analysis failed: %w", err)
	}
	fmt.Printf("📥 LLM response received (length: %d chars)\n", len(response.Text))

	// Parse the response
	result, err := s.parseGEOAnalysisWithOpportunities(response.Text)
	if err != nil {
		fmt.Printf("❌ Failed to parse LLM response: %v\n", err)
		fmt.Printf("📋 Raw response (first 500 chars): %s\n", truncateString(response.Text, 500))
		return nil, nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	fmt.Printf("✅ Parsed response - found %d opportunities\n", len(result.Opportunities))

	// Convert LLM opportunities to database models with deduplication
	opportunities, err := s.processOpportunities(ctx, orgID, brandID, promptID, responseID, result.Opportunities)
	if err != nil {
		fmt.Printf("❌ Failed to process opportunities: %v\n", err)
		return nil, nil, fmt.Errorf("failed to process opportunities: %w", err)
	}
	fmt.Printf("✅ Saved %d opportunities to database\n", len(opportunities))

	return result, opportunities, nil
}

// parseGEOAnalysisWithOpportunities parses the LLM response into structured data
func (s *OpportunityService) parseGEOAnalysisWithOpportunities(responseText string) (*GEOAnalysisWithOpportunitiesResult, error) {
	// Clean up the response
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Find the JSON object
	startIdx := strings.Index(responseText, "{")
	if startIdx == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	// Find matching closing brace
	braceCount := 0
	endIdx := -1
	inString := false
	escapeNext := false

	for i := startIdx; i < len(responseText); i++ {
		char := responseText[i]

		if escapeNext {
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					endIdx = i
					break
				}
			}
		}
	}

	if endIdx == -1 {
		return nil, fmt.Errorf("malformed JSON in response")
	}

	jsonStr := responseText[startIdx : endIdx+1]

	var result GEOAnalysisWithOpportunitiesResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}

// processOpportunities converts LLM opportunities to database models with semantic deduplication
func (s *OpportunityService) processOpportunities(
	ctx context.Context,
	orgID string,
	brandID string,
	promptID string,
	responseID string,
	llmOpportunities []models.LLMOpportunity,
) ([]*models.Opportunity, error) {
	var opportunities []*models.Opportunity

	// Get or create in-memory matcher for this brand (pre-populated with existing opportunities)
	matcher := s.getOrCreateMatcher(ctx, brandID)

	for _, llmOpp := range llmOpportunities {
		// Generate content hash for fast deduplication
		contentHash := s.generateContentHash(llmOpp.Type, llmOpp.Title, promptID)

		// Fast path: Check if opportunity already exists by hash (same prompt)
		existing, err := s.db.GetOpportunityByContentHash(ctx, promptID, contentHash)
		if err != nil {
			fmt.Printf("Warning: failed to check for existing opportunity by hash: %v\n", err)
		}

		if existing != nil {
			// Skip if already exists and not new (archived opportunities should stay archived)
			if existing.Status != models.OpportunityStatusNew {
				continue
			}
			// Update existing new opportunity with fresh data
			existing.ResponseID = responseID
			existing.ImpactScore = llmOpp.ImpactScore
			existing.UpdatedAt = time.Now()
			if err := s.db.UpdateOpportunity(ctx, existing); err != nil {
				fmt.Printf("Warning: failed to update existing opportunity: %v\n", err)
			}
			opportunities = append(opportunities, existing)
			continue
		}

		// Semantic deduplication using in-memory embeddings (no LLM calls)
		isDuplicate, similarID, similarity := matcher.FindDuplicate(llmOpp.Type, llmOpp.Title, llmOpp.Description)
		if isDuplicate {
			fmt.Printf("Skipping duplicate opportunity (%.2f similar to %s): %s\n", similarity, similarID, llmOpp.Title)
			continue
		}

		// Create new opportunity
		opportunity := &models.Opportunity{
			ID:         uuid.New().String(),
			OrgID:      orgID,
			BrandID:    brandID,
			PromptID:   promptID,
			ResponseID: responseID,
			Type:       models.ParseOpportunityType(llmOpp.Type),
			Status:     models.OpportunityStatusNew,

			// Core details
			Title:       llmOpp.Title,
			Description: llmOpp.Description,

			// Context and evidence
			CurrentState:   llmOpp.CurrentState,
			SourceEvidence: llmOpp.SourceEvidence,

			// Priority and scoring
			ImpactScore:    llmOpp.ImpactScore,
			Urgency:        llmOpp.Urgency,
			EffortEstimate: llmOpp.EffortEstimate,

			// Internal fields
			ContentHash: contentHash,
			Metadata:    llmOpp.Metadata,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := s.db.CreateOpportunity(ctx, opportunity); err != nil {
			// If it fails due to unique constraint, skip silently
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				continue
			}
			return nil, fmt.Errorf("failed to create opportunity: %w", err)
		}

		opportunities = append(opportunities, opportunity)

		// Add to matcher for subsequent dedup checks in this batch
		matcher.AddOpportunity(opportunity.ID, string(opportunity.Type), opportunity.Title, opportunity.Description)
	}

	return opportunities, nil
}

// generateContentHash creates a hash for deduplication
func (s *OpportunityService) generateContentHash(oppType, title, promptID string) string {
	// Normalize the input
	normalized := strings.ToLower(strings.TrimSpace(oppType + "|" + title))
	// Remove extra spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes (32 hex chars)
}

// ConvertToAction generates a detailed action plan for an opportunity
func (s *OpportunityService) ConvertToAction(
	ctx context.Context,
	opportunityID string,
	additionalContext string,
) (*models.Action, error) {
	// Get the opportunity
	opportunity, err := s.db.GetOpportunity(ctx, opportunityID)
	if err != nil {
		return nil, fmt.Errorf("opportunity not found: %w", err)
	}

	// Check if action already exists
	existingAction, err := s.db.GetActionByOpportunityID(ctx, opportunityID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing action: %w", err)
	}
	if existingAction != nil {
		return existingAction, nil // Return existing action
	}

	// Get brand name for context
	brandName := opportunity.BrandID // Default to ID if name not available

	// Get the best available LLM
	provider, model, err := s.getBestLLM(ctx)
	if err != nil {
		return nil, fmt.Errorf("no LLM available: %w", err)
	}

	// Generate action plan prompt
	prompt := utils.ActionGenerationPrompt(
		brandName,
		opportunity.Title,
		opportunity.Description,
		string(opportunity.Type),
		opportunity.Metadata,
		additionalContext,
	)

	// Call LLM
	response, err := provider.Generate(ctx, prompt, llm.Config{
		Model:       model,
		Temperature: 0.4,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM action generation failed: %w", err)
	}

	// Parse the response
	actionPlan, err := s.parseActionPlan(response.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse action plan: %w", err)
	}

	// Create action in database
	action := &models.Action{
		ID:              uuid.New().String(),
		OrgID:           opportunity.OrgID,
		OpportunityID:   opportunityID,
		BrandID:         opportunity.BrandID,
		Title:           actionPlan.Title,
		Description:     actionPlan.Description,
		Steps:           convertLLMSteps(actionPlan.Steps),
		EstimatedEffort: actionPlan.EstimatedEffort,
		Resources:       actionPlan.Resources,
		Status:          models.ActionStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.db.CreateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	// Update opportunity with action ID and status
	opportunity.ActionID = action.ID
	opportunity.Status = models.OpportunityStatusInProgress
	opportunity.UpdatedAt = time.Now()
	if err := s.db.UpdateOpportunity(ctx, opportunity); err != nil {
		return nil, fmt.Errorf("failed to update opportunity: %w", err)
	}

	return action, nil
}

// parseActionPlan parses the LLM response into an action plan
func (s *OpportunityService) parseActionPlan(responseText string) (*models.LLMActionPlan, error) {
	// Clean up the response
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Find the JSON object
	startIdx := strings.Index(responseText, "{")
	if startIdx == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	endIdx := strings.LastIndex(responseText, "}")
	if endIdx == -1 {
		return nil, fmt.Errorf("malformed JSON in response")
	}

	jsonStr := responseText[startIdx : endIdx+1]

	var result models.LLMActionPlan
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}

// convertLLMSteps converts LLM action steps to model action steps
func convertLLMSteps(llmSteps []models.LLMActionStep) []models.ActionStep {
	steps := make([]models.ActionStep, len(llmSteps))
	for i, s := range llmSteps {
		steps[i] = models.ActionStep{
			Order:       s.Order,
			Title:       s.Title,
			Description: s.Description,
			Completed:   false,
		}
	}
	return steps
}

// SuppressOpportunity archives/suppresses an opportunity
func (s *OpportunityService) SuppressOpportunity(ctx context.Context, opportunityID string) error {
	opportunity, err := s.db.GetOpportunity(ctx, opportunityID)
	if err != nil {
		return fmt.Errorf("opportunity not found: %w", err)
	}

	opportunity.Status = models.OpportunityStatusArchived
	opportunity.UpdatedAt = time.Now()

	return s.db.UpdateOpportunity(ctx, opportunity)
}

// ListOpportunities retrieves opportunities with filtering
func (s *OpportunityService) ListOpportunities(ctx context.Context, filter models.OpportunityFilter) ([]*models.Opportunity, int64, error) {
	opportunities, err := s.db.ListOpportunities(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count without pagination
	countFilter := filter
	countFilter.Limit = 0
	countFilter.Offset = 0
	total, err := s.db.CountOpportunities(ctx, countFilter)
	if err != nil {
		return nil, 0, err
	}

	return opportunities, total, nil
}

// GetOpportunityWithDetails retrieves an opportunity with its action and prompt details
func (s *OpportunityService) GetOpportunityWithDetails(ctx context.Context, opportunityID string) (*models.Opportunity, *models.Action, *models.Prompt, error) {
	opportunity, err := s.db.GetOpportunity(ctx, opportunityID)
	if err != nil {
		return nil, nil, nil, err
	}

	var action *models.Action
	if opportunity.ActionID != "" {
		action, _ = s.db.GetAction(ctx, opportunity.ActionID)
	}

	var prompt *models.Prompt
	if opportunity.PromptID != "" {
		prompt, _ = s.db.GetPrompt(ctx, opportunity.PromptID)
	}

	return opportunity, action, prompt, nil
}

// ListActions retrieves actions with filtering
func (s *OpportunityService) ListActions(ctx context.Context, filter models.ActionFilter) ([]*models.Action, int64, error) {
	actions, err := s.db.ListActions(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count without pagination
	countFilter := filter
	countFilter.Limit = 0
	countFilter.Offset = 0
	total, err := s.db.CountActions(ctx, countFilter)
	if err != nil {
		return nil, 0, err
	}

	return actions, total, nil
}

// UpdateActionStatus updates the status of an action and optionally marks steps as completed
func (s *OpportunityService) UpdateActionStatus(ctx context.Context, actionID string, status *models.ActionStatus, completedStep *int) (*models.Action, error) {
	action, err := s.db.GetAction(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if status != nil {
		action.Status = *status
	}

	if completedStep != nil && *completedStep > 0 && *completedStep <= len(action.Steps) {
		action.Steps[*completedStep-1].Completed = true
	}

	action.UpdatedAt = time.Now()

	if err := s.db.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to update action: %w", err)
	}

	// If action is completed, update the opportunity status
	if status != nil && *status == models.ActionStatusCompleted {
		opportunity, err := s.db.GetOpportunity(ctx, action.OpportunityID)
		if err == nil {
			opportunity.Status = models.OpportunityStatusCompleted
			opportunity.UpdatedAt = time.Now()
			s.db.UpdateOpportunity(ctx, opportunity)
		}
	}

	return action, nil
}

// GetOpportunitySummary returns a summary of opportunities for a brand
func (s *OpportunityService) GetOpportunitySummary(ctx context.Context, brandID string) (*models.OpportunitySummary, error) {
	return s.db.GetOpportunitySummary(ctx, brandID)
}

// getBestLLM returns the best available LLM provider and model for analysis
func (s *OpportunityService) getBestLLM(ctx context.Context) (llm.Provider, string, error) {
	// Priority order: Google (Gemini), OpenAI (GPT-4), Anthropic (Claude)
	preferredProviders := []string{"google", "openai", "anthropic", "perplexity"}

	enabled := true
	llmConfigs, err := s.db.ListLLMs(ctx, &enabled)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list LLMs: %w", err)
	}

	if len(llmConfigs) == 0 {
		return nil, "", fmt.Errorf("no enabled LLMs configured")
	}

	// Find best LLM by provider preference
	var selectedConfig *models.LLMConfig
	for _, preferredProvider := range preferredProviders {
		for _, config := range llmConfigs {
			if strings.ToLower(config.Provider) == preferredProvider {
				selectedConfig = config
				break
			}
		}
		if selectedConfig != nil {
			break
		}
	}

	// If no preferred provider found, use first available
	if selectedConfig == nil {
		selectedConfig = llmConfigs[0]
	}

	provider, ok := s.llmRegistry.Get(selectedConfig.Provider)
	if !ok {
		return nil, "", fmt.Errorf("LLM provider not found: %s", selectedConfig.Provider)
	}

	return provider, selectedConfig.Model, nil
}

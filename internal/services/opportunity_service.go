package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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
}

// NewOpportunityService creates a new OpportunityService
func NewOpportunityService(database db.Database, llmRegistry *llm.Registry) *OpportunityService {
	return &OpportunityService{
		db:          database,
		llmRegistry: llmRegistry,
	}
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
	llmID string,
	llmModel string,
) (*GEOAnalysisWithOpportunitiesResult, []*models.Opportunity, error) {
	fmt.Printf("🎯 AnalyzeAndGenerateOpportunities called for brand: %s, prompt: %s\n", brandName, promptID)
	fmt.Printf("✅ Using same LLM model: %s (ID: %s) for opportunity analysis\n", llmModel, llmID)

	// Use the provided LLM provider (same one that executed the prompt)
	provider := llmProvider
	model := llmModel

	// === LLM-BASED DEDUPLICATION ===
	// Fetch existing opportunities for this brand to pass to LLM for deduplication
	existingOpps := s.getExistingOpportunitiesForDedup(ctx, brandID)
	fmt.Printf("📊 Found %d existing opportunities to pass for LLM deduplication\n", len(existingOpps))

	// Generate the combined prompt WITH existing opportunities for deduplication
	prompt := utils.GEOAnalysisWithOpportunitiesPromptWithDedup(
		brandName, searchQuery, searchAnswer, sourcesInfo, competitors, existingOpps,
	)

	// Call LLM
	fmt.Printf("📤 Calling LLM for opportunity analysis with deduplication (prompt length: %d chars)\n", len(prompt))
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
		fmt.Printf("📋 Raw response (first 500 chars): %s\n", truncateForLog(response.Text, 500))
		return nil, nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	fmt.Printf("✅ Parsed response - found %d NEW opportunities (LLM filtered duplicates)\n", len(result.Opportunities))

	// Convert LLM opportunities to database models (minimal dedup since LLM already filtered)
	opportunities, err := s.processOpportunities(ctx, orgID, brandID, promptID, responseID, llmID, result.Opportunities)
	if err != nil {
		fmt.Printf("❌ Failed to process opportunities: %v\n", err)
		return nil, nil, fmt.Errorf("failed to process opportunities: %w", err)
	}
	fmt.Printf("✅ Saved %d opportunities to database\n", len(opportunities))

	return result, opportunities, nil
}

// getExistingOpportunitiesForDedup fetches existing opportunities for LLM-based deduplication
func (s *OpportunityService) getExistingOpportunitiesForDedup(ctx context.Context, brandID string) []utils.ExistingOpportunity {
	// Fetch all non-archived opportunities for this brand
	filter := models.OpportunityFilter{
		BrandID: brandID,
		Limit:   100, // Limit to avoid prompt getting too large
	}

	opps, err := s.db.ListOpportunities(ctx, filter)
	if err != nil {
		fmt.Printf("⚠️ Warning: could not fetch existing opportunities for dedup: %v\n", err)
		return nil
	}

	var existing []utils.ExistingOpportunity
	for _, opp := range opps {
		// Skip archived opportunities
		if opp.IsArchived {
			continue
		}
		existing = append(existing, utils.ExistingOpportunity{
			Title:       opp.Title,
			Type:        string(opp.Type),
			Description: opp.Description,
		})
	}

	return existing
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

// processOpportunities converts LLM opportunities to database models
// RULE: Only ONE opportunity per type per prompt
// - If type exists with no action_id → UPDATE with new data
// - If type exists with action_id → skip (action in progress)
// - If type doesn't exist → CREATE new
func (s *OpportunityService) processOpportunities(
	ctx context.Context,
	orgID string,
	brandID string,
	promptID string,
	responseID string,
	llmID string,
	llmOpportunities []models.LLMOpportunity,
) ([]*models.Opportunity, error) {
	var opportunities []*models.Opportunity

	fmt.Printf("🔍 Processing %d opportunities from LLM (one per type per prompt)\n", len(llmOpportunities))

	for i, llmOpp := range llmOpportunities {
		oppType := models.ParseOpportunityType(llmOpp.Type)

		// Check if an opportunity of this type already exists for this prompt
		existing, err := s.db.GetOpportunityByPromptAndType(ctx, promptID, string(oppType))
		if err != nil {
			fmt.Printf("Warning: failed to check for existing opportunity by type: %v\n", err)
		}

		if existing != nil {
			// Type already exists for this prompt
			if existing.ActionID != "" {
				// Has action associated - don't overwrite, skip
				fmt.Printf("  [%d] Skipping (action in progress): type=%s, title=%s\n", i+1, llmOpp.Type, existing.Title)
				opportunities = append(opportunities, existing)
				continue
			}

			// No action associated - UPDATE with new data
			existing.ResponseID = responseID
			existing.LLMID = llmID
			existing.Title = llmOpp.Title
			existing.Description = llmOpp.Description
			existing.CurrentState = llmOpp.CurrentState
			existing.SourceEvidence = llmOpp.SourceEvidence
			existing.ImpactScore = llmOpp.ImpactScore
			existing.Urgency = llmOpp.Urgency
			existing.EffortEstimate = llmOpp.EffortEstimate
			existing.ContentHash = s.generateContentHash(llmOpp.Type, llmOpp.Title, promptID)
			existing.UpdatedAt = time.Now()

			if err := s.db.UpdateOpportunity(ctx, existing); err != nil {
				fmt.Printf("Warning: failed to update existing opportunity: %v\n", err)
			} else {
				fmt.Printf("  [%d] 🔄 UPDATED: type=%s, title=%s\n", i+1, llmOpp.Type, llmOpp.Title)
			}
			opportunities = append(opportunities, existing)
			continue
		}

		// Type doesn't exist for this prompt - CREATE new
		contentHash := s.generateContentHash(llmOpp.Type, llmOpp.Title, promptID)
		opportunity := &models.Opportunity{
			ID:         uuid.New().String(),
			OrgID:      orgID,
			BrandID:    brandID,
			PromptID:   promptID,
			ResponseID: responseID,
			LLMID:      llmID, // Store the LLM ID used for analysis
			Type:       oppType,
			Status:     models.OpportunityStatusOpen, // New opportunities start as "open"

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
				fmt.Printf("  [%d] Skipping (DB unique constraint): type=%s\n", i+1, llmOpp.Type)
				continue
			}
			return nil, fmt.Errorf("failed to create opportunity: %w", err)
		}

		fmt.Printf("  [%d] ✅ CREATED: type=%s, title=%s, impact=%d\n", i+1, llmOpp.Type, llmOpp.Title, llmOpp.ImpactScore)
		opportunities = append(opportunities, opportunity)
	}

	fmt.Printf("📊 Result: %d opportunities processed\n", len(opportunities))

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
// Uses the LLM stored in the opportunity (from when it was analyzed) for action generation
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

	// Get LLM provider from stored LLM ID (use the same LLM that analyzed this opportunity)
	var provider llm.Provider
	var model string

	if opportunity.LLMID != "" {
		// Use the stored LLM ID
		llmConfig, err := s.db.GetLLM(ctx, opportunity.LLMID)
		if err != nil || llmConfig == nil || !llmConfig.Enabled {
			fmt.Printf("⚠️ Stored LLM ID %s not found or disabled, falling back to best available\n", opportunity.LLMID)
			provider, model, err = s.getBestLLM(ctx)
			if err != nil {
				return nil, fmt.Errorf("no LLM available: %w", err)
			}
		} else {
			provider, _ = s.llmRegistry.Get(llmConfig.Provider)
			if provider == nil {
				fmt.Printf("⚠️ Provider %s not found, falling back to best available\n", llmConfig.Provider)
				provider, model, err = s.getBestLLM(ctx)
				if err != nil {
					return nil, fmt.Errorf("no LLM available: %w", err)
				}
			} else {
				model = llmConfig.Model
				fmt.Printf("✅ Using stored LLM: provider=%s, model=%s (ID: %s)\n", llmConfig.Provider, model, opportunity.LLMID)
			}
		}
	} else {
		// No stored LLM ID, fall back to brand's schedule or best available
		fmt.Printf("⚠️ No LLM ID stored in opportunity, falling back to brand schedule or best available\n")
		provider, model, err = s.getLLMForBrand(ctx, opportunity.BrandID)
		if err != nil {
			provider, model, err = s.getBestLLM(ctx)
			if err != nil {
				return nil, fmt.Errorf("no LLM available: %w", err)
			}
		}
	}
	fmt.Printf("✅ Using LLM model: %s for action generation\n", model)

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

	// Create action with "preparing" status first
	actionID := uuid.New().String()
	action := &models.Action{
		ID:            actionID,
		OrgID:         opportunity.OrgID,
		OpportunityID: opportunityID,
		BrandID:       opportunity.BrandID,
		Status:        models.ActionStatusPreparing,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save action with preparing status
	if err := s.db.CreateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	// Parse the response
	actionPlan, err := s.parseActionPlan(response.Text)
	if err != nil {
		// If parsing fails, update action with error status or keep as preparing
		// For now, we'll leave it as preparing and let the user retry
		return action, fmt.Errorf("failed to parse action plan: %w", err)
	}

	// Determine action type and execution mode
	actionType := models.ActionTypeOther
	if actionPlan.ActionType != "" {
		switch strings.ToUpper(actionPlan.ActionType) {
		case "CONTENT_CREATION":
			actionType = models.ActionTypeContentCreation
		case "SEO":
			actionType = models.ActionTypeSEO
		case "PR":
			actionType = models.ActionTypePR
		case "SOCIAL":
			actionType = models.ActionTypeSocial
		default:
			actionType = models.ActionTypeOther
		}
	}

	executionMode := models.ExecutionModeManual
	if actionPlan.ExecutionMode != "" {
		switch strings.ToLower(actionPlan.ExecutionMode) {
		case "manual_copy":
			executionMode = models.ExecutionModeManualCopy
		case "api":
			executionMode = models.ExecutionModeAPI
		case "manual":
			executionMode = models.ExecutionModeManual
		default:
			executionMode = models.ExecutionModeManual
		}
	}

	// Use summary if available, otherwise use description
	summary := actionPlan.Summary
	if summary == "" {
		summary = actionPlan.Description
	}

	// Determine priority, effort, expected_impact
	priority := "medium"
	if actionPlan.Priority != "" {
		priority = strings.ToLower(actionPlan.Priority)
	}

	effort := actionPlan.Effort
	if effort == "" {
		effort = actionPlan.EstimatedEffort
	}
	if effort == "" {
		effort = "medium"
	}

	expectedImpact := "medium"
	if actionPlan.ExpectedImpact != "" {
		expectedImpact = strings.ToLower(actionPlan.ExpectedImpact)
	}

	// Update action with full details
	action.ActionType = actionType
	action.ExecutionMode = executionMode
	action.Title = actionPlan.Title
	action.Summary = summary
	action.Description = actionPlan.Description // Keep for backward compatibility
	action.Priority = priority
	action.Effort = effort
	action.ExpectedImpact = expectedImpact
	action.Assets = convertLLMAssets(actionPlan.Assets)
	action.Steps = convertLLMSteps(actionPlan.Steps)
	action.SuccessCriteria = actionPlan.SuccessCriteria
	action.EstimatedEffort = actionPlan.EstimatedEffort // Keep for backward compatibility
	action.Resources = actionPlan.Resources
	action.Status = models.ActionStatusReady // Mark as ready after successful generation
	action.UpdatedAt = time.Now()

	if err := s.db.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to update action: %w", err)
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

// BatchConvertToActions converts multiple opportunities to actions in a single batch operation
// Uses the LLM stored in each opportunity for action generation
// Returns results for each opportunity including successes and failures
func (s *OpportunityService) BatchConvertToActions(
	ctx context.Context,
	brandID string,
	opportunityIDs []string,
	additionalContext string,
) (*models.BatchConvertToActionResponse, error) {
	results := make([]models.BatchConvertResult, 0, len(opportunityIDs))
	successCount := 0
	failureCount := 0

	for _, oppID := range opportunityIDs {
		result := models.BatchConvertResult{
			OpportunityID: oppID,
		}

		// Verify the opportunity belongs to the brand
		opportunity, err := s.db.GetOpportunity(ctx, oppID)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("opportunity not found: %v", err)
			results = append(results, result)
			failureCount++
			continue
		}

		if opportunity.BrandID != brandID {
			result.Success = false
			result.Error = "opportunity does not belong to this brand"
			results = append(results, result)
			failureCount++
			continue
		}

		// Check if already archived
		if opportunity.IsArchived {
			result.Success = false
			result.Error = "opportunity is archived"
			results = append(results, result)
			failureCount++
			continue
		}

		// Convert the opportunity to action (uses stored LLM ID)
		action, err := s.ConvertToAction(ctx, oppID, additionalContext)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			results = append(results, result)
			failureCount++
			continue
		}

		result.Success = true
		result.Action = action
		results = append(results, result)
		successCount++
	}

	response := &models.BatchConvertToActionResponse{
		Results:      results,
		TotalCount:   len(opportunityIDs),
		SuccessCount: successCount,
		FailureCount: failureCount,
		Message:      fmt.Sprintf("Batch conversion completed: %d succeeded, %d failed", successCount, failureCount),
	}

	return response, nil
}

// BatchSuppressOpportunities archives/suppresses multiple opportunities in a single batch operation
// Returns results for each opportunity including successes and failures
func (s *OpportunityService) BatchSuppressOpportunities(
	ctx context.Context,
	brandID string,
	opportunityIDs []string,
) (*models.BatchSuppressOpportunityResponse, error) {
	results := make([]models.BatchSuppressResult, 0, len(opportunityIDs))
	successCount := 0
	failureCount := 0

	for _, oppID := range opportunityIDs {
		result := models.BatchSuppressResult{
			OpportunityID: oppID,
		}

		// Verify the opportunity belongs to the brand
		opportunity, err := s.db.GetOpportunity(ctx, oppID)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("opportunity not found: %v", err)
			results = append(results, result)
			failureCount++
			continue
		}

		if opportunity.BrandID != brandID {
			result.Success = false
			result.Error = "opportunity does not belong to this brand"
			results = append(results, result)
			failureCount++
			continue
		}

		// Check if already archived
		if opportunity.IsArchived {
			result.Success = true
			result.IsArchived = true
			result.SuppressedAt = opportunity.UpdatedAt
			result.Error = "opportunity was already archived"
			results = append(results, result)
			successCount++ // Count as success since the desired state is achieved
			continue
		}

		// Suppress the opportunity
		if err := s.SuppressOpportunity(ctx, oppID); err != nil {
			result.Success = false
			result.Error = err.Error()
			results = append(results, result)
			failureCount++
			continue
		}

		result.Success = true
		result.IsArchived = true
		result.SuppressedAt = time.Now()
		results = append(results, result)
		successCount++
	}

	response := &models.BatchSuppressOpportunityResponse{
		Results:      results,
		TotalCount:   len(opportunityIDs),
		SuccessCount: successCount,
		FailureCount: failureCount,
		Message:      fmt.Sprintf("Batch suppression completed: %d succeeded, %d failed", successCount, failureCount),
	}

	return response, nil
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
			Instruction: s.Instruction,
			Completed:   false,
		}
	}
	return steps
}

// convertLLMAssets converts LLM action assets to model action assets
func convertLLMAssets(llmAssets []models.LLMActionAsset) []models.ActionAsset {
	assets := make([]models.ActionAsset, len(llmAssets))
	for i, a := range llmAssets {
		assets[i] = models.ActionAsset{
			AssetType: a.AssetType,
			Role:      a.Role,
			Title:     a.Title,
			Content:   a.Content,
		}
	}
	return assets
}

// SuppressOpportunity archives/suppresses an opportunity by setting IsArchived = true
func (s *OpportunityService) SuppressOpportunity(ctx context.Context, opportunityID string) error {
	opportunity, err := s.db.GetOpportunity(ctx, opportunityID)
	if err != nil {
		return fmt.Errorf("opportunity not found: %w", err)
	}

	opportunity.IsArchived = true
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

// getLLMForBrand returns the LLM provider and model configured for a brand (from schedule)
func (s *OpportunityService) getLLMForBrand(ctx context.Context, brandID string) (llm.Provider, string, error) {
	// Get schedules for this brand to find the LLM config
	enabled := true
	schedules, err := s.db.ListSchedules(ctx, brandID, &enabled)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get schedules for brand: %w", err)
	}

	if len(schedules) == 0 {
		return nil, "", fmt.Errorf("no schedules found for brand")
	}

	// Get the first schedule's LLM IDs
	schedule := schedules[0]
	if len(schedule.LLMIDs) == 0 {
		return nil, "", fmt.Errorf("no LLM configured in schedule")
	}

	// Get the first LLM config
	llmID := schedule.LLMIDs[0]
	llmConfig, err := s.db.GetLLM(ctx, llmID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get LLM config: %w", err)
	}

	if llmConfig == nil || !llmConfig.Enabled {
		return nil, "", fmt.Errorf("LLM not found or disabled: %s", llmID)
	}

	// Get the provider from registry
	provider, ok := s.llmRegistry.Get(llmConfig.Provider)
	if !ok {
		return nil, "", fmt.Errorf("LLM provider not found: %s", llmConfig.Provider)
	}

	fmt.Printf("🔧 Using LLM from schedule: provider=%s, model=%s\n", llmConfig.Provider, llmConfig.Model)
	return provider, llmConfig.Model, nil
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

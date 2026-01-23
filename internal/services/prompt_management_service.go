package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/google/uuid"
)

// BrandPromptService provides prompt management operations similar to CompetitorService
type BrandPromptService struct {
	db          db.Database
	llmRegistry *llm.Registry
	scraper     *WebScraperService
}

// NewBrandPromptService creates a new brand prompt service
func NewBrandPromptService(database db.Database, llmRegistry *llm.Registry) *BrandPromptService {
	return &BrandPromptService{
		db:          database,
		llmRegistry: llmRegistry,
		scraper:     NewWebScraperService(),
	}
}

// GetPrompts retrieves both active and suggested prompts for a brand
func (s *BrandPromptService) GetPrompts(ctx context.Context, brandID string) (*models.GetPromptsResponse, error) {
	// Get brand prompts record first to get brand name and prompt IDs
	brandPromptsRecord, err := s.db.GetBrandPrompts(ctx, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get brand prompts: %w", err)
	}

	var brand string
	var activePrompts []models.PromptDetail

	if brandPromptsRecord != nil {
		brand = brandPromptsRecord.Brand

		// Get active prompts from active_prompt_ids in brand_prompts table
		if len(brandPromptsRecord.ActivePromptIDs) > 0 {
			// Fetch prompt details for active prompt IDs - optimized: batch fetch instead of N+1 queries
			prompts, err := s.db.GetPromptsByIDs(ctx, brandPromptsRecord.ActivePromptIDs)
			if err != nil {
				// If batch fetch fails, fall back to individual fetches (backward compatibility)
				for _, promptID := range brandPromptsRecord.ActivePromptIDs {
					prompt, err := s.db.GetPrompt(ctx, promptID)
					if err != nil || prompt == nil {
						continue // Skip invalid prompts
					}
					activePrompts = append(activePrompts, models.PromptDetail{
						ID:                      prompt.ID,
						Template:                prompt.Template,
						PromptType:              prompt.PromptType,
						Category:                prompt.Category,
						TargetingSearchKeywords: prompt.TargetingSearchKeywords,
						SupportingFanoutQueries: prompt.SupportingFanoutQueries,
					})
				}
			} else {
				// Use a map to track unique templates (normalized) and keep the first occurrence
				templateMap := make(map[string]models.PromptDetail)
				for _, prompt := range prompts {
					normalizedTemplate := normalizeTemplate(prompt.Template)

					// Only add if we haven't seen this template before
					if _, exists := templateMap[normalizedTemplate]; !exists {
						detail := models.PromptDetail{
							ID:                      prompt.ID,
							Template:                prompt.Template,
							PromptType:              prompt.PromptType,
							Category:                prompt.Category,
							TargetingSearchKeywords: prompt.TargetingSearchKeywords,
							SupportingFanoutQueries: prompt.SupportingFanoutQueries,
						}
						templateMap[normalizedTemplate] = detail
						activePrompts = append(activePrompts, detail)
					}
				}
			}
		}
	} else {
		// Fallback: If no brand_prompts record exists, try to get prompts by brand name
		// This is for backward compatibility with existing data
		// Note: This requires brand name, which we don't have without brandID lookup
		// So we'll return empty active prompts if no record exists
	}

	var suggestedPrompts []models.PromptDetail
	if brandPromptsRecord != nil && len(brandPromptsRecord.SuggestedPromptIDs) > 0 {
		// Fetch prompt details for suggested prompt IDs - optimized: batch fetch instead of N+1 queries
		prompts, err := s.db.GetPromptsByIDs(ctx, brandPromptsRecord.SuggestedPromptIDs)
		if err != nil {
			// If batch fetch fails, fall back to individual fetches (backward compatibility)
			for _, promptID := range brandPromptsRecord.SuggestedPromptIDs {
				prompt, err := s.db.GetPrompt(ctx, promptID)
				if err != nil || prompt == nil {
					continue // Skip invalid prompts
				}
				suggestedPrompts = append(suggestedPrompts, models.PromptDetail{
					ID:         prompt.ID,
					Template:   prompt.Template,
					PromptType: prompt.PromptType,
					Category:   prompt.Category,
				})
			}
		} else {
			// Convert batch-fetched prompts to PromptDetail
			for _, prompt := range prompts {
				suggestedPrompts = append(suggestedPrompts, models.PromptDetail{
					ID:                      prompt.ID,
					Template:                prompt.Template,
					PromptType:              prompt.PromptType,
					Category:                prompt.Category,
					TargetingSearchKeywords: prompt.TargetingSearchKeywords,
					SupportingFanoutQueries: prompt.SupportingFanoutQueries,
				})
			}
		}
	}

	return &models.GetPromptsResponse{
		Brand:            brand,
		ActivePrompts:    activePrompts,
		SuggestedPrompts: suggestedPrompts,
		Source:           "database",
		UpdatedAt:        time.Now(),
	}, nil
}

// SuggestPrompts suggests prompts for a brand using LLM
// Tries all enabled LLMs until one works and caches the results
func (s *BrandPromptService) SuggestPrompts(
	ctx context.Context,
	brand string,
	brandID string,
	orgID string,
	website string,
	category string,
	domain string,
	description string,
	count int,
	forceRefresh bool,
) (*models.SuggestPromptsResponse, error) {
	// Check if we already have cached suggestions (unless force refresh)
	var existing *models.BrandPrompts
	var err error

	if !forceRefresh {
		existing, err = s.db.GetBrandPrompts(ctx, brandID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing prompts: %w", err)
		}

		// If we have suggested list cached, return it (deterministic behavior)
		if existing != nil && len(existing.SuggestedPromptIDs) > 0 {
			// Fetch prompt details - optimized: batch fetch instead of N+1 queries
			var prompts []models.PromptDetail
			missingStructuredFields := false
			promptRecords, err := s.db.GetPromptsByIDs(ctx, existing.SuggestedPromptIDs)
			if err != nil {
				// If batch fetch fails, fall back to individual fetches (backward compatibility)
				for _, promptID := range existing.SuggestedPromptIDs {
					prompt, err := s.db.GetPrompt(ctx, promptID)
					if err != nil || prompt == nil {
						continue
					}
					// Auto-refresh cache if legacy prompts are missing new structured fields
					// (this happens for prompts generated before we started storing them).
					if len(prompt.TargetingSearchKeywords) == 0 || len(prompt.SupportingFanoutQueries) == 0 {
						missingStructuredFields = true
					}
					prompts = append(prompts, models.PromptDetail{
						ID:                      prompt.ID,
						Template:                prompt.Template,
						PromptType:              prompt.PromptType,
						Category:                prompt.Category,
						TargetingSearchKeywords: prompt.TargetingSearchKeywords,
						SupportingFanoutQueries: prompt.SupportingFanoutQueries,
					})
				}
			} else {
				// Convert batch-fetched prompts to PromptDetail
				for _, prompt := range promptRecords {
					if len(prompt.TargetingSearchKeywords) == 0 || len(prompt.SupportingFanoutQueries) == 0 {
						missingStructuredFields = true
					}
					prompts = append(prompts, models.PromptDetail{
						ID:                      prompt.ID,
						Template:                prompt.Template,
						PromptType:              prompt.PromptType,
						Category:                prompt.Category,
						TargetingSearchKeywords: prompt.TargetingSearchKeywords,
						SupportingFanoutQueries: prompt.SupportingFanoutQueries,
					})
				}
			}
			// If cached prompts don't contain the structured fields, treat cache as stale and regenerate.
			if !missingStructuredFields {
				return &models.SuggestPromptsResponse{
					Brand:      brand,
					Prompts:    prompts,
					Source:     "cached",
					Message:    "Returning cached prompt suggestions",
					LLMDetails: nil, // LLM details not available for cached responses
				}, nil
			}

			fmt.Printf("⚠️ Cached suggested prompts missing structured fields; regenerating suggestions for brandID=%s\n", brandID)
		}
	} else {
		// Even if force refresh, we need to get existing record to preserve active prompts
		existing, err = s.db.GetBrandPrompts(ctx, brandID)
		if err != nil {
			fmt.Printf("Warning: failed to check existing prompts when caching: %v\n", err)
		}
	}

	// Use LLM to suggest prompts (first request or force refresh)
	// Tries all enabled LLMs until one works
	promptTemplates, llmDetails, err := s.suggestPromptsWithLLM(ctx, brand, website, category, domain, description, count)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest prompts: %w", err)
	}

	if len(promptTemplates) == 0 {
		return &models.SuggestPromptsResponse{
			Brand:      brand,
			Prompts:    []models.PromptDetail{},
			Source:     "llm",
			Message:    "No prompts could be generated. Please provide more details about your brand.",
			LLMDetails: llmDetails,
		}, nil
	}

	// Create prompt records and get their IDs
	var promptIDs []string
	var promptDetails []models.PromptDetail
	for _, result := range promptTemplates {
		// Map intentType to PromptType
		promptType := mapIntentTypeToPromptType(result.IntentType)

		// If intentType is empty or mapping failed, use default
		if promptType == models.PromptTypeCustom {
			promptType = models.PromptTypeCustom
		}

		prompt := &models.Prompt{
			ID:                      uuid.New().String(),
			BrandID:                 brandID,
			OrgID:                   orgID,
			Template:                result.Prompt,
			PromptType:              promptType,
			Category:                category,
			Domain:                  domain,
			Brand:                   brand,
			Generated:               true,
			Enabled:                 false, // Suggested prompts are not enabled by default
			TargetingSearchKeywords: result.TargetingSearchKeywords,
			SupportingFanoutQueries: result.SupportingFanoutQueries,
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		}

		if err := s.db.CreatePrompt(ctx, prompt); err != nil {
			fmt.Printf("Warning: failed to create prompt: %v\n", err)
			continue
		}

		promptIDs = append(promptIDs, prompt.ID)
		promptDetails = append(promptDetails, models.PromptDetail{
			ID:                      prompt.ID,
			Template:                prompt.Template,
			PromptType:              prompt.PromptType,
			Category:                prompt.Category,
			TargetingSearchKeywords: prompt.TargetingSearchKeywords,
			SupportingFanoutQueries: prompt.SupportingFanoutQueries,
		})
	}

	// Cache the suggestions for future use
	// Preserve existing active prompts if they exist
	var activePromptIDs []string
	var id string
	var createdAt time.Time
	var source string

	if existing != nil {
		// Preserve existing active prompts and metadata
		activePromptIDs = existing.ActivePromptIDs
		id = existing.ID
		createdAt = existing.CreatedAt
		if len(activePromptIDs) > 0 {
			source = existing.Source
		} else {
			source = "suggested"
		}
	} else {
		// New record
		id = uuid.New().String()
		createdAt = time.Now()
		source = "suggested"
	}

	brandPrompts := &models.BrandPrompts{
		ID:                 id,
		Brand:              brand,
		BrandID:            brandID,
		OrgID:              orgID,
		ActivePromptIDs:    activePromptIDs, // Preserve existing active prompts
		SuggestedPromptIDs: promptIDs,       // LLM-suggested list - cache for deterministic behavior
		Source:             source,
		CreatedAt:          createdAt,
		UpdatedAt:          time.Now(),
	}

	if err := s.db.SaveBrandPrompts(ctx, brandPrompts); err != nil {
		// Log error but don't fail - we can still return suggestions
		fmt.Printf("Warning: failed to cache prompt suggestions: %v\n", err)
	}

	return &models.SuggestPromptsResponse{
		Brand:      brand,
		Prompts:    promptDetails,
		Source:     "llm",
		Message:    fmt.Sprintf("Found %d prompts for %s", len(promptDetails), brand),
		LLMDetails: llmDetails,
	}, nil
}

// suggestPromptsWithLLM uses LLM to suggest prompts based on brand info
// Tries all enabled LLMs in the system until one works successfully
// Returns results from the first working LLM along with LLM details
func (s *BrandPromptService) suggestPromptsWithLLM(
	ctx context.Context,
	brand string,
	website string,
	category string,
	domain string,
	description string,
	count int,
) ([]PromptGenerationResult, *models.LLMDetails, error) {
	// Get all enabled LLMs from the database
	enabled := true
	llms, err := s.db.ListLLMs(ctx, &enabled)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get enabled LLMs: %w", err)
	}

	if len(llms) == 0 {
		return nil, nil, fmt.Errorf("no enabled LLMs found in the system")
	}

	// Try each LLM until one works
	var lastErr error
	for _, llmConfig := range llms {
		// Get the provider from registry using the provider name
		provider, ok := s.llmRegistry.Get(llmConfig.Provider)
		if !ok {
			fmt.Printf("Warning: LLM provider '%s' not found in registry for LLM %s, trying next...\n", llmConfig.Provider, llmConfig.Name)
			continue
		}

		// Try to get suggestions with this LLM
		results, err := s.tryLLMForPromptSuggestions(ctx, provider, llmConfig, brand, website, category, domain, description, count)
		if err == nil && len(results) > 0 {
			// Success! Return the results with LLM details
			llmDetails := &models.LLMDetails{
				ID:       llmConfig.ID,
				Name:     llmConfig.Name,
				Provider: llmConfig.Provider,
				Model:    llmConfig.Model,
			}
			return results, llmDetails, nil
		}

		// This LLM failed, log and try next
		if err != nil {
			fmt.Printf("Warning: LLM %s (%s) failed: %v, trying next...\n", llmConfig.Name, llmConfig.Provider, err)
			lastErr = err
		} else {
			fmt.Printf("Warning: LLM %s (%s) returned empty results, trying next...\n", llmConfig.Name, llmConfig.Provider)
		}
	}

	// All LLMs failed
	return nil, nil, fmt.Errorf("all enabled LLMs failed. Last error: %w", lastErr)
}

// tryLLMForPromptSuggestions attempts to get prompt suggestions from a specific LLM
// Uses the shared generatePromptsWithLLM function for consistency
func (s *BrandPromptService) tryLLMForPromptSuggestions(
	ctx context.Context,
	provider llm.Provider,
	llmConfig *models.LLMConfig,
	brand string,
	website string,
	category string,
	domain string,
	description string,
	count int,
) ([]PromptGenerationResult, error) {
	// Scrape website if provided to enrich context
	var websiteContent *WebsiteContent
	if website != "" {
		content, err := s.scraper.ScrapeWebsite(ctx, website)
		if err != nil {
			fmt.Printf("Warning: failed to scrape website %s: %v\n", website, err)
		} else {
			websiteContent = content
			if description == "" && content.Description != "" {
				description = content.Description
			}
		}
	}

	// Get model from config
	model := ""
	if llmConfig != nil {
		model = llmConfig.Model
	}

	// Use shared function for prompt generation
	results, err := generatePromptsWithLLM(ctx, provider, model, brand, websiteContent, category, description, count, "[BrandPromptService]")
	if err != nil {
		return nil, err
	}

	// Check for errors in response (handled in generatePromptsWithLLM)
	if len(results) == 0 {
		return nil, fmt.Errorf("LLM returned empty prompt list")
	}

	fmt.Printf("✅ [BrandPromptService] Successfully parsed %d prompts\n", len(results))
	return results, nil
}

// SavePrompts saves prompts (moves from suggested to active)
// Accepts both prompt IDs (from suggested) and custom prompts
func (s *BrandPromptService) SavePrompts(
	ctx context.Context,
	brand string,
	brandID string,
	orgID string,
	promptIDs []string,
	customPrompts []models.CustomPrompt,
	source string,
) (*models.SavePromptsResponse, error) {
	// Get existing brand prompts
	existing, err := s.db.GetBrandPrompts(ctx, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing prompts: %w", err)
	}

	var existingActivePromptIDs []string
	var existingSuggestedPromptIDs []string
	var id string
	var createdAt time.Time

	if existing != nil {
		existingActivePromptIDs = existing.ActivePromptIDs
		existingSuggestedPromptIDs = existing.SuggestedPromptIDs
		id = existing.ID
		createdAt = existing.CreatedAt
	} else {
		// No BrandPrompts record exists - check for any existing suggested prompts in database
		// (prompts with enabled=false or brand set but not in active list)
		allPrompts, err := s.db.ListPrompts(ctx, nil)
		if err == nil {
			suggestedMap := make(map[string]bool)
			for _, prompt := range allPrompts {
				if prompt.Brand != "" && strings.EqualFold(prompt.Brand, brand) && !prompt.Enabled {
					// This is a suggested prompt (disabled, has brand)
					if !suggestedMap[prompt.ID] {
						existingSuggestedPromptIDs = append(existingSuggestedPromptIDs, prompt.ID)
						suggestedMap[prompt.ID] = true
					}
				}
			}
		}
		id = uuid.New().String()
		createdAt = time.Now()
	}

	// Build a map of existing active prompt templates for deduplication
	// Key: normalized template, Value: prompt ID
	activeTemplateMap := make(map[string]string)
	for _, activeID := range existingActivePromptIDs {
		activePrompt, err := s.db.GetPrompt(ctx, activeID)
		if err == nil && activePrompt != nil {
			normalizedTemplate := normalizeTemplate(activePrompt.Template)
			activeTemplateMap[normalizedTemplate] = activeID
		}
	}

	// Process prompt IDs (from suggested list)
	var savedPromptIDs []string
	createdCount := 0
	existingCount := 0

	// Add prompt IDs from suggested list (validate and move to active)
	for _, promptID := range promptIDs {
		prompt, err := s.db.GetPrompt(ctx, promptID)
		if err != nil || prompt == nil {
			continue // Skip invalid prompts
		}

		// Check for duplicate template in active prompts
		normalizedTemplate := normalizeTemplate(prompt.Template)
		if existingID, exists := activeTemplateMap[normalizedTemplate]; exists {
			// Template already exists in active prompts, skip adding duplicate
			fmt.Printf("Skipping duplicate prompt template (already exists as ID: %s): %s\n", existingID, prompt.Template)
			// Still remove from suggested list
			var newSuggested []string
			for _, suggestedID := range existingSuggestedPromptIDs {
				if suggestedID != promptID {
					newSuggested = append(newSuggested, suggestedID)
				}
			}
			existingSuggestedPromptIDs = newSuggested
			continue
		}

		// Check if already in active list by ID
		found := false
		for _, existingID := range existingActivePromptIDs {
			if existingID == promptID {
				found = true
				break
			}
		}
		if found {
			continue // Already in active list
		}

		// Mark prompt as active (enabled and set brand)
		prompt.Enabled = true
		prompt.Brand = brand
		prompt.UpdatedAt = time.Now()
		if err := s.db.UpdatePrompt(ctx, prompt); err != nil {
			continue
		}

		// Add to active list and template map
		existingActivePromptIDs = append(existingActivePromptIDs, promptID)
		activeTemplateMap[normalizedTemplate] = promptID

		// Remove from suggested list
		var newSuggested []string
		for _, suggestedID := range existingSuggestedPromptIDs {
			if suggestedID != promptID {
				newSuggested = append(newSuggested, suggestedID)
			}
		}
		existingSuggestedPromptIDs = newSuggested

		savedPromptIDs = append(savedPromptIDs, promptID)
		existingCount++
	}

	// Create new custom prompts
	for _, customPrompt := range customPrompts {
		// Check for duplicate template in active prompts
		normalizedTemplate := normalizeTemplate(customPrompt.Template)
		if existingID, exists := activeTemplateMap[normalizedTemplate]; exists {
			// Template already exists in active prompts, skip creating duplicate
			fmt.Printf("Skipping duplicate custom prompt template (already exists as ID: %s): %s\n", existingID, customPrompt.Template)
			// Add existing prompt ID to saved list if not already there
			found := false
			for _, savedID := range savedPromptIDs {
				if savedID == existingID {
					found = true
					break
				}
			}
			if !found {
				savedPromptIDs = append(savedPromptIDs, existingID)
				existingCount++
			}
			continue
		}

		promptType := models.PromptType(customPrompt.PromptType)
		if promptType == "" {
			promptType = models.PromptTypeCustom
		}

		prompt := &models.Prompt{
			ID:         uuid.New().String(),
			BrandID:    brandID,
			OrgID:      orgID,
			Template:   customPrompt.Template,
			PromptType: promptType,
			Category:   customPrompt.Category,
			Tags:       customPrompt.Tags,
			Brand:      brand,
			Generated:  false,
			Enabled:    true, // Custom prompts are active by default
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := s.db.CreatePrompt(ctx, prompt); err != nil {
			return nil, fmt.Errorf("failed to create custom prompt: %w", err)
		}

		// Add to active list and template map
		existingActivePromptIDs = append(existingActivePromptIDs, prompt.ID)
		activeTemplateMap[normalizedTemplate] = prompt.ID
		savedPromptIDs = append(savedPromptIDs, prompt.ID)
		createdCount++
	}

	// Update brand prompts record
	brandPrompts := &models.BrandPrompts{
		ID:                 id,
		Brand:              brand,
		BrandID:            brandID,
		OrgID:              orgID,
		ActivePromptIDs:    existingActivePromptIDs,
		SuggestedPromptIDs: existingSuggestedPromptIDs,
		Source:             source,
		CreatedAt:          createdAt,
		UpdatedAt:          time.Now(),
	}

	if err := s.db.SaveBrandPrompts(ctx, brandPrompts); err != nil {
		return nil, fmt.Errorf("failed to save brand prompts: %w", err)
	}

	return &models.SavePromptsResponse{
		Brand:          brand,
		SavedPromptIDs: savedPromptIDs,
		CreatedCount:   createdCount,
		ExistingCount:  existingCount,
	}, nil
}

// DeletePrompt deletes a prompt from active list and moves it back to suggested list
func (s *BrandPromptService) DeletePrompt(ctx context.Context, brandID string, promptID string) error {
	// Get existing brand prompts
	existing, err := s.db.GetBrandPrompts(ctx, brandID)
	if err != nil {
		return fmt.Errorf("failed to get existing prompts: %w", err)
	}

	if existing == nil {
		return fmt.Errorf("no prompts found for brandID: %s", brandID)
	}

	// Find and remove from active list
	var newActive []string
	found := false
	for _, id := range existing.ActivePromptIDs {
		if id == promptID {
			found = true
			// Don't add to newActive - removing it
		} else {
			newActive = append(newActive, id)
		}
	}

	if !found {
		return fmt.Errorf("prompt %s not found in active list for brandID %s", promptID, brandID)
	}

	// Add back to suggested list (if not already present)
	var newSuggested []string
	alreadyInSuggested := false
	for _, id := range existing.SuggestedPromptIDs {
		if id == promptID {
			alreadyInSuggested = true
		}
		newSuggested = append(newSuggested, id)
	}

	if !alreadyInSuggested {
		newSuggested = append(newSuggested, promptID)
	}

	// Disable the prompt (but don't delete it)
	prompt, err := s.db.GetPrompt(ctx, promptID)
	if err == nil && prompt != nil {
		prompt.Enabled = false
		prompt.UpdatedAt = time.Now()
		_ = s.db.UpdatePrompt(ctx, prompt)
	}

	// Update brand prompts record
	brandPrompts := &models.BrandPrompts{
		ID:                 existing.ID,
		Brand:              existing.Brand,
		BrandID:            existing.BrandID,
		OrgID:              existing.OrgID,
		ActivePromptIDs:    newActive,
		SuggestedPromptIDs: newSuggested,
		Source:             existing.Source,
		CreatedAt:          existing.CreatedAt,
		UpdatedAt:          time.Now(),
	}

	return s.db.SaveBrandPrompts(ctx, brandPrompts)
}

// DeleteAllPrompts deletes all active prompts for a brand and moves them to suggested list
func (s *BrandPromptService) DeleteAllPrompts(ctx context.Context, brandID string) error {
	// Get existing brand prompts
	existing, err := s.db.GetBrandPrompts(ctx, brandID)
	if err != nil {
		return fmt.Errorf("failed to get existing prompts: %w", err)
	}

	// If no BrandPrompts record exists, there's nothing to delete - return nil (not an error)
	if existing == nil {
		return nil
	}

	// Move all active prompts to suggested list (deduplicate)
	var newSuggested []string
	suggestedMap := make(map[string]bool)

	// Add existing suggested prompts
	for _, id := range existing.SuggestedPromptIDs {
		if !suggestedMap[id] {
			newSuggested = append(newSuggested, id)
			suggestedMap[id] = true
		}
	}

	// Add all active prompts to suggested
	for _, id := range existing.ActivePromptIDs {
		if !suggestedMap[id] {
			newSuggested = append(newSuggested, id)
			suggestedMap[id] = true
		}

		// Disable the prompt
		prompt, err := s.db.GetPrompt(ctx, id)
		if err == nil && prompt != nil {
			prompt.Enabled = false
			prompt.UpdatedAt = time.Now()
			_ = s.db.UpdatePrompt(ctx, prompt)
		}
	}

	// Update brand prompts record
	brandPrompts := &models.BrandPrompts{
		ID:                 existing.ID,
		Brand:              existing.Brand,
		BrandID:            existing.BrandID,
		OrgID:              existing.OrgID,
		ActivePromptIDs:    []string{}, // Clear active list
		SuggestedPromptIDs: newSuggested,
		Source:             existing.Source,
		CreatedAt:          existing.CreatedAt,
		UpdatedAt:          time.Now(),
	}

	return s.db.SaveBrandPrompts(ctx, brandPrompts)
}

// normalizeTemplate normalizes a prompt template for comparison
// Removes extra whitespace, converts to lowercase, and trims
func normalizeTemplate(template string) string {
	// Trim whitespace and convert to lowercase for comparison
	normalized := strings.TrimSpace(template)
	normalized = strings.ToLower(normalized)
	// Replace multiple spaces with single space
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

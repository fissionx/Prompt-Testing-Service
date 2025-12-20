package services

import (
	"context"
	"encoding/json"
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
func (s *BrandPromptService) GetPrompts(ctx context.Context, brand string) (*models.GetPromptsResponse, error) {
	// Get active prompts (enabled prompts with brand set)
	enabled := true
	allPrompts, err := s.db.ListPrompts(ctx, &enabled)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	var activePrompts []models.PromptDetail
	for _, prompt := range allPrompts {
		if prompt.Brand != "" && strings.EqualFold(prompt.Brand, brand) && prompt.Enabled {
			activePrompts = append(activePrompts, models.PromptDetail{
				ID:         prompt.ID,
				Template:   prompt.Template,
				PromptType: prompt.PromptType,
				Category:   prompt.Category,
			})
		}
	}

	// Get suggested prompts from BrandPrompts
	brandPrompts, err := s.db.GetBrandPrompts(ctx, brand)
	if err != nil {
		return nil, fmt.Errorf("failed to get brand prompts: %w", err)
	}

	var suggestedPrompts []models.PromptDetail
	if brandPrompts != nil && len(brandPrompts.SuggestedPromptIDs) > 0 {
		// Fetch prompt details for suggested prompt IDs
		for _, promptID := range brandPrompts.SuggestedPromptIDs {
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
	}

	return &models.GetPromptsResponse{
		Brand:          brand,
		ActivePrompts:  activePrompts,
		SuggestedPrompts: suggestedPrompts,
		Source:         "database",
		UpdatedAt:      time.Now(),
	}, nil
}

// SuggestPrompts suggests prompts for a brand using LLM
// Tries all enabled LLMs until one works and caches the results
func (s *BrandPromptService) SuggestPrompts(
	ctx context.Context,
	brand string,
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
		existing, err = s.db.GetBrandPrompts(ctx, brand)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing prompts: %w", err)
		}

		// If we have suggested list cached, return it (deterministic behavior)
		if existing != nil && len(existing.SuggestedPromptIDs) > 0 {
			// Fetch prompt details
			var prompts []models.PromptDetail
			for _, promptID := range existing.SuggestedPromptIDs {
				prompt, err := s.db.GetPrompt(ctx, promptID)
				if err != nil || prompt == nil {
					continue
				}
				prompts = append(prompts, models.PromptDetail{
					ID:         prompt.ID,
					Template:   prompt.Template,
					PromptType: prompt.PromptType,
					Category:   prompt.Category,
				})
			}
			return &models.SuggestPromptsResponse{
				Brand:       brand,
				Prompts:     prompts,
				Source:      "cached",
				Message:     "Returning cached prompt suggestions",
				LLMDetails:   nil, // LLM details not available for cached responses
			}, nil
		}
	} else {
		// Even if force refresh, we need to get existing record to preserve active prompts
		existing, err = s.db.GetBrandPrompts(ctx, brand)
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
			Brand:       brand,
			Prompts:     []models.PromptDetail{},
			Source:      "llm",
			Message:     "No prompts could be generated. Please provide more details about your brand.",
			LLMDetails:   llmDetails,
		}, nil
	}

	// Create prompt records and get their IDs
	var promptIDs []string
	var promptDetails []models.PromptDetail
	for _, template := range promptTemplates {
		prompt := &models.Prompt{
			ID:         uuid.New().String(),
			Template:   template,
			PromptType: models.PromptTypeCustom, // Default type
			Category:   category,
			Domain:     domain,
			Brand:      brand,
			Generated:  true,
			Enabled:    false, // Suggested prompts are not enabled by default
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := s.db.CreatePrompt(ctx, prompt); err != nil {
			fmt.Printf("Warning: failed to create prompt: %v\n", err)
			continue
		}

		promptIDs = append(promptIDs, prompt.ID)
		promptDetails = append(promptDetails, models.PromptDetail{
			ID:         prompt.ID,
			Template:   prompt.Template,
			PromptType: prompt.PromptType,
			Category:   prompt.Category,
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
		ID:                  id,
		Brand:               brand,
		ActivePromptIDs:     activePromptIDs, // Preserve existing active prompts
		SuggestedPromptIDs: promptIDs,        // LLM-suggested list - cache for deterministic behavior
		Source:              source,
		CreatedAt:           createdAt,
		UpdatedAt:           time.Now(),
	}

	if err := s.db.SaveBrandPrompts(ctx, brandPrompts); err != nil {
		// Log error but don't fail - we can still return suggestions
		fmt.Printf("Warning: failed to cache prompt suggestions: %v\n", err)
	}

	return &models.SuggestPromptsResponse{
		Brand:       brand,
		Prompts:     promptDetails,
		Source:      "llm",
		Message:     fmt.Sprintf("Found %d prompts for %s", len(promptDetails), brand),
		LLMDetails:   llmDetails,
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
) ([]string, *models.LLMDetails, error) {
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
		prompts, err := s.tryLLMForPromptSuggestions(ctx, provider, llmConfig, brand, website, category, domain, description, count)
		if err == nil && len(prompts) > 0 {
			// Success! Return the results with LLM details
			llmDetails := &models.LLMDetails{
				ID:       llmConfig.ID,
				Name:     llmConfig.Name,
				Provider: llmConfig.Provider,
				Model:    llmConfig.Model,
			}
			return prompts, llmDetails, nil
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
) ([]string, error) {
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

	// Build rich context for prompt suggestion
	var contextParts []string
	contextParts = append(contextParts, fmt.Sprintf("Brand/Company: %s", brand))

	if website != "" {
		contextParts = append(contextParts, fmt.Sprintf("Website: %s", website))
	}

	if description != "" {
		contextParts = append(contextParts, fmt.Sprintf("Description: %s", description))
	}

	if category != "" {
		contextParts = append(contextParts, fmt.Sprintf("Category/Industry: %s", category))
	}

	if domain != "" {
		contextParts = append(contextParts, fmt.Sprintf("Domain: %s", domain))
	}

	// Add scraped website content for richer context
	if websiteContent != nil {
		if websiteContent.Title != "" {
			contextParts = append(contextParts, fmt.Sprintf("Website Title: %s", websiteContent.Title))
		}
		if websiteContent.Description != "" && description == "" {
			contextParts = append(contextParts, fmt.Sprintf("Website Meta: %s", websiteContent.Description))
		}
	}

	brandContext := strings.Join(contextParts, "\n")

	// Build the LLM prompt for prompt suggestion
	prompt := fmt.Sprintf(`You are a prompt generation expert. Based on the following brand information, generate %d SEO-optimized prompts/questions that would help this brand appear in AI search results.

%s

---

TASK: Generate %d diverse prompts/questions that:
1. Are relevant to this brand/company
2. Would help the brand appear in AI search results (like ChatGPT, Gemini, etc.)
3. Cover different question types (what, how, comparison, top/best lists, brand-specific)
4. Are natural, conversational, and search-friendly

RULES:
1. Each prompt should be a complete question or search query
2. Make prompts specific and actionable
3. Include brand name naturally where appropriate
4. Vary the question types (what, how, which, best, top, etc.)
5. Focus on topics relevant to the brand's domain/category

RESPOND WITH ONLY A JSON ARRAY of prompt strings. No explanations, no markdown, just the JSON array.

Example response format:
["What is Brand X?", "How does Brand X work?", "Best alternatives to Brand X", "Brand X vs Competitor Y"]

RESPOND NOW:`, count, brandContext, count)

	// Call LLM
	llmConfigStruct := llm.Config{
		Temperature: 0.7, // Higher temperature for more creative prompts
		MaxTokens:   1000,
	}

	// Use model from LLM config if available
	if llmConfig.Model != "" {
		llmConfigStruct.Model = llmConfig.Model
	}

	response, err := provider.Generate(ctx, prompt, llmConfigStruct)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Check for errors in response
	if response.Error != "" {
		return nil, fmt.Errorf("LLM returned error: %s", response.Error)
	}

	// Parse the JSON response
	prompts, err := parsePromptResponse(response.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(prompts) == 0 {
		return nil, fmt.Errorf("LLM returned empty prompt list")
	}

	return prompts, nil
}

// parsePromptResponse parses the LLM response into a list of prompts
func parsePromptResponse(response string) ([]string, error) {
	// Clean up the response
	response = strings.TrimSpace(response)
	
	// Remove markdown code blocks if present
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to parse as JSON array
	var prompts []string
	if err := json.Unmarshal([]byte(response), &prompts); err == nil {
		// Successfully parsed as JSON
		return prompts, nil
	}

	// If JSON parsing fails, try to parse line by line
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove list markers (1., 2., -, *, etc.)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		if len(line) > 2 && (line[1] == '.' || line[1] == ')') && (line[0] >= '0' && line[0] <= '9') {
			line = strings.TrimSpace(line[2:])
		}

		// Remove quotes if present
		line = strings.Trim(line, `"'`)

		if line != "" {
			prompts = append(prompts, line)
		}
	}

	if len(prompts) == 0 {
		return nil, fmt.Errorf("no valid prompts found in response")
	}

	return prompts, nil
}

// SavePrompts saves prompts (moves from suggested to active)
// Accepts both prompt IDs (from suggested) and custom prompts
func (s *BrandPromptService) SavePrompts(
	ctx context.Context,
	brand string,
	promptIDs []string,
	customPrompts []models.CustomPrompt,
	source string,
) (*models.SavePromptsResponse, error) {
	// Get existing brand prompts
	existing, err := s.db.GetBrandPrompts(ctx, brand)
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

		// Mark prompt as active (enabled and set brand)
		prompt.Enabled = true
		prompt.Brand = brand
		prompt.UpdatedAt = time.Now()
		if err := s.db.UpdatePrompt(ctx, prompt); err != nil {
			continue
		}

		// Add to active list (deduplicate)
		found := false
		for _, existingID := range existingActivePromptIDs {
			if existingID == promptID {
				found = true
				break
			}
		}
		if !found {
			existingActivePromptIDs = append(existingActivePromptIDs, promptID)
		}

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
		promptType := models.PromptType(customPrompt.PromptType)
		if promptType == "" {
			promptType = models.PromptTypeCustom
		}

		prompt := &models.Prompt{
			ID:         uuid.New().String(),
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

		// Add to active list
		existingActivePromptIDs = append(existingActivePromptIDs, prompt.ID)
		savedPromptIDs = append(savedPromptIDs, prompt.ID)
		createdCount++
	}

	// Update brand prompts record
	brandPrompts := &models.BrandPrompts{
		ID:                  id,
		Brand:               brand,
		ActivePromptIDs:     existingActivePromptIDs,
		SuggestedPromptIDs: existingSuggestedPromptIDs,
		Source:              source,
		CreatedAt:           createdAt,
		UpdatedAt:           time.Now(),
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
func (s *BrandPromptService) DeletePrompt(ctx context.Context, brand string, promptID string) error {
	// Get existing brand prompts
	existing, err := s.db.GetBrandPrompts(ctx, brand)
	if err != nil {
		return fmt.Errorf("failed to get existing prompts: %w", err)
	}

	if existing == nil {
		return fmt.Errorf("no prompts found for brand: %s", brand)
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
		return fmt.Errorf("prompt %s not found in active list for brand %s", promptID, brand)
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
		ID:                  existing.ID,
		Brand:               brand,
		ActivePromptIDs:     newActive,
		SuggestedPromptIDs: newSuggested,
		Source:              existing.Source,
		CreatedAt:           existing.CreatedAt,
		UpdatedAt:           time.Now(),
	}

	return s.db.SaveBrandPrompts(ctx, brandPrompts)
}

// DeleteAllPrompts deletes all active prompts for a brand and moves them to suggested list
func (s *BrandPromptService) DeleteAllPrompts(ctx context.Context, brand string) error {
	// Get existing brand prompts
	existing, err := s.db.GetBrandPrompts(ctx, brand)
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
		ID:                  existing.ID,
		Brand:               brand,
		ActivePromptIDs:     []string{}, // Clear active list
		SuggestedPromptIDs: newSuggested,
		Source:              existing.Source,
		CreatedAt:           existing.CreatedAt,
		UpdatedAt:           time.Now(),
	}

	return s.db.SaveBrandPrompts(ctx, brandPrompts)
}


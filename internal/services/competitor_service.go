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

// CompetitorService provides competitor management operations
type CompetitorService struct {
	db          db.Database
	llmRegistry *llm.Registry
	scraper     *WebScraperService
}

// NewCompetitorService creates a new competitor service
func NewCompetitorService(database db.Database, llmRegistry *llm.Registry) *CompetitorService {
	return &CompetitorService{
		db:          database,
		llmRegistry: llmRegistry,
		scraper:     NewWebScraperService(),
	}
}

// SuggestCompetitors suggests competitors for a brand using LLM
// Returns cached suggestions if available (deterministic), otherwise uses LLM to generate and cache suggestions
// First request: Uses LLM and caches results
// Subsequent requests: Returns cached suggestions (unless forceRefresh=true)
// Automatically tries all enabled LLMs until one works
func (s *CompetitorService) SuggestCompetitors(
	ctx context.Context,
	brand string,
	brandID string,
	website string,
	description string,
	category string,
	forceRefresh bool,
) (*models.SuggestCompetitorsResponse, error) {
	// Check if we already have cached suggestions (unless force refresh)
	var existing *models.BrandCompetitors
	var err error

	if !forceRefresh {
		existing, err = s.db.GetBrandCompetitors(ctx, brandID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing competitors: %w", err)
		}

		// If we have suggested list cached, return it (deterministic behavior)
		// Note: LLM details not available for cached responses
		if existing != nil && len(existing.SuggestedList) > 0 {
			competitors := convertStringListToCompetitors(existing.SuggestedList)
			return &models.SuggestCompetitorsResponse{
				Brand:       brand,
				Competitors: competitors,
				Source:      "cached",
				Message:     "Returning cached competitor suggestions",
				LLMDetails:  nil, // LLM details not available for cached responses
			}, nil
		}
	} else {
		// Even if force refresh, we need to get existing record to preserve saved competitors
		existing, err = s.db.GetBrandCompetitors(ctx, brandID)
		if err != nil {
			// Log error but continue - we'll create a new record
			fmt.Printf("Warning: failed to check existing competitors when caching: %v\n", err)
		}
	}

	// Use LLM to suggest competitors (first request or force refresh)
	// Tries all enabled LLMs until one works
	competitorNames, llmDetails, err := s.suggestCompetitorsWithLLM(ctx, brand, website, description, category)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest competitors: %w", err)
	}

	if len(competitorNames) == 0 {
		return &models.SuggestCompetitorsResponse{
			Brand:       brand,
			Competitors: []models.Competitor{},
			Source:      "llm",
			Message:     "No competitors could be identified. Please provide more details about your brand.",
			LLMDetails:  llmDetails,
		}, nil
	}

	// Convert competitor names to Competitor objects with derived domains
	competitors := convertStringListToCompetitors(competitorNames)

	// Cache the suggestions for future use (store as strings for backward compatibility)
	// Preserve existing saved competitors if they exist
	var savedCompetitors []string
	var id string
	var createdAt time.Time
	var source string

	if existing != nil {
		// Preserve existing saved competitors and metadata
		savedCompetitors = existing.Competitors
		id = existing.ID
		createdAt = existing.CreatedAt
		// Keep existing source if competitors are saved, otherwise mark as suggested
		if len(savedCompetitors) > 0 {
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

	brandCompetitors := &models.BrandCompetitors{
		ID:            id,
		BrandID:       brandID,
		Brand:         brand,
		Competitors:   savedCompetitors, // Preserve existing saved competitors
		SuggestedList: competitorNames,  // LLM-suggested list (as strings) - cache for deterministic behavior
		Source:        source,
		CreatedAt:     createdAt,
		UpdatedAt:     time.Now(),
	}

	if err := s.db.SaveBrandCompetitors(ctx, brandCompetitors); err != nil {
		// Log error but don't fail - we can still return suggestions
		fmt.Printf("Warning: failed to cache competitor suggestions: %v\n", err)
	}

	return &models.SuggestCompetitorsResponse{
		Brand:       brand,
		Competitors: competitors,
		Source:      "llm",
		Message:     fmt.Sprintf("Found %d competitors for %s", len(competitors), brand),
		LLMDetails:  llmDetails,
	}, nil
}

// suggestCompetitorsWithLLM uses LLM to suggest competitors based on brand info
// Tries all enabled LLMs in the system until one works successfully
// Returns results from the first working LLM along with LLM details
func (s *CompetitorService) suggestCompetitorsWithLLM(
	ctx context.Context,
	brand string,
	website string,
	description string,
	category string,
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
		competitorNames, err := s.tryLLMForSuggestions(ctx, provider, llmConfig, brand, website)
		if err == nil && len(competitorNames) > 0 {
			// Success! Return the results with LLM details
			llmDetails := &models.LLMDetails{
				ID:       llmConfig.ID,
				Name:     llmConfig.Name,
				Provider: llmConfig.Provider,
				Model:    llmConfig.Model,
			}
			return competitorNames, llmDetails, nil
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

// tryLLMForSuggestions attempts to get competitor suggestions from a specific LLM
func (s *CompetitorService) tryLLMForSuggestions(
	ctx context.Context,
	provider llm.Provider,
	llmConfig *models.LLMConfig,
	brand string,
	website string,
) ([]string, error) {

	// Scrape website if provided to enrich context
	var websiteContent *WebsiteContent
	if website != "" {
		content, err := s.scraper.ScrapeWebsite(ctx, website)
		if err != nil {
			// Log but don't fail - continue with other info
			fmt.Printf("Warning: failed to scrape website %s: %v\n", website, err)
		} else {
			websiteContent = content
		}
	}

	// Build rich context for competitor suggestion
	var contextParts []string
	contextParts = append(contextParts, fmt.Sprintf("Brand/Company: %s", brand))

	if website != "" {
		contextParts = append(contextParts, fmt.Sprintf("Website: %s", website))
	}

	// Add scraped website content for richer context
	if websiteContent != nil {
		if websiteContent.Title != "" {
			contextParts = append(contextParts, fmt.Sprintf("Website Title: %s", websiteContent.Title))
		}

		if len(websiteContent.Keywords) > 0 {
			contextParts = append(contextParts, fmt.Sprintf("Keywords: %s", strings.Join(websiteContent.Keywords, ", ")))
		}
		if websiteContent.MainContent != "" {
			// Limit content length
			content := websiteContent.MainContent
			if len(content) > 800 {
				content = content[:800] + "..."
			}
			contextParts = append(contextParts, fmt.Sprintf("Website Content: %s", content))
		}
	}

	brandContext := strings.Join(contextParts, "\n")

	// Build the LLM prompt for competitor suggestion
	prompt := fmt.Sprintf(`You are a competitive intelligence analyst. Based on the following brand information, identify the main competitors in the market.

%s

---

TASK: Identify 5-15 direct competitors for this brand/company.

RULES:
1. Focus on DIRECT competitors that offer similar products/services
2. Include both well-known industry leaders and emerging competitors
3. Consider companies that target the same customer base
4. Include companies that appear in the same "best of" lists or comparison articles
5. Be specific with company/product names (not generic terms)
6. Incase any competitors or comparision is mentioned in the website content, use that for competitors list

RESPOND WITH ONLY A JSON ARRAY of competitor names and their domains. No explanations, no markdown, just the JSON array. The domain should be the domain of the competitor website.

Few sample examples of competitor names:
  1. brand name is walmart, then competitors are amazon, target, costco, etc.
  2. brand name is apple, then competitors are samsung, google, microsoft, etc.
  3. brand name is IIT then competitors are IIT Madras, IIT Bombay, IIT Kanpur, etc.
  4. brand name is Zoho then competitors are Salesforce, SAP, Oracle, Freshworks, etc.
  5. brand name is instagram then competitors are facebook, twitter, tiktok, etc.

Example response format:
[{"name": "Competitor 1", "domain": "www.competitor1.com"}, {"name": "Competitor 2", "domain": "www.competitor2.com"}, {"name": "Competitor 3", "domain": "www.competitor3.com"}]

RESPOND NOW:`, brandContext)

	// Call LLM
	llmConfigStruct := llm.Config{
		Temperature: 0.3, // Lower temperature for more focused results
		MaxTokens:   500,
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
	competitors, err := parseCompetitorResponse(response.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(competitors) == 0 {
		return nil, fmt.Errorf("LLM returned empty competitor list")
	}

	return competitors, nil
}

// parseCompetitorResponse parses the LLM response into a list of competitors
// Supports both new format (JSON array of objects with name/domain) and old format (JSON array of strings)
func parseCompetitorResponse(response string) ([]string, error) {
	// Clean up the response
	response = strings.TrimSpace(response)

	// Remove markdown code blocks if present
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// First, try to parse as JSON array of objects (new format: [{"name": "...", "domain": "..."}])
	var competitorObjects []struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal([]byte(response), &competitorObjects); err == nil && len(competitorObjects) > 0 {
		// Successfully parsed as array of objects - convert to "name|domain" format
		var result []string
		seen := make(map[string]bool)
		for _, comp := range competitorObjects {
			name := strings.TrimSpace(comp.Name)
			domain := strings.TrimSpace(comp.Domain)

			if name == "" {
				continue // Skip entries without a name
			}

			// Create storage format: "name|domain" or just "name" if domain is empty
			storageStr := name
			if domain != "" {
				storageStr = name + "|" + domain
			}

			// Deduplicate by name (case-insensitive)
			nameLower := strings.ToLower(name)
			if !seen[nameLower] {
				seen[nameLower] = true
				result = append(result, storageStr)
			}
		}

		if len(result) > 0 {
			return result, nil
		}
	}

	// Fall back to old format: try to parse as JSON array of strings
	var competitors []string
	if err := json.Unmarshal([]byte(response), &competitors); err == nil && len(competitors) > 0 {
		// Successfully parsed as array of strings - clean and deduplicate
		seen := make(map[string]bool)
		var result []string
		for _, comp := range competitors {
			comp = strings.TrimSpace(comp)
			if comp != "" && !seen[strings.ToLower(comp)] {
				seen[strings.ToLower(comp)] = true
				result = append(result, comp)
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	// If JSON parsing fails, try to extract competitors from text
	competitors = extractCompetitorsFromText(response)

	// Clean and deduplicate
	seen := make(map[string]bool)
	var result []string
	for _, comp := range competitors {
		comp = strings.TrimSpace(comp)
		if comp != "" && !seen[strings.ToLower(comp)] {
			seen[strings.ToLower(comp)] = true
			result = append(result, comp)
		}
	}

	return result, nil
}

// extractCompetitorsFromText extracts competitors from plain text response
// Tries to extract JSON objects first, then falls back to text parsing
func extractCompetitorsFromText(text string) []string {
	var competitors []string

	// First, try to find and extract JSON array of objects in the text
	// Look for patterns like [{"name": "...", "domain": "..."}]
	// Try to find the start and end of a JSON array
	startIdx := strings.Index(text, "[{")
	if startIdx == -1 {
		startIdx = strings.Index(text, "[\n{")
	}
	if startIdx == -1 {
		startIdx = strings.Index(text, "[\r\n{")
	}

	if startIdx != -1 {
		// Find the matching closing bracket
		// Count brackets to find the proper end
		bracketCount := 0
		inString := false
		escapeNext := false
		endIdx := -1

		for i := startIdx; i < len(text); i++ {
			char := text[i]

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
				if char == '[' {
					bracketCount++
				} else if char == ']' {
					bracketCount--
					if bracketCount == 0 {
						endIdx = i
						break
					}
				}
			}
		}

		if endIdx != -1 && endIdx > startIdx {
			jsonStr := text[startIdx : endIdx+1]
			var competitorObjects []struct {
				Name   string `json:"name"`
				Domain string `json:"domain"`
			}
			if err := json.Unmarshal([]byte(jsonStr), &competitorObjects); err == nil && len(competitorObjects) > 0 {
				// Successfully extracted JSON objects
				for _, comp := range competitorObjects {
					name := strings.TrimSpace(comp.Name)
					domain := strings.TrimSpace(comp.Domain)
					if name != "" {
						if domain != "" {
							competitors = append(competitors, name+"|"+domain)
						} else {
							competitors = append(competitors, name)
						}
					}
				}
				if len(competitors) > 0 {
					return competitors
				}
			}
		}
	}

	// Fall back to text-based extraction
	// Split by common delimiters
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove common prefixes like "1.", "- ", "* "
		line = strings.TrimLeft(line, "0123456789.-*) ")
		line = strings.TrimSpace(line)
		if line != "" && len(line) < 100 { // Reasonable company name length
			competitors = append(competitors, line)
		}
	}

	return competitors
}

// SaveCompetitors adds new competitors to the existing list for a brand (does not replace)
// Deduplicates competitors by name (case-insensitive) to prevent duplicates
func (s *CompetitorService) SaveCompetitors(
	ctx context.Context,
	brand string,
	brandID string,
	newCompetitors []models.Competitor,
	source string,
) (*models.SaveCompetitorsResponse, error) {
	if brand == "" {
		return nil, fmt.Errorf("brand is required")
	}

	if len(newCompetitors) == 0 {
		return nil, fmt.Errorf("at least one competitor is required")
	}

	// Normalize source
	if source == "" {
		source = "custom"
	}

	// Get existing data
	existing, err := s.db.GetBrandCompetitors(ctx, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing competitors: %w", err)
	}

	var suggestedList []string
	var existingCompetitors []models.Competitor
	var id string
	var createdAt time.Time
	var existingSource string

	if existing != nil {
		// Preserve suggested list, existing competitors, and metadata
		suggestedList = existing.SuggestedList
		existingCompetitors = convertStringListToCompetitors(existing.Competitors)
		id = existing.ID
		createdAt = existing.CreatedAt
		existingSource = existing.Source
		// Keep existing source if it's not empty, otherwise use new source
		if existingSource != "" && existingSource != "suggested" {
			source = existingSource
		}
	} else {
		id = uuid.New().String()
		createdAt = time.Now()
	}

	// Build a map of existing competitor names (case-insensitive) for deduplication
	existingNames := make(map[string]bool)
	for _, comp := range existingCompetitors {
		name := strings.ToLower(strings.TrimSpace(comp.Name))
		existingNames[name] = true
	}

	// Merge new competitors with existing ones, avoiding duplicates
	addedCount := 0
	for _, newComp := range newCompetitors {
		newName := strings.ToLower(strings.TrimSpace(newComp.Name))
		if !existingNames[newName] {
			existingCompetitors = append(existingCompetitors, newComp)
			existingNames[newName] = true
			addedCount++
		}
	}

	// Convert merged competitor list to storage format
	allCompetitorStrings := convertCompetitorsToStorageFormat(existingCompetitors)

	// Save the merged competitor list
	brandCompetitors := &models.BrandCompetitors{
		ID:            id,
		Brand:         brand,
		BrandID:       brandID,
		Competitors:   allCompetitorStrings,
		SuggestedList: suggestedList,
		Source:        source,
		CreatedAt:     createdAt,
		UpdatedAt:     time.Now(),
	}

	if err := s.db.SaveBrandCompetitors(ctx, brandCompetitors); err != nil {
		return nil, fmt.Errorf("failed to save competitors: %w", err)
	}

	// Build response message
	var message string
	if addedCount == 0 {
		message = fmt.Sprintf("No new competitors added. All %d competitor(s) already exist for %s", len(newCompetitors), brand)
	} else if addedCount < len(newCompetitors) {
		skipped := len(newCompetitors) - addedCount
		message = fmt.Sprintf("Added %d new competitor(s) to %s. %d competitor(s) were already in the list", addedCount, brand, skipped)
	} else {
		message = fmt.Sprintf("Successfully added %d competitor(s) to %s", addedCount, brand)
	}

	return &models.SaveCompetitorsResponse{
		Brand:       brand,
		Competitors: existingCompetitors, // Return the full merged list
		Source:      source,
		SavedAt:     brandCompetitors.UpdatedAt,
		Message:     message,
	}, nil
}

// GetCompetitors retrieves saved competitors for a brand
func (s *CompetitorService) GetCompetitors(
	ctx context.Context,
	brandID string,
) (*models.GetCompetitorsResponse, error) {
	if brandID == "" {
		return nil, fmt.Errorf("brand is required")
	}

	competitors, err := s.db.GetBrandCompetitors(ctx, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get competitors: %w", err)
	}

	if competitors == nil {
		return &models.GetCompetitorsResponse{
			BrandID:     brandID,
			Competitors: []models.Competitor{},
			Source:      "none",
			UpdatedAt:   time.Now(),
		}, nil
	}

	// Convert string lists to Competitor objects
	competitorList := convertStringListToCompetitors(competitors.Competitors)
	suggestedList := convertStringListToCompetitors(competitors.SuggestedList)

	return &models.GetCompetitorsResponse{
		BrandID:       brandID,
		Competitors:   competitorList,
		SuggestedList: suggestedList,
		Source:        competitors.Source,
		UpdatedAt:     competitors.UpdatedAt,
	}, nil
}

// DeleteCompetitors moves all competitors from the competitors list to suggestedList
// This preserves the data and makes the suggestedList reliable
// The existing suggestedList is preserved and competitors are added to it (no duplicates)
func (s *CompetitorService) DeleteCompetitors(
	ctx context.Context,
	brandID string,
) error {
	if brandID == "" {
		return fmt.Errorf("brandID is required")
	}

	// Get existing data
	existing, err := s.db.GetBrandCompetitors(ctx, brandID)
	if err != nil {
		return fmt.Errorf("failed to get existing competitors: %w", err)
	}

	if existing == nil {
		// No competitors found, nothing to do
		return nil
	}

	// Convert to Competitor objects
	existingCompetitors := convertStringListToCompetitors(existing.Competitors)
	existingSuggestedList := convertStringListToCompetitors(existing.SuggestedList)

	// If no competitors to move, just return
	if len(existingCompetitors) == 0 {
		return nil
	}

	// Build a map of existing suggested competitor names (case-insensitive) for deduplication
	suggestedNames := make(map[string]bool)
	for _, suggested := range existingSuggestedList {
		name := strings.ToLower(strings.TrimSpace(suggested.Name))
		suggestedNames[name] = true
	}

	// Add all competitors to suggestedList (avoiding duplicates)
	updatedSuggestedList := existingSuggestedList
	for _, comp := range existingCompetitors {
		compNameLower := strings.ToLower(strings.TrimSpace(comp.Name))
		if !suggestedNames[compNameLower] {
			updatedSuggestedList = append(updatedSuggestedList, comp)
			suggestedNames[compNameLower] = true
		}
	}

	// Convert updated suggestedList to storage format
	updatedSuggestedStrings := convertCompetitorsToStorageFormat(updatedSuggestedList)

	// Update the record: clear competitors list, update suggestedList
	brandCompetitors := &models.BrandCompetitors{
		ID:            existing.ID,
		Brand:         existing.Brand,
		Competitors:   []string{},              // Clear competitors list
		SuggestedList: updatedSuggestedStrings, // Add all competitors to suggestedList
		Source:        existing.Source,
		CreatedAt:     existing.CreatedAt,
		UpdatedAt:     time.Now(),
	}

	if err := s.db.SaveBrandCompetitors(ctx, brandCompetitors); err != nil {
		return fmt.Errorf("failed to update competitors: %w", err)
	}

	return nil
}

// DeleteCompetitorByName deletes a specific competitor by name from a brand's list
// Returns the updated competitor list and a message indicating success
func (s *CompetitorService) DeleteCompetitorByName(
	ctx context.Context,
	brandID string,
	competitorName string,
) (*models.DeleteCompetitorResponse, error) {
	if brandID == "" {
		return nil, fmt.Errorf("brandID is required")
	}

	if competitorName == "" {
		return nil, fmt.Errorf("competitor name is required")
	}

	// Get existing competitors
	existing, err := s.db.GetBrandCompetitors(ctx, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing competitors: %w", err)
	}

	if existing == nil {
		return nil, fmt.Errorf("no competitors found for brand: %s", brandID)
	}

	// Convert existing competitors to Competitor objects
	existingCompetitors := convertStringListToCompetitors(existing.Competitors)
	existingSuggestedList := convertStringListToCompetitors(existing.SuggestedList)

	// Find and remove the competitor (case-insensitive match)
	competitorNameLower := strings.ToLower(strings.TrimSpace(competitorName))
	var deletedCompetitor *models.Competitor
	found := false
	updatedCompetitors := make([]models.Competitor, 0, len(existingCompetitors))

	for _, comp := range existingCompetitors {
		compNameLower := strings.ToLower(strings.TrimSpace(comp.Name))
		if compNameLower == competitorNameLower {
			found = true
			// Save the deleted competitor info to add back to suggestedList (make a copy)
			compCopy := comp
			deletedCompetitor = &compCopy
			// Skip this competitor (don't add it to updated list)
			continue
		}
		updatedCompetitors = append(updatedCompetitors, comp)
	}

	if !found {
		return nil, fmt.Errorf("competitor '%s' not found in the list for brand: %s", competitorName, brandID)
	}

	// Add deleted competitor back to suggestedList (if not already present)
	updatedSuggestedList := existingSuggestedList
	if deletedCompetitor != nil {
		// Check if it's already in suggestedList (case-insensitive)
		alreadyInSuggested := false
		deletedNameLower := strings.ToLower(strings.TrimSpace(deletedCompetitor.Name))
		for _, suggested := range existingSuggestedList {
			suggestedNameLower := strings.ToLower(strings.TrimSpace(suggested.Name))
			if suggestedNameLower == deletedNameLower {
				alreadyInSuggested = true
				break
			}
		}

		// Add to suggestedList if not already present
		if !alreadyInSuggested {
			updatedSuggestedList = append(existingSuggestedList, *deletedCompetitor)
		}
	}

	// Convert updated lists back to storage format
	updatedCompetitorStrings := convertCompetitorsToStorageFormat(updatedCompetitors)
	updatedSuggestedStrings := convertCompetitorsToStorageFormat(updatedSuggestedList)

	// Update the brand competitors record
	brandCompetitors := &models.BrandCompetitors{
		ID:            existing.ID,
		Brand:         existing.Brand,
		BrandID:       brandID,
		Competitors:   updatedCompetitorStrings,
		SuggestedList: updatedSuggestedStrings, // Add deleted competitor back to suggestedList
		Source:        existing.Source,
		CreatedAt:     existing.CreatedAt,
		UpdatedAt:     time.Now(),
	}

	if err := s.db.SaveBrandCompetitors(ctx, brandCompetitors); err != nil {
		return nil, fmt.Errorf("failed to update competitors: %w", err)
	}

	return &models.DeleteCompetitorResponse{
		Brand:       brandID,
		DeletedName: competitorName,
		Competitors: updatedCompetitors,
		Message:     fmt.Sprintf("Successfully deleted competitor '%s' from %s", competitorName, brandID),
	}, nil
}

// GetCompetitorsForAnalytics gets the competitor list to use for analytics
// Returns user-defined competitors if available, otherwise returns empty (falls back to auto-detect in analytics)
func (s *CompetitorService) GetCompetitorsForAnalytics(
	ctx context.Context,
	brandID string,
	requestedCompetitors []string,
) ([]string, error) {
	// If specific competitors requested, use those
	if len(requestedCompetitors) > 0 {
		return requestedCompetitors, nil
	}

	// Check for saved competitors
	saved, err := s.db.GetBrandCompetitors(ctx, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saved competitors: %w", err)
	}

	// Use saved competitors if available
	if saved != nil && len(saved.Competitors) > 0 {
		return saved.Competitors, nil
	}

	// Return empty - let analytics auto-detect
	return []string{}, nil
}

// convertStringListToCompetitors converts a list of competitor strings to Competitor objects
// Supports both old format (just name) and new format (name|domain)
func convertStringListToCompetitors(competitorStrings []string) []models.Competitor {
	competitors := make([]models.Competitor, 0, len(competitorStrings))
	for _, str := range competitorStrings {
		var name, domain string

		// Check if it's in "name|domain" format
		if idx := strings.Index(str, "|"); idx != -1 {
			name = strings.TrimSpace(str[:idx])
			domain = strings.TrimSpace(str[idx+1:])
		} else {
			// Old format: just the name, derive domain
			name = str
			domain = deriveCompetitorDomainFromName(name)
		}

		// If domain is empty, derive it
		if domain == "" {
			domain = deriveCompetitorDomainFromName(name)
		}

		competitors = append(competitors, models.Competitor{
			Name:   name,
			Domain: domain,
		})
	}
	return competitors
}

// convertCompetitorsToStorageFormat converts Competitor objects to storage format (name|domain)
func convertCompetitorsToStorageFormat(competitors []models.Competitor) []string {
	strings := make([]string, 0, len(competitors))
	for _, comp := range competitors {
		// Store as "name|domain" format to preserve both
		storageStr := comp.Name
		if comp.Domain != "" {
			storageStr = comp.Name + "|" + comp.Domain
		}
		strings = append(strings, storageStr)
	}
	return strings
}

// deriveCompetitorDomainFromName derives a domain from a competitor name
// This is a simplified version of the dashboard service function
func deriveCompetitorDomainFromName(competitorName string) string {
	// If it already looks like a domain, normalize and return
	if strings.Contains(competitorName, ".") {
		normalized := strings.ToLower(strings.TrimSpace(competitorName))
		// Remove protocol if present
		normalized = strings.TrimPrefix(normalized, "http://")
		normalized = strings.TrimPrefix(normalized, "https://")
		// Remove path if present
		if idx := strings.Index(normalized, "/"); idx != -1 {
			normalized = normalized[:idx]
		}
		// Add www. if not present
		if !strings.HasPrefix(normalized, "www.") {
			return "www." + normalized
		}
		return normalized
	}

	// Clean the competitor name to extract core brand name
	cleaned := cleanCompetitorNameForDomain(competitorName)

	// Convert to lowercase and remove spaces
	normalized := strings.ToLower(strings.TrimSpace(cleaned))

	// Check if the cleaned name already looks like a domain (e.g., from special cases)
	if strings.Contains(normalized, ".") {
		// It's already a domain-like string, add www. prefix and .com suffix if needed
		if !strings.HasPrefix(normalized, "www.") {
			normalized = "www." + normalized
		}
		// Add .com if it doesn't already have a TLD
		if !strings.HasSuffix(normalized, ".com") && !strings.HasSuffix(normalized, ".org") &&
			!strings.HasSuffix(normalized, ".net") && !strings.HasSuffix(normalized, ".io") &&
			!strings.HasSuffix(normalized, ".ai") && !strings.HasSuffix(normalized, ".co") {
			normalized = normalized + ".com"
		}
		return normalized
	}

	// Remove spaces for single-word or multi-word names
	normalized = strings.ReplaceAll(normalized, " ", "")

	// Remove any remaining invalid characters for domain names
	normalized = sanitizeDomainNameForCompetitor(normalized)

	// Construct www.{name}.com
	return "www." + normalized + ".com"
}

// cleanCompetitorNameForDomain removes parentheses, special characters, and extracts core brand name
func cleanCompetitorNameForDomain(name string) string {
	// Remove content in parentheses (e.g., "Windsurf (by Codeium)" -> "Windsurf")
	cleaned := name
	for {
		openIdx := strings.Index(cleaned, "(")
		if openIdx == -1 {
			break
		}
		closeIdx := strings.Index(cleaned[openIdx:], ")")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx
		cleaned = cleaned[:openIdx] + cleaned[closeIdx+1:]
	}

	// Remove common suffixes like " (VS Code)", " - ", etc.
	cleaned = strings.Split(cleaned, " - ")[0]
	cleaned = strings.Split(cleaned, " | ")[0]
	cleaned = strings.TrimSpace(cleaned)

	// For multi-word names, try to extract the main brand name
	// Special handling for known cases
	cleaned = handleSpecialBrandNamesForDomain(cleaned)

	// If it's a special case that returned a domain-like string, return as-is
	if strings.Contains(cleaned, ".") {
		return cleaned
	}

	// For simple names, extract the first word
	words := strings.Fields(cleaned)
	if len(words) >= 1 {
		return words[0]
	}

	return strings.TrimSpace(cleaned)
}

// handleSpecialBrandNamesForDomain handles special cases for well-known brands
func handleSpecialBrandNamesForDomain(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))

	// Special cases for known brands
	specialCases := map[string]string{
		"visual studio code": "code.visualstudio",
		"vs code":            "code.visualstudio",
		"vscode":             "code.visualstudio",
	}

	if domain, ok := specialCases[name]; ok {
		return domain
	}

	// Return the original name (will be processed further)
	return name
}

// sanitizeDomainNameForCompetitor removes invalid characters for domain names
func sanitizeDomainNameForCompetitor(name string) string {
	var result strings.Builder
	for _, r := range name {
		// Allow alphanumeric, hyphens, and dots
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			result.WriteRune(r)
		}
	}
	sanitized := result.String()

	// Remove consecutive dots or hyphens
	sanitized = strings.ReplaceAll(sanitized, "..", ".")
	sanitized = strings.ReplaceAll(sanitized, "--", "-")

	// Remove leading/trailing dots or hyphens
	sanitized = strings.Trim(sanitized, ".-")

	return sanitized
}

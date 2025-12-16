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
// Returns cached suggestions if available, otherwise uses LLM to generate suggestions
func (s *CompetitorService) SuggestCompetitors(
	ctx context.Context,
	brand string,
	website string,
	description string,
	category string,
	forceRefresh bool,
) (*models.SuggestCompetitorsResponse, error) {
	// Check if we already have cached suggestions (unless force refresh)
	if !forceRefresh {
		existing, err := s.db.GetBrandCompetitors(ctx, brand)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing competitors: %w", err)
		}

		// If we have suggested list cached, return it
		if existing != nil && len(existing.SuggestedList) > 0 {
			competitors := convertStringListToCompetitors(existing.SuggestedList)
			return &models.SuggestCompetitorsResponse{
				Brand:       brand,
				Competitors: competitors,
				Source:      "cached",
				Message:     "Returning cached competitor suggestions",
			}, nil
		}
	}

	// Use LLM to suggest competitors
	competitorNames, err := s.suggestCompetitorsWithLLM(ctx, brand, website, description, category)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest competitors: %w", err)
	}

	if len(competitorNames) == 0 {
		return &models.SuggestCompetitorsResponse{
			Brand:       brand,
			Competitors: []models.Competitor{},
			Source:      "llm",
			Message:     "No competitors could be identified. Please provide more details about your brand.",
		}, nil
	}

	// Convert competitor names to Competitor objects with derived domains
	competitors := convertStringListToCompetitors(competitorNames)

	// Cache the suggestions for future use (store as strings for backward compatibility)
	brandCompetitors := &models.BrandCompetitors{
		ID:            uuid.New().String(),
		Brand:         brand,
		Competitors:   []string{},  // Not yet confirmed by user
		SuggestedList: competitorNames, // LLM-suggested list (as strings)
		Source:        "suggested",
		CreatedAt:     time.Now(),
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
	}, nil
}

// suggestCompetitorsWithLLM uses LLM to suggest competitors based on brand info
func (s *CompetitorService) suggestCompetitorsWithLLM(
	ctx context.Context,
	brand string,
	website string,
	description string,
	category string,
) ([]string, error) {
	// Get an LLM provider (prefer Google for latest info)
	provider, ok := s.llmRegistry.Get("google")
	if !ok {
		// Fallback to any available provider
		providers := s.llmRegistry.List()
		if len(providers) == 0 {
			return nil, fmt.Errorf("no LLM providers available")
		}
		provider, _ = s.llmRegistry.Get(providers[0])
	}

	// Scrape website if provided to enrich context
	var websiteContent *WebsiteContent
	if website != "" {
		content, err := s.scraper.ScrapeWebsite(ctx, website)
		if err != nil {
			// Log but don't fail - continue with other info
			fmt.Printf("Warning: failed to scrape website %s: %v\n", website, err)
		} else {
			websiteContent = content
			// If description not provided, use scraped description
			if description == "" && content.Description != "" {
				description = content.Description
			}
		}
	}

	// Build rich context for competitor suggestion
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

	// Add scraped website content for richer context
	if websiteContent != nil {
		if websiteContent.Title != "" {
			contextParts = append(contextParts, fmt.Sprintf("Website Title: %s", websiteContent.Title))
		}
		if websiteContent.Description != "" && description == "" {
			contextParts = append(contextParts, fmt.Sprintf("Website Meta: %s", websiteContent.Description))
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

RESPOND WITH ONLY A JSON ARRAY of competitor names. No explanations, no markdown, just the JSON array.

Example response format:
["Competitor 1", "Competitor 2", "Competitor 3", "Competitor 4", "Competitor 5"]

RESPOND NOW:`, brandContext)

	// Call LLM
	config := llm.Config{
		Temperature: 0.3, // Lower temperature for more focused results
		MaxTokens:   500,
	}

	response, err := provider.Generate(ctx, prompt, config)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse the JSON response
	competitors, err := parseCompetitorResponse(response.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return competitors, nil
}

// parseCompetitorResponse parses the LLM response into a list of competitors
func parseCompetitorResponse(response string) ([]string, error) {
	// Clean up the response
	response = strings.TrimSpace(response)
	
	// Remove markdown code blocks if present
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to parse as JSON array
	var competitors []string
	if err := json.Unmarshal([]byte(response), &competitors); err != nil {
		// If JSON parsing fails, try to extract competitors from text
		competitors = extractCompetitorsFromText(response)
	}

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
func extractCompetitorsFromText(text string) []string {
	var competitors []string
	
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

// SaveCompetitors saves user-defined competitors for a brand
func (s *CompetitorService) SaveCompetitors(
	ctx context.Context,
	brand string,
	competitors []models.Competitor,
	source string,
) (*models.SaveCompetitorsResponse, error) {
	if brand == "" {
		return nil, fmt.Errorf("brand is required")
	}

	if len(competitors) == 0 {
		return nil, fmt.Errorf("at least one competitor is required")
	}

	// Normalize source
	if source == "" {
		source = "custom"
	}

	// Check if we have existing data
	existing, err := s.db.GetBrandCompetitors(ctx, brand)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing competitors: %w", err)
	}

	var suggestedList []string
	var id string
	var createdAt time.Time

	if existing != nil {
		// Preserve suggested list and creation time
		suggestedList = existing.SuggestedList
		id = existing.ID
		createdAt = existing.CreatedAt
	} else {
		id = uuid.New().String()
		createdAt = time.Now()
	}

	// Convert Competitor objects to strings for storage (store as "name|domain" format to preserve domains)
	competitorStrings := convertCompetitorsToStorageFormat(competitors)

	// Create updated competitor list
	brandCompetitors := &models.BrandCompetitors{
		ID:            id,
		Brand:         brand,
		Competitors:   competitorStrings,
		SuggestedList: suggestedList,
		Source:        source,
		CreatedAt:     createdAt,
		UpdatedAt:     time.Now(),
	}

	if err := s.db.SaveBrandCompetitors(ctx, brandCompetitors); err != nil {
		return nil, fmt.Errorf("failed to save competitors: %w", err)
	}

	return &models.SaveCompetitorsResponse{
		Brand:       brand,
		Competitors: competitors,
		Source:      source,
		SavedAt:     brandCompetitors.UpdatedAt,
		Message:     fmt.Sprintf("Successfully saved %d competitors for %s", len(competitors), brand),
	}, nil
}

// GetCompetitors retrieves saved competitors for a brand
func (s *CompetitorService) GetCompetitors(
	ctx context.Context,
	brand string,
) (*models.GetCompetitorsResponse, error) {
	if brand == "" {
		return nil, fmt.Errorf("brand is required")
	}

	competitors, err := s.db.GetBrandCompetitors(ctx, brand)
	if err != nil {
		return nil, fmt.Errorf("failed to get competitors: %w", err)
	}

	if competitors == nil {
		return &models.GetCompetitorsResponse{
			Brand:       brand,
			Competitors: []models.Competitor{},
			Source:      "none",
			UpdatedAt:   time.Now(),
		}, nil
	}

	// Convert string lists to Competitor objects
	competitorList := convertStringListToCompetitors(competitors.Competitors)
	suggestedList := convertStringListToCompetitors(competitors.SuggestedList)

	return &models.GetCompetitorsResponse{
		Brand:         brand,
		Competitors:   competitorList,
		SuggestedList: suggestedList,
		Source:        competitors.Source,
		UpdatedAt:     competitors.UpdatedAt,
	}, nil
}

// DeleteCompetitors deletes saved competitors for a brand
func (s *CompetitorService) DeleteCompetitors(
	ctx context.Context,
	brand string,
) error {
	if brand == "" {
		return fmt.Errorf("brand is required")
	}

	return s.db.DeleteBrandCompetitors(ctx, brand)
}

// GetCompetitorsForAnalytics gets the competitor list to use for analytics
// Returns user-defined competitors if available, otherwise returns empty (falls back to auto-detect in analytics)
func (s *CompetitorService) GetCompetitorsForAnalytics(
	ctx context.Context,
	brand string,
	requestedCompetitors []string,
) ([]string, error) {
	// If specific competitors requested, use those
	if len(requestedCompetitors) > 0 {
		return requestedCompetitors, nil
	}

	// Check for saved competitors
	saved, err := s.db.GetBrandCompetitors(ctx, brand)
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

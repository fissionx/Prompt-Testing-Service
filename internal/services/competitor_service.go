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
			return &models.SuggestCompetitorsResponse{
				Brand:       brand,
				Competitors: existing.SuggestedList,
				Source:      "cached",
				Message:     "Returning cached competitor suggestions",
			}, nil
		}
	}

	// Use LLM to suggest competitors
	competitors, err := s.suggestCompetitorsWithLLM(ctx, brand, website, description, category)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest competitors: %w", err)
	}

	if len(competitors) == 0 {
		return &models.SuggestCompetitorsResponse{
			Brand:       brand,
			Competitors: []string{},
			Source:      "llm",
			Message:     "No competitors could be identified. Please provide more details about your brand.",
		}, nil
	}

	// Cache the suggestions for future use
	brandCompetitors := &models.BrandCompetitors{
		ID:            uuid.New().String(),
		Brand:         brand,
		Competitors:   []string{},  // Not yet confirmed by user
		SuggestedList: competitors, // LLM-suggested list
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
	competitors []string,
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

	// Create updated competitor list
	brandCompetitors := &models.BrandCompetitors{
		ID:            id,
		Brand:         brand,
		Competitors:   competitors,
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
			Competitors: []string{},
			Source:      "none",
			UpdatedAt:   time.Now(),
		}, nil
	}

	return &models.GetCompetitorsResponse{
		Brand:         brand,
		Competitors:   competitors.Competitors,
		SuggestedList: competitors.SuggestedList,
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

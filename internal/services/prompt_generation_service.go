package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/utils"

	// "github.com/fissionx/gego/internal/llm/anthropic"
	// "github.com/fissionx/gego/internal/llm/google"
	// "github.com/fissionx/gego/internal/llm/ollama"
	"github.com/fissionx/gego/internal/llm/openai"
	// "github.com/fissionx/gego/internal/llm/perplexity"
	"github.com/fissionx/gego/internal/models"
)

// PromptGenerationService handles intelligent prompt generation and reuse
type PromptGenerationService struct {
	db          db.Database
	llmRegistry *llm.Registry
	scraper     *WebScraperService
}

// NewPromptGenerationService creates a new prompt generation service
func NewPromptGenerationService(database db.Database, registry *llm.Registry) *PromptGenerationService {
	return &PromptGenerationService{
		db:          database,
		llmRegistry: registry,
		scraper:     NewWebScraperService(),
	}
}

// GeneratePromptsForBrand generates prompts for a brand, reusing existing ones where possible
// llmConfig is optional - if not provided, defaults to Google/Gemini
func (s *PromptGenerationService) GeneratePromptsForBrand(ctx context.Context, brandID string, orgID string, brand, website, category, domain, description string, count int, llmConfig *models.LLMConfig) ([]models.Prompt, int, int, error) {
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}
	fmt.Printf("🔍 Generating prompts: brandID=%s, orgID=%s, brand=%s, website=%s, category=%s, domain=%s, description=%s, count=%d\n", brandID, orgID, brand, website, category, domain, description, count)

	// Step 1: Scrape website if provided to enrich context
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

	// Step 2: Derive domain/category if not provided (using LLM if needed)
	// Since we now use API to get brand details, we derive domain/category directly
	if domain == "" || category == "" {
		derivedDomain, derivedCategory, err := s.deriveBrandMetadata(ctx, brand, description, websiteContent, llmConfig)
		if err != nil {
			// Fallback to defaults if derivation fails
			if domain == "" {
				domain = "general"
			}
			if category == "" {
				category = "general"
			}
		} else {
			if domain == "" {
				domain = derivedDomain
			}
			if category == "" {
				category = derivedCategory
			}
		}
	}

	fmt.Printf("🔍 Generating prompts: brand=%s, domain=%s, category=%s\n", brand, domain, category)

	// Step 3: Always generate new prompts using LLM for accuracy and freshness
	newPrompts, err := s.generateNewPrompts(ctx, brand, category, domain, description, websiteContent, count, nil, llmConfig)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to generate prompts: %w", err)
	}

	// Step 4: Save prompts to database
	savedPrompts, err := s.savePrompts(ctx, newPrompts, brandID, orgID, brand, category, domain)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to save prompts: %w", err)
	}

	fmt.Printf("✅ Generated and saved %d new prompts for brand=%s\n", len(savedPrompts), brand)

	return savedPrompts, 0, len(savedPrompts), nil
}

// createProviderFromConfig creates an LLM provider instance from LLMConfig
func (s *PromptGenerationService) createProviderFromConfig(llmConfig *models.LLMConfig) (llm.Provider, error) {
	if llmConfig == nil {
		// Default to Google/Gemini if no LLM config provided
		provider, ok := s.llmRegistry.Get("google")
		if !ok {
			// Fallback to any available provider
			providers := s.llmRegistry.List()
			if len(providers) == 0 {
				return nil, fmt.Errorf("no LLM providers available")
			}
			provider, _ = s.llmRegistry.Get(providers[0])
		}
		return provider, nil
	}

	// Create provider with specific API key from config
	var provider llm.Provider
	switch llmConfig.Provider {
	case "openai":
		provider = openai.New(llmConfig.APIKey, llmConfig.BaseURL)
	// case "anthropic":
	// 	provider = anthropic.New(llmConfig.APIKey, llmConfig.BaseURL)
	// case "ollama":
	// 	provider = ollama.New(llmConfig.BaseURL)
	// case "google":
	// 	provider = google.New(llmConfig.APIKey, llmConfig.BaseURL)
	// case "perplexity":
	// 	provider = perplexity.New(llmConfig.APIKey, llmConfig.BaseURL)
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", llmConfig.Provider)
	}

	return provider, nil
}

// deriveBrandMetadata uses LLM to derive domain and category for a brand
func (s *PromptGenerationService) deriveBrandMetadata(ctx context.Context, brand, description string, websiteContent *WebsiteContent, llmConfig *models.LLMConfig) (string, string, error) {
	// Create provider from config
	provider, err := s.createProviderFromConfig(llmConfig)
	if err != nil {
		return "", "", err
	}

	var model string
	if llmConfig != nil {
		model = llmConfig.Model
	}

	// Build rich context from available sources
	var contextParts []string
	contextParts = append(contextParts, fmt.Sprintf("Brand: %s", brand))

	if description != "" {
		contextParts = append(contextParts, fmt.Sprintf("Description: %s", description))
	}

	// Add scraped website content for much richer context
	if websiteContent != nil {
		if websiteContent.Title != "" {
			contextParts = append(contextParts, fmt.Sprintf("Website Title: %s", websiteContent.Title))
		}
		if websiteContent.Description != "" {
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

	derivationPrompt := utils.BrandMetadataDerivationPrompt(brandContext)

	response, err := provider.Generate(ctx, derivationPrompt, llm.Config{
		Model:       model,
		Temperature: 0.3, // Low temperature for consistent categorization
		MaxTokens:   200,
	})
	if err != nil {
		return "", "", err
	}

	// Parse the response
	lines := strings.Split(response.Text, "\n")
	domain := "general"
	category := "general"

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "domain:") {
			domain = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "domain:"))
		} else if strings.HasPrefix(strings.ToLower(line), "category:") {
			category = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "category:"))
		}
	}

	// Normalize categories for consistency
	domain = normalizeCategory(domain)
	category = normalizeCategory(category)

	fmt.Printf("🤖 AI derived metadata for '%s': domain=%s, category=%s\n", brand, domain, category)

	return domain, category, nil
}

// normalizeCategory standardizes common category variations for consistent reuse
func normalizeCategory(cat string) string {
	cat = strings.TrimSpace(strings.ToLower(cat))

	// Normalize common education variations
	educationPatterns := map[string]string{
		"engineering college":          "engineering college",
		"technical university":         "engineering college",
		"institute of technology":      "engineering college",
		"higher education institution": "higher education",
		"university":                   "higher education",
		"college":                      "higher education",
		"business school":              "business school",
		"management institute":         "business school",

		// Technology variations
		"ai tool":      "ai tools",
		"ai tools":     "ai tools",
		"ai platform":  "ai tools",
		"seo tool":     "seo tools",
		"seo tools":    "seo tools",
		"seo platform": "seo tools",
		"crm":          "crm software",
		"crm software": "crm software",
		"crm platform": "crm software",

		// Healthcare variations
		"hospital":            "hospital",
		"medical center":      "hospital",
		"healthcare facility": "hospital",
		"clinic":              "clinic",

		// Finance variations
		"payment gateway":   "payment platform",
		"payment processor": "payment platform",
		"payment platform":  "payment platform",
	}

	// Check for exact matches first
	if normalized, ok := educationPatterns[cat]; ok {
		return normalized
	}

	// Check for partial matches
	for pattern, normalized := range educationPatterns {
		if strings.Contains(cat, pattern) {
			return normalized
		}
	}

	return cat
}

// calculatePromptTypeDistribution calculates how many prompts of each type to generate
func calculatePromptTypeDistribution(total int) map[string]int {
	distribution := make(map[string]int)

	// Base distribution (proportional)
	distribution["what"] = total / 5       // 20%
	distribution["how"] = total / 5        // 20%
	distribution["comparison"] = total / 5 // 20%
	distribution["top_best"] = total / 5   // 20%
	distribution["brand"] = total / 5      // 20%

	// Distribute remainder
	remainder := total - (distribution["what"] + distribution["how"] + distribution["comparison"] + distribution["top_best"] + distribution["brand"])

	// Add remainder to most useful types
	types := []string{"top_best", "how", "what", "comparison", "brand"}
	for i := 0; i < remainder; i++ {
		distribution[types[i%len(types)]]++
	}

	return distribution
}

// generateNewPrompts generates new prompts using an LLM with the new structured format
// Uses the shared generatePromptsWithLLM function
func (s *PromptGenerationService) generateNewPrompts(ctx context.Context, brand, category, domain, description string, websiteContent *WebsiteContent, count int, existingPrompts []models.Prompt, llmConfig *models.LLMConfig) ([]PromptGenerationResult, error) {
	// Create provider from config
	provider, err := s.createProviderFromConfig(llmConfig)
	if err != nil {
		return nil, err
	}

	var model string
	if llmConfig != nil {
		model = llmConfig.Model
	}

	// Use shared function for prompt generation
	return generatePromptsWithLLM(ctx, provider, model, brand, websiteContent, category, description, "", count, "")
}

// parsePromptType extracts prompt type from prefix (e.g., "WHAT|question" → "what", "question")
func parsePromptType(text string) (models.PromptType, string) {
	prefixMap := map[string]models.PromptType{
		"WHAT|":    models.PromptTypeWhat,
		"HOW|":     models.PromptTypeHow,
		"COMPARE|": models.PromptTypeComparison,
		"TOPBEST|": models.PromptTypeTopBest,
		"BRAND|":   models.PromptTypeBrand,
	}

	for prefix, promptType := range prefixMap {
		if strings.HasPrefix(text, prefix) {
			cleanText := strings.TrimPrefix(text, prefix)
			return promptType, strings.TrimSpace(cleanText)
		}
	}

	// If no prefix found, try to infer from question content
	lowerText := strings.ToLower(text)
	if strings.HasPrefix(lowerText, "what ") {
		return models.PromptTypeWhat, text
	} else if strings.HasPrefix(lowerText, "how ") {
		return models.PromptTypeHow, text
	} else if strings.Contains(lowerText, " vs ") || strings.Contains(lowerText, " versus ") || strings.Contains(lowerText, "compare") {
		return models.PromptTypeComparison, text
	} else if strings.HasPrefix(lowerText, "best ") || strings.HasPrefix(lowerText, "top ") || strings.Contains(lowerText, "most popular") {
		return models.PromptTypeTopBest, text
	}

	// Default to "what" type if can't determine
	return models.PromptTypeWhat, text
}

// savePrompts saves generated prompts to the database
func (s *PromptGenerationService) savePrompts(ctx context.Context, promptResults []PromptGenerationResult, brandID string, orgID string, brand, category, domain string) ([]models.Prompt, error) {
	var savedPrompts []models.Prompt

	for _, result := range promptResults {
		// Map intentType to PromptType
		promptType := mapIntentTypeToPromptType(result.IntentType)

		// If intentType is empty or mapping failed, try to infer from prompt text
		if promptType == models.PromptTypeCustom && result.Prompt != "" {
			promptType, _ = parsePromptType(result.Prompt)
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
			Enabled:                 false, // Generated prompts are suggestions - only enabled when saved via /save API
			TargetingSearchKeywords: result.TargetingSearchKeywords,
			SupportingFanoutQueries: result.SupportingFanoutQueries,
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		}

		if err := s.db.CreatePrompt(ctx, prompt); err != nil {
			// Log error but continue with other prompts
			continue
		}

		savedPrompts = append(savedPrompts, *prompt)
	}

	return savedPrompts, nil
}

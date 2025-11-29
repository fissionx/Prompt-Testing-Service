package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/AI2HU/gego/internal/db"
	"github.com/AI2HU/gego/internal/llm"
	"github.com/AI2HU/gego/internal/models"
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
func (s *PromptGenerationService) GeneratePromptsForBrand(ctx context.Context, brand, website, category, domain, description string, count int) ([]models.Prompt, int, int, error) {
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

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

	// Step 2: Get or derive brand profile (determines domain/category if not provided)
	brandProfile, err := s.getOrCreateBrandProfile(ctx, brand, website, category, domain, description, websiteContent)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get brand profile: %w", err)
	}

	// Use profile's domain/category if not explicitly provided
	if domain == "" && brandProfile.Domain != "" {
		domain = brandProfile.Domain
	}
	if category == "" && brandProfile.Category != "" {
		category = brandProfile.Category
	}

	fmt.Printf("🔍 Looking for prompts: brand=%s, domain=%s, category=%s\n", brand, domain, category)

	// Step 3: Check if prompt library exists for this domain/category (NOT brand-specific)
	// This allows reuse across similar brands (e.g., all engineering colleges)
	library, err := s.db.GetPromptLibrary(ctx, "", domain, category)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to check prompt library: %w", err)
	}

	// If library exists, return those prompts (after validation)
	if library != nil && len(library.PromptIDs) > 0 {
		fmt.Printf("♻️  Checking existing prompt library for domain=%s, category=%s (created for: %s)\n", domain, category, library.Brand)
		
		prompts, err := s.getPromptsFromLibrary(ctx, library, count, brand)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to get prompts from library: %w", err)
		}

		// Only reuse if we got enough generic prompts (at least 70% of requested)
		minRequired := int(float64(count) * 0.7)
		if len(prompts) >= minRequired {
			fmt.Printf("✅ Reusing %d generic prompts from library\n", len(prompts))
			
			// Increment library usage count
			library.UsageCount++
			_ = s.db.UpdatePromptLibrary(ctx, library)

			return prompts, len(prompts), 0, nil
		} else {
			fmt.Printf("⚠️  Library has too many brand-specific prompts (%d generic out of %d needed). Generating new prompts instead.\n", len(prompts), minRequired)
			// Fall through to generate new prompts
		}
	}

	// Step 4: No library exists, generate new prompts with enriched context
	newPrompts, err := s.generateNewPrompts(ctx, brand, category, domain, description, websiteContent, count, nil)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to generate prompts: %w", err)
	}

	// Step 4: Save prompts to database
	savedPrompts, err := s.savePrompts(ctx, newPrompts, brand, category, domain)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to save prompts: %w", err)
	}

	// Step 5: Create prompt library entry (brand is for reference only, library is shared by domain/category)
	promptIDs := make([]string, len(savedPrompts))
	for i, p := range savedPrompts {
		promptIDs[i] = p.ID
	}

	newLibrary := &models.PromptLibrary{
		ID:         uuid.New().String(),
		Brand:      brand, // Track which brand first created this library
		Domain:     domain,
		Category:   category,
		PromptIDs:  promptIDs,
		UsageCount: 1,
	}

	fmt.Printf("📚 Creating new prompt library: domain=%s, category=%s, created_by=%s\n", domain, category, brand)

	if err := s.db.CreatePromptLibrary(ctx, newLibrary); err != nil {
		// Log but don't fail - prompts are already saved
		fmt.Printf("❌ Warning: failed to create prompt library: %v\n", err)
	} else {
		fmt.Printf("✅ Prompt library created successfully! Future requests for domain=%s + category=%s will reuse these prompts.\n", domain, category)
	}

	return savedPrompts, 0, len(savedPrompts), nil
}

// getOrCreateBrandProfile gets existing profile or derives one using LLM
func (s *PromptGenerationService) getOrCreateBrandProfile(ctx context.Context, brand, website, category, domain, description string, websiteContent *WebsiteContent) (*models.BrandProfile, error) {
	// Check if profile already exists
	profile, err := s.db.GetBrandProfile(ctx, brand)
	if err != nil {
		return nil, err
	}

	if profile != nil {
		// Update website if provided and not set
		if website != "" && profile.Website == "" {
			profile.Website = website
			_ = s.db.UpdateBrandProfile(ctx, profile)
		}
		return profile, nil
	}

	// Profile doesn't exist - create one
	// If domain and category provided, use them
	if domain != "" && category != "" {
		profile = &models.BrandProfile{
			ID:          uuid.New().String(),
			BrandName:   brand,
			Website:     website,
			Domain:      domain,
			Category:    category,
			Description: description,
		}

		if err := s.db.CreateBrandProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("failed to create brand profile: %w", err)
		}

		return profile, nil
	}

	// Otherwise, derive domain/category using LLM with enriched context
	derivedDomain, derivedCategory, err := s.deriveBrandMetadata(ctx, brand, description, websiteContent)
	if err != nil {
		// Fallback to defaults if derivation fails
		derivedDomain = "general"
		derivedCategory = "general"
	}

	profile = &models.BrandProfile{
		ID:          uuid.New().String(),
		BrandName:   brand,
		Website:     website,
		Domain:      derivedDomain,
		Category:    derivedCategory,
		Description: description,
	}

	if err := s.db.CreateBrandProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create brand profile: %w", err)
	}

	return profile, nil
}

// deriveBrandMetadata uses LLM to derive domain and category for a brand
func (s *PromptGenerationService) deriveBrandMetadata(ctx context.Context, brand, description string, websiteContent *WebsiteContent) (string, string, error) {
	// Get an LLM for metadata derivation
	provider, ok := s.llmRegistry.Get("google")
	if !ok {
		providers := s.llmRegistry.List()
		if len(providers) == 0 {
			return "", "", fmt.Errorf("no LLM providers available")
		}
		provider, _ = s.llmRegistry.Get(providers[0])
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

	derivationPrompt := fmt.Sprintf(`Analyze this brand based on the provided information and determine its industry domain and BROAD category.

IMPORTANT: Use BROAD, GENERIC categories that apply to many similar organizations. DO NOT be too specific.

%s

Respond in EXACTLY this format (one line for domain, one for category):
Domain: <industry domain like "technology", "healthcare", "finance", "retail", "education", etc>
Category: <BROAD category that would apply to similar organizations>

Examples of GOOD (broad) categories:
- Domain: technology, Category: ai tools
- Domain: technology, Category: crm software
- Domain: education, Category: engineering college
- Domain: education, Category: business school
- Domain: healthcare, Category: hospital
- Domain: finance, Category: payment platform

Examples of BAD (too specific) categories:
- "premier higher education institution" ❌ (too specific, use "engineering college")
- "technical university in south asia" ❌ (too specific, use "engineering college")
- "AI-powered content optimization platform" ❌ (too specific, use "ai tools")

Choose the BROADEST category that accurately describes what this organization does.`, brandContext)

	response, err := provider.Generate(ctx, derivationPrompt, llm.Config{
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

// getPromptsFromLibrary retrieves prompts from a library and validates they are generic
func (s *PromptGenerationService) getPromptsFromLibrary(ctx context.Context, library *models.PromptLibrary, count int, currentBrand string) ([]models.Prompt, error) {
	var prompts []models.Prompt

	// Get all prompts from the library and filter out brand-specific ones
	for _, promptID := range library.PromptIDs {
		if len(prompts) >= count {
			break // Got enough prompts
		}

		prompt, err := s.db.GetPrompt(ctx, promptID)
		if err != nil {
			continue // Skip if prompt not found
		}

		// Validate that prompt is generic (doesn't mention specific brands)
		if isGenericPrompt(prompt.Template, library.Brand, currentBrand) {
			prompts = append(prompts, *prompt)
		} else {
			fmt.Printf("⚠️  Skipping brand-specific prompt: %s\n", prompt.Template)
		}
	}

	// If we don't have enough generic prompts, return what we have
	if len(prompts) < count {
		fmt.Printf("⚠️  Only found %d generic prompts out of %d requested. Library needs regeneration.\n", len(prompts), count)
	}

	return prompts, nil
}

// isGenericPrompt checks if a prompt is generic (doesn't mention specific brand names)
func isGenericPrompt(promptText, originalBrand, currentBrand string) bool {
	lowerPrompt := strings.ToLower(promptText)
	
	// Check if prompt mentions the original brand that created it
	if originalBrand != "" {
		// Split brand name into words to check each part
		brandWords := strings.Fields(strings.ToLower(originalBrand))
		for _, word := range brandWords {
			// Skip common words that might appear in generic questions
			if len(word) <= 3 || isCommonWord(word) {
				continue
			}
			if strings.Contains(lowerPrompt, word) {
				return false // Found brand-specific mention
			}
		}
	}
	
	// Check if prompt mentions the current brand
	if currentBrand != "" {
		brandWords := strings.Fields(strings.ToLower(currentBrand))
		for _, word := range brandWords {
			if len(word) <= 3 || isCommonWord(word) {
				continue
			}
			if strings.Contains(lowerPrompt, word) {
				return false
			}
		}
	}
	
	return true
}

// isCommonWord checks if a word is too common to be considered brand-specific
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "with": true,
		"from": true, "that": true, "this": true, "what": true, "which": true,
		"best": true, "good": true, "top": true, "how": true, "can": true,
		"college": true, "university": true, "institute": true, "school": true,
		"engineering": true, "technology": true, "software": true, "platform": true,
	}
	return commonWords[word]
}

// generateNewPrompts generates new prompts using an LLM
func (s *PromptGenerationService) generateNewPrompts(ctx context.Context, brand, category, domain, description string, websiteContent *WebsiteContent, count int, existingPrompts []models.Prompt) ([]string, error) {
	// Get a capable LLM for generation (prefer Google for latest info)
	provider, ok := s.llmRegistry.Get("google")
	if !ok {
		// Fallback to any available provider
		providers := s.llmRegistry.List()
		if len(providers) == 0 {
			return nil, fmt.Errorf("no LLM providers available")
		}
		provider, _ = s.llmRegistry.Get(providers[0])
	}

	// Build the generation prompt
	existingText := ""
	if len(existingPrompts) > 0 {
		var templates []string
		for _, p := range existingPrompts {
			templates = append(templates, p.Template)
		}
		existingText = fmt.Sprintf("\n\nEXISTING PROMPTS (avoid duplication):\n%s", strings.Join(templates, "\n"))
	}

	// Build rich brand context
	brandInfo := fmt.Sprintf("Brand: %s", brand)
	if category != "" {
		brandInfo += fmt.Sprintf("\nCategory: %s", category)
	}
	if domain != "" {
		brandInfo += fmt.Sprintf("\nDomain/Industry: %s", domain)
	}
	if description != "" {
		brandInfo += fmt.Sprintf("\nDescription: %s", description)
	}

	// Add scraped website content for hyper-realistic prompts
	if websiteContent != nil {
		brandInfo += "\n\n=== WEBSITE CONTENT (use this to understand what they actually do) ==="
		if websiteContent.Title != "" {
			brandInfo += fmt.Sprintf("\nWebsite: %s", websiteContent.Title)
		}
		if websiteContent.Description != "" {
			brandInfo += fmt.Sprintf("\nTagline: %s", websiteContent.Description)
		}
		if len(websiteContent.Keywords) > 0 {
			brandInfo += fmt.Sprintf("\nKeywords: %s", strings.Join(websiteContent.Keywords, ", "))
		}
		if websiteContent.MainContent != "" {
			content := websiteContent.MainContent
			if len(content) > 1000 {
				content = content[:1000] + "..."
			}
			brandInfo += fmt.Sprintf("\nMain Content: %s", content)
		}
	}

	generationPrompt := fmt.Sprintf(`Generate %d unique, natural questions that people would ask when searching for products/services in this space.

%s%s

CRITICAL REQUIREMENTS:
1. Questions MUST BE GENERIC - DO NOT mention the specific brand name "%s"
2. Questions should apply to ANY brand in this category (e.g., "engineering college" not "MIT")
3. Questions should be diverse and natural (how people actually search)
4. Mix different query types: comparisons, recommendations, how-to, best practices, reviews
5. Questions should be likely to generate responses mentioning brands in this space
6. Avoid duplicating existing prompts
7. Each question should be on a new line
8. DO NOT include numbers, bullets, or any formatting - just the questions

Examples of GOOD (generic) questions:
- "What are the best engineering colleges in India?"
- "How to choose a good engineering college?"
- "Which engineering college has the best placement record?"
- "What are the admission requirements for engineering colleges?"
- "How do top engineering colleges compare in terms of faculty?"

Examples of BAD (too specific) questions:
- "What are the facilities at MIT?" ❌ (mentions specific brand)
- "How is the campus life at Stanford?" ❌ (mentions specific brand)
- "What courses does Harvard offer?" ❌ (mentions specific brand)

Generate exactly %d GENERIC questions that apply to the entire category, one per line:`, count, brandInfo, existingText, brand, count)

	response, err := provider.Generate(ctx, generationPrompt, llm.Config{
		Model:       "",
		Temperature: 0.9, // High creativity for diverse prompts
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, err
	}

	// Parse the response - split by newlines
	lines := strings.Split(response.Text, "\n")
	var prompts []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines
		if line == "" {
			continue
		}
		// Remove any numbering or bullet points
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimPrefix(line, "•")
		// Remove leading numbers like "1. " or "1) "
		for i := 0; i < 10; i++ {
			line = strings.TrimPrefix(line, fmt.Sprintf("%d. ", i))
			line = strings.TrimPrefix(line, fmt.Sprintf("%d) ", i))
		}
		line = strings.TrimSpace(line)

		if line != "" && len(line) > 10 {
			prompts = append(prompts, line)
		}
	}

	if len(prompts) == 0 {
		return nil, fmt.Errorf("failed to parse generated prompts from LLM response")
	}

	return prompts, nil
}

// savePrompts saves generated prompts to the database
func (s *PromptGenerationService) savePrompts(ctx context.Context, promptTexts []string, brand, category, domain string) ([]models.Prompt, error) {
	var savedPrompts []models.Prompt

	for _, text := range promptTexts {
		prompt := &models.Prompt{
			ID:        uuid.New().String(),
			Template:  text,
			Category:  category,
			Domain:    domain,
			Brand:     brand,
			Generated: true,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.db.CreatePrompt(ctx, prompt); err != nil {
			// Log error but continue with other prompts
			continue
		}

		savedPrompts = append(savedPrompts, *prompt)
	}

	return savedPrompts, nil
}

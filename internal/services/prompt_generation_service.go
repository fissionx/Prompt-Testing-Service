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

		existingPrompts, err := s.getPromptsFromLibrary(ctx, library, count, brand)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to get prompts from library: %w", err)
		}

		existingCount := len(existingPrompts)

		// If we have enough prompts, just return them
		if existingCount >= count {
			fmt.Printf("✅ Reusing %d generic prompts from library\n", existingCount)

			// Increment library usage count
			library.UsageCount++
			_ = s.db.UpdatePromptLibrary(ctx, library)

			return existingPrompts[:count], existingCount, 0, nil
		}

		// If we have some prompts but not enough, generate the difference
		if existingCount > 0 {
			needToGenerate := count - existingCount
			fmt.Printf("♻️  Found %d generic prompts, generating %d more to reach %d total\n", existingCount, needToGenerate, count)

			// Generate additional prompts
			newPromptTexts, err := s.generateNewPrompts(ctx, brand, category, domain, description, websiteContent, needToGenerate, existingPrompts)
			if err != nil {
				// If generation fails, return what we have
				fmt.Printf("⚠️  Failed to generate additional prompts: %v. Returning %d existing prompts.\n", err, existingCount)
				return existingPrompts, existingCount, 0, nil
			}

			// Save new prompts
			savedNewPrompts, err := s.savePrompts(ctx, newPromptTexts, brand, category, domain)
			if err != nil {
				return existingPrompts, existingCount, 0, nil
			}

			// Add new prompt IDs to the library
			for _, p := range savedNewPrompts {
				library.PromptIDs = append(library.PromptIDs, p.ID)
			}
			library.UsageCount++
			_ = s.db.UpdatePromptLibrary(ctx, library)

			// Combine existing and new prompts
			allPrompts := append(existingPrompts, savedNewPrompts...)
			fmt.Printf("✅ Using %d existing + %d newly generated = %d total prompts\n", existingCount, len(savedNewPrompts), len(allPrompts))

			return allPrompts, existingCount, len(savedNewPrompts), nil
		}

		// If no generic prompts found, fall through to generate all new
		fmt.Printf("⚠️  No generic prompts found in library. Generating all new prompts.\n")
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

	// Calculate distribution across question types for diversity
	distribution := calculatePromptTypeDistribution(count)

	generationPrompt := fmt.Sprintf(`Generate %d unique, natural questions that people would ask when searching for products/services in this space.

%s%s

CRITICAL REQUIREMENTS:
1. Questions MUST BE GENERIC - DO NOT mention the specific brand name "%s"
2. Questions should apply to ANY brand in this category
3. Questions MUST be balanced across these 5 types:

TYPE 1: WHAT QUESTIONS (%d questions) [PREFIX: WHAT|]
- Definitional, explanatory questions
- Examples: "What is GEO?", "What are the benefits of..."
- Format: WHAT|What is GEO and why is it important?

TYPE 2: HOW QUESTIONS (%d questions) [PREFIX: HOW|]
- Instructional, process-oriented questions
- Examples: "How to appear in AI search?", "How does X work?"
- Format: HOW|How to optimize content for AI search?

TYPE 3: COMPARISON QUESTIONS (%d questions) [PREFIX: COMPARE|]
- Competitive analysis, versus questions
- Examples: "X vs Y", "Which is better for...", "How does X compare to Y?"
- Format: COMPARE|How does brand A compare to brand B for SEO?

TYPE 4: TOP/BEST QUESTIONS (%d questions) [PREFIX: TOPBEST|]
- List-based, recommendation questions
- Examples: "Best AI SEO tools", "Top platforms for...", "Most popular..."
- Format: TOPBEST|What are the best AI SEO tools for small businesses?

TYPE 5: BRAND-SPECIFIC PATTERN (%d questions) [PREFIX: BRAND|]
- Questions that follow "what does [BRAND] do" pattern (but keep generic)
- Examples: "What features do [category] platforms offer?", "How do [category] tools help?"
- Format: BRAND|What features do AI SEO platforms typically offer?

IMPORTANT:
- Use EXACT prefixes (WHAT|, HOW|, COMPARE|, TOPBEST|, BRAND|)
- Generate the EXACT count for each type as specified above
- Questions must be generic (no specific brand names)
- One question per line
- NO numbers, bullets, or extra formatting

Generate exactly %d questions in this format:`, count, brandInfo, existingText, brand,
		distribution["what"], distribution["how"], distribution["comparison"], distribution["top_best"], distribution["brand"], count)

	response, err := provider.Generate(ctx, generationPrompt, llm.Config{
		Model:       "",
		Temperature: 0.9, // High creativity for diverse prompts
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, err
	}

	// Parse the response - split by newlines and extract type prefix
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
		for i := 0; i < 100; i++ {
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
func (s *PromptGenerationService) savePrompts(ctx context.Context, promptTexts []string, brand, category, domain string) ([]models.Prompt, error) {
	var savedPrompts []models.Prompt

	for _, text := range promptTexts {
		// Parse prompt type from prefix
		promptType, cleanText := parsePromptType(text)

		prompt := &models.Prompt{
			ID:         uuid.New().String(),
			Template:   cleanText,
			PromptType: promptType,
			Category:   category,
			Domain:     domain,
			Brand:      brand,
			Generated:  true,
			Enabled:    true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := s.db.CreatePrompt(ctx, prompt); err != nil {
			// Log error but continue with other prompts
			continue
		}

		savedPrompts = append(savedPrompts, *prompt)
	}

	return savedPrompts, nil
}

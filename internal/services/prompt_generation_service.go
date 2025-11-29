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
}

// NewPromptGenerationService creates a new prompt generation service
func NewPromptGenerationService(database db.Database, registry *llm.Registry) *PromptGenerationService {
	return &PromptGenerationService{
		db:          database,
		llmRegistry: registry,
	}
}

// GeneratePromptsForBrand generates prompts for a brand, reusing existing ones where possible
func (s *PromptGenerationService) GeneratePromptsForBrand(ctx context.Context, brand, category, domain, description string, count int) ([]models.Prompt, int, int, error) {
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	// Step 1: Get or derive brand profile (determines domain/category if not provided)
	brandProfile, err := s.getOrCreateBrandProfile(ctx, brand, category, domain, description)
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

	// Step 2: Check if prompt library exists for this brand/domain/category
	library, err := s.db.GetPromptLibrary(ctx, brand, domain, category)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to check prompt library: %w", err)
	}

	// If library exists, return those prompts
	if library != nil && len(library.PromptIDs) > 0 {
		prompts, err := s.getPromptsFromLibrary(ctx, library, count)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to get prompts from library: %w", err)
		}

		// Increment library usage count
		library.UsageCount++
		_ = s.db.UpdatePromptLibrary(ctx, library)

		return prompts, len(prompts), 0, nil
	}

	// Step 3: No library exists, generate new prompts
	newPrompts, err := s.generateNewPrompts(ctx, brand, category, domain, description, count, nil)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to generate prompts: %w", err)
	}

	// Step 4: Save prompts to database
	savedPrompts, err := s.savePrompts(ctx, newPrompts, brand, category, domain)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to save prompts: %w", err)
	}

	// Step 5: Create prompt library entry
	promptIDs := make([]string, len(savedPrompts))
	for i, p := range savedPrompts {
		promptIDs[i] = p.ID
	}

	newLibrary := &models.PromptLibrary{
		ID:         uuid.New().String(),
		Brand:      brand,
		Domain:     domain,
		Category:   category,
		PromptIDs:  promptIDs,
		UsageCount: 1,
	}

	if err := s.db.CreatePromptLibrary(ctx, newLibrary); err != nil {
		// Log but don't fail - prompts are already saved
		fmt.Printf("Warning: failed to create prompt library: %v\n", err)
	}

	return savedPrompts, 0, len(savedPrompts), nil
}

// getOrCreateBrandProfile gets existing profile or derives one using LLM
func (s *PromptGenerationService) getOrCreateBrandProfile(ctx context.Context, brand, category, domain, description string) (*models.BrandProfile, error) {
	// Check if profile already exists
	profile, err := s.db.GetBrandProfile(ctx, brand)
	if err != nil {
		return nil, err
	}

	if profile != nil {
		return profile, nil
	}

	// Profile doesn't exist - create one
	// If domain and category provided, use them
	if domain != "" && category != "" {
		profile = &models.BrandProfile{
			ID:          uuid.New().String(),
			BrandName:   brand,
			Domain:      domain,
			Category:    category,
			Description: description,
		}

		if err := s.db.CreateBrandProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("failed to create brand profile: %w", err)
		}

		return profile, nil
	}

	// Otherwise, derive domain/category using LLM
	derivedDomain, derivedCategory, err := s.deriveBrandMetadata(ctx, brand, description)
	if err != nil {
		// Fallback to defaults if derivation fails
		derivedDomain = "general"
		derivedCategory = "general"
	}

	profile = &models.BrandProfile{
		ID:          uuid.New().String(),
		BrandName:   brand,
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
func (s *PromptGenerationService) deriveBrandMetadata(ctx context.Context, brand, description string) (string, string, error) {
	// Get an LLM for metadata derivation
	provider, ok := s.llmRegistry.Get("google")
	if !ok {
		providers := s.llmRegistry.List()
		if len(providers) == 0 {
			return "", "", fmt.Errorf("no LLM providers available")
		}
		provider, _ = s.llmRegistry.Get(providers[0])
	}

	descText := ""
	if description != "" {
		descText = fmt.Sprintf("\nDescription: %s", description)
	}

	derivationPrompt := fmt.Sprintf(`Analyze this brand and provide its domain/industry and specific category.

Brand: %s%s

Respond in EXACTLY this format (one line for domain, one for category):
Domain: <industry domain like "technology", "healthcare", "finance", "retail", etc>
Category: <specific category like "AI SEO Tools", "CRM Software", "Cloud Storage", etc>

Example:
Domain: technology
Category: AI SEO Tools`, brand, descText)

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

	return domain, category, nil
}

// getPromptsFromLibrary retrieves prompts from a library
func (s *PromptGenerationService) getPromptsFromLibrary(ctx context.Context, library *models.PromptLibrary, count int) ([]models.Prompt, error) {
	var prompts []models.Prompt

	// Get up to 'count' prompts from the library
	limit := count
	if limit > len(library.PromptIDs) {
		limit = len(library.PromptIDs)
	}

	for i := 0; i < limit; i++ {
		prompt, err := s.db.GetPrompt(ctx, library.PromptIDs[i])
		if err != nil {
			continue // Skip if prompt not found
		}
		prompts = append(prompts, *prompt)
	}

	return prompts, nil
}

// generateNewPrompts generates new prompts using an LLM
func (s *PromptGenerationService) generateNewPrompts(ctx context.Context, brand, category, domain, description string, count int, existingPrompts []models.Prompt) ([]string, error) {
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

	generationPrompt := fmt.Sprintf(`Generate %d unique, natural questions that people would ask when searching for products/services in this space.

%s%s

REQUIREMENTS:
1. Questions should be diverse and natural (how people actually search)
2. Mix different query types: comparisons, recommendations, how-to, best practices, reviews
3. Questions should be likely to generate responses mentioning brands in this space
4. Avoid duplicating existing prompts
5. Each question should be on a new line
6. DO NOT include numbers, bullets, or any formatting - just the questions

Examples of good formats:
- "What are the best tools for..."
- "How do I choose between..."
- "Which platform is better for..."
- "Can you recommend..."
- "What's the difference between..."
- "How does X compare to Y..."

Generate exactly %d questions, one per line:`, count, brandInfo, existingText, count)

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

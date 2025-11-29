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

	// Step 1: Check for existing prompts in the same category/domain
	existingPrompts, err := s.findReusablePrompts(ctx, category, domain, count)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to find reusable prompts: %w", err)
	}

	existingCount := len(existingPrompts)
	needToGenerate := count - existingCount

	// If we have enough existing prompts, return them
	if needToGenerate <= 0 {
		return existingPrompts[:count], count, 0, nil
	}

	// Step 2: Generate new prompts using LLM
	newPrompts, err := s.generateNewPrompts(ctx, brand, category, domain, description, needToGenerate, existingPrompts)
	if err != nil {
		// If generation fails but we have some existing prompts, return those
		if existingCount > 0 {
			return existingPrompts, existingCount, 0, nil
		}
		return nil, 0, 0, fmt.Errorf("failed to generate prompts: %w", err)
	}

	// Step 3: Save new prompts to database
	savedPrompts, err := s.savePrompts(ctx, newPrompts, brand, category, domain)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to save prompts: %w", err)
	}

	// Combine existing and new prompts
	allPrompts := append(existingPrompts, savedPrompts...)

	return allPrompts, existingCount, len(savedPrompts), nil
}

// findReusablePrompts finds existing prompts that can be reused
func (s *PromptGenerationService) findReusablePrompts(ctx context.Context, category, domain string, limit int) ([]models.Prompt, error) {
	// Get all enabled prompts
	allPrompts, err := s.db.ListPrompts(ctx, nil)
	if err != nil {
		return nil, err
	}

	var matchingPrompts []models.Prompt
	for _, prompt := range allPrompts {
		if !prompt.Enabled {
			continue
		}

		// Match by category (exact match)
		if category != "" && prompt.Category == category {
			matchingPrompts = append(matchingPrompts, *prompt)
			if len(matchingPrompts) >= limit {
				break
			}
			continue
		}

		// Match by domain (if no category match)
		if domain != "" && prompt.Domain == domain && category == "" {
			matchingPrompts = append(matchingPrompts, *prompt)
			if len(matchingPrompts) >= limit {
				break
			}
		}
	}

	return matchingPrompts, nil
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

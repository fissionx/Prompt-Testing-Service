package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/AI2HU/gego/internal/db"
	"github.com/AI2HU/gego/internal/llm"
	"github.com/AI2HU/gego/internal/models"
)

// BulkExecutionService handles batch execution of prompts across multiple LLMs
type BulkExecutionService struct {
	db          db.Database
	llmRegistry *llm.Registry
}

// NewBulkExecutionService creates a new bulk execution service
func NewBulkExecutionService(database db.Database, registry *llm.Registry) *BulkExecutionService {
	return &BulkExecutionService{
		db:          database,
		llmRegistry: registry,
	}
}

// ExecuteCampaign executes all prompts across all LLMs for a GEO campaign
func (s *BulkExecutionService) ExecuteCampaign(ctx context.Context, campaignName, brand string, promptIDs, llmIDs []string, temperature float64) (*models.GEOCampaign, error) {
	if temperature == 0 {
		temperature = 0.7
	}

	// Create campaign
	campaign := &models.GEOCampaign{
		ID:        uuid.New().String(),
		Name:      campaignName,
		Brand:     brand,
		PromptIDs: promptIDs,
		LLMIDs:    llmIDs,
		Status:    "running",
		TotalRuns: len(promptIDs) * len(llmIDs),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start execution in background
	go s.executeInBackground(context.Background(), campaign, temperature)

	return campaign, nil
}

// executeInBackground runs the campaign execution asynchronously
func (s *BulkExecutionService) executeInBackground(ctx context.Context, campaign *models.GEOCampaign, temperature float64) {
	log.Printf("========== STARTING CAMPAIGN: %s ==========", campaign.Name)
	log.Printf("Brand: %s, Prompts: %d, LLMs: %d, Total Runs: %d", 
		campaign.Brand, len(campaign.PromptIDs), len(campaign.LLMIDs), campaign.TotalRuns)

	// Fetch prompts and LLMs
	prompts, err := s.getPrompts(ctx, campaign.PromptIDs)
	if err != nil {
		log.Printf("Failed to fetch prompts: %v", err)
		campaign.Status = "failed"
		return
	}

	llms, err := s.getLLMs(ctx, campaign.LLMIDs)
	if err != nil {
		log.Printf("Failed to fetch LLMs: %v", err)
		campaign.Status = "failed"
		return
	}

	// Execute with concurrency control
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Max 3 concurrent executions
	completed := 0
	mu := sync.Mutex{}

	for _, prompt := range prompts {
		for _, llmConfig := range llms {
			wg.Add(1)
			
			go func(p *models.Prompt, llm *models.LLMConfig) {
				defer wg.Done()
				
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Execute single prompt-LLM pair
				err := s.executeSingle(ctx, p, llm, campaign.Brand, temperature)
				
				mu.Lock()
				completed++
				if completed%10 == 0 || completed == campaign.TotalRuns {
					log.Printf("Campaign %s: %d/%d completed", campaign.Name, completed, campaign.TotalRuns)
				}
				mu.Unlock()

				if err != nil {
					log.Printf("Execution failed for prompt %s with LLM %s: %v", p.ID, llm.ID, err)
				}
			}(prompt, llmConfig)
		}
	}

	wg.Wait()

	completedTime := time.Now()
	campaign.Status = "completed"
	campaign.CompletedAt = &completedTime
	campaign.UpdatedAt = completedTime

	log.Printf("========== CAMPAIGN COMPLETED: %s ==========", campaign.Name)
	log.Printf("Total executions: %d", completed)
}

// executeSingle executes a single prompt with a single LLM
func (s *BulkExecutionService) executeSingle(ctx context.Context, prompt *models.Prompt, llmConfig *models.LLMConfig, brand string, temperature float64) error {
	// Create LLM provider
	provider, ok := s.llmRegistry.Get(llmConfig.Provider)
	if !ok || provider == nil {
		return fmt.Errorf("provider not available: %s", llmConfig.Provider)
	}

	// Execute prompt
	response, err := provider.Generate(ctx, prompt.Template, llm.Config{
		Model:       llmConfig.Model,
		Temperature: temperature,
		MaxTokens:   4096,
		Brand:       brand,
	})
	if err != nil {
		// Save error response
		errorResponse := &models.Response{
			ID:           uuid.New().String(),
			PromptID:     prompt.ID,
			PromptText:   prompt.Template,
			LLMID:        llmConfig.ID,
			LLMName:      llmConfig.Name,
			LLMProvider:  llmConfig.Provider,
			LLMModel:     llmConfig.Model,
			Brand:        brand,
			Temperature:  temperature,
			Error:        err.Error(),
			CreatedAt:    time.Now(),
		}
		s.db.CreateResponse(ctx, errorResponse)
		return err
	}

	// Parse GEO analysis from response (if brand was provided)
	responseModel := &models.Response{
		ID:           uuid.New().String(),
		PromptID:     prompt.ID,
		PromptText:   prompt.Template,
		LLMID:        llmConfig.ID,
		LLMName:      llmConfig.Name,
		LLMProvider:  llmConfig.Provider,
		LLMModel:     llmConfig.Model,
		ResponseText: response.Text,
		Brand:        brand,
		Temperature:  temperature,
		TokensUsed:   response.TokensUsed,
		LatencyMs:    response.LatencyMs,
		CreatedAt:    time.Now(),
	}

	// Parse GEO metrics if brand was provided
	if brand != "" {
		geoAnalysis := parseGEOAnalysis(response.Text)
		if geoAnalysis != nil {
			// Access nested GEOAnalysis struct
			geo := geoAnalysis.GEOAnalysis
			responseModel.VisibilityScore = geo.VisibilityScore
			responseModel.BrandMentioned = geo.BrandMentioned
			responseModel.InGroundingSources = geo.InGroundingSources
			responseModel.Sentiment = geo.Sentiment
			responseModel.CompetitorsMention = geo.Competitors
			responseModel.GroundingSources = response.GroundingSources
			
			// Extract position/ranking from the search_answer text
			searchAnswer := geoAnalysis.SearchAnswer
			if searchAnswer == "" {
				searchAnswer = response.Text
			}
			
			if geo.BrandMentioned {
				position, totalBrands := ExtractBrandPosition(searchAnswer, brand)
				responseModel.BrandPosition = position
				responseModel.TotalBrandsListed = totalBrands
			}
			
			// Extract domains from grounding sources
			if len(response.GroundingSources) > 0 {
				responseModel.GroundingDomains = ExtractDomainsFromSources(response.GroundingSources)
			}
		}
	}
	
	// Add time-series fields
	now := time.Now()
	responseModel.Week = now.Format("2006-W02")
	responseModel.Month = now.Format("2006-01")
	quarter := (int(now.Month()) - 1) / 3 + 1
	responseModel.Quarter = fmt.Sprintf("%d-Q%d", now.Year(), quarter)

	// Save response
	return s.db.CreateResponse(ctx, responseModel)
}

// getPrompts fetches prompts by IDs
func (s *BulkExecutionService) getPrompts(ctx context.Context, promptIDs []string) ([]*models.Prompt, error) {
	var prompts []*models.Prompt
	for _, id := range promptIDs {
		prompt, err := s.db.GetPrompt(ctx, id)
		if err != nil {
			continue
		}
		prompts = append(prompts, prompt)
	}
	if len(prompts) == 0 {
		return nil, fmt.Errorf("no valid prompts found")
	}
	return prompts, nil
}

// getLLMs fetches LLM configs by IDs
func (s *BulkExecutionService) getLLMs(ctx context.Context, llmIDs []string) ([]*models.LLMConfig, error) {
	llmService := NewLLMService(s.db)
	var llms []*models.LLMConfig
	for _, id := range llmIDs {
		llmConfig, err := llmService.GetLLM(ctx, id)
		if err != nil {
			continue
		}
		if llmConfig.Enabled {
			llms = append(llms, llmConfig)
		}
	}
	if len(llms) == 0 {
		return nil, fmt.Errorf("no valid enabled LLMs found")
	}
	return llms, nil
}

// GEOAnalysisResult represents the parsed GEO analysis from LLM
type GEOAnalysisResult struct {
	SearchAnswer string `json:"search_answer"`
	GEOAnalysis  struct {
		VisibilityScore    int      `json:"visibility_score"`
		BrandMentioned     bool     `json:"brand_mentioned"`
		InGroundingSources bool     `json:"in_grounding_sources"`
		MentionStatus      string   `json:"mention_status"`
		Reason             string   `json:"reason"`
		Sentiment          string   `json:"sentiment"`
		Competitors        []string `json:"competitors"`
		Insights           []string `json:"insights"`
		Actions            []string `json:"actions"`
		CompetitorInfo     string   `json:"competitor_info"`
	} `json:"geo_analysis"`
}

// parseGEOAnalysis extracts and parses JSON from the LLM response
func parseGEOAnalysis(text string) *GEOAnalysisResult {
	// Clean up the response - remove markdown code blocks if present
	cleanedText := strings.TrimSpace(text)
	
	// Remove markdown code block wrappers (```json ... ``` or ``` ... ```)
	jsonBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	if matches := jsonBlockRegex.FindStringSubmatch(cleanedText); len(matches) > 1 {
		cleanedText = strings.TrimSpace(matches[1])
	} else {
		// Try simple prefix/suffix removal
		cleanedText = strings.TrimPrefix(cleanedText, "```json")
		cleanedText = strings.TrimPrefix(cleanedText, "```")
		cleanedText = strings.TrimSuffix(cleanedText, "```")
		cleanedText = strings.TrimSpace(cleanedText)
	}

	// Try to find JSON object in the text if it's mixed with other content
	if !strings.HasPrefix(cleanedText, "{") {
		jsonStartIdx := strings.Index(cleanedText, "{")
		jsonEndIdx := strings.LastIndex(cleanedText, "}")
		if jsonStartIdx != -1 && jsonEndIdx != -1 && jsonEndIdx > jsonStartIdx {
			cleanedText = cleanedText[jsonStartIdx : jsonEndIdx+1]
		}
	}
	
	var result GEOAnalysisResult
	if err := json.Unmarshal([]byte(cleanedText), &result); err != nil {
		log.Printf("❌ Failed to parse GEO analysis JSON: %v", err)
		log.Printf("Cleaned text (first 500 chars): %s", truncateForLog(cleanedText, 500))
		return nil
	}
	
	log.Printf("✅ Parsed GEO: Score=%d, Mentioned=%v, Sentiment=%s, Competitors=%v", 
		result.GEOAnalysis.VisibilityScore, 
		result.GEOAnalysis.BrandMentioned,
		result.GEOAnalysis.Sentiment,
		result.GEOAnalysis.Competitors)
	
	return &result
}

// truncateForLog truncates a string for logging
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

